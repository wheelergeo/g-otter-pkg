package token

import (
	"context"
	"encoding"
	"errors"
	"reflect"
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/bytedance/sonic"
	"github.com/redis/go-redis/v9"
)

var _ encoding.BinaryMarshaler = new(TokenValue)
var _ encoding.BinaryUnmarshaler = new(TokenValue)

type Callback func(*TokenValue, ClaimData, time.Time)

type STokenAuth struct {
	redis     *redis.Client
	cacheKey  string
	parser    paseto.Parser
	onRefresh Callback
}

type ClaimData map[string]interface{}

type TokenValue struct {
	Key           [32]byte  `json:"key"`
	Authorization string    `json:"authorization"`
	Refresh       int       `json:"refresh"`
	Timeout       int       `json:"timeout"`
	IssuedAt      time.Time `json:"issueAt"`
}

var sTokenAuth *STokenAuth

func Init(redis *redis.Client, cacheKey string, onRefresh Callback) {
	sTokenAuth = &STokenAuth{
		redis:     redis,
		cacheKey:  cacheKey,
		parser:    paseto.NewParser(),
		onRefresh: onRefresh,
	}
	sonic.Pretouch(reflect.TypeOf(TokenValue{}))
}

func TokenAuth() *STokenAuth {
	return sTokenAuth
}

func (m TokenValue) MarshalBinary() (data []byte, err error) {
	return sonic.Marshal(&m)
}

func (m *TokenValue) UnmarshalBinary(data []byte) error {
	return sonic.Unmarshal(data, m)
}

func (ta *STokenAuth) New(refresh, timeout int,
	data ClaimData) (authorization string, expiredAt time.Time, err error) {

	duration := time.Duration(timeout * int(time.Second))
	tokenValue, pToken, err := ta.newToken(refresh, timeout, data)
	if err != nil {
		return
	}

	authorization = tokenValue.Authorization
	expiredAt, _ = pToken.GetExpiration()
	err = ta.redis.Set(
		context.Background(),
		ta.cacheKey+":"+tokenValue.Authorization,
		*tokenValue,
		duration,
	).Err()

	return
}

func (ta *STokenAuth) newToken(refresh, timeout int,
	data ClaimData) (tokenValue *TokenValue, pToken *paseto.Token, err error) {

	t := time.Now()
	key := paseto.NewV4SymmetricKey()
	duration := time.Duration(timeout * int(time.Second))
	tokenValue = &TokenValue{
		Refresh:  refresh,
		Timeout:  timeout,
		IssuedAt: t,
		Key:      [32]byte(key.ExportBytes()),
	}

	pt := paseto.NewToken()
	pToken = &pt
	pToken.SetIssuedAt(t)
	pToken.SetNotBefore(t)
	pToken.SetExpiration(t.Add(duration))

	for k, v := range data {
		if err = pToken.Set(k, v); err != nil {
			return
		}
	}

	tokenValue.Authorization = pToken.V4Encrypt(key, nil)

	return
}

func (ta *STokenAuth) IsEffective(authorization string) bool {
	cmd := ta.redis.Get(context.Background(), ta.cacheKey+":"+authorization)
	if cmd.Err() == redis.Nil {
		return false
	}
	return true
}

func (ta *STokenAuth) Parse(authorization string) (data ClaimData, err error) {
	// 1. get token from redis
	cmd := ta.redis.Get(context.Background(), ta.cacheKey+":"+authorization)
	if cmd.Err() == redis.Nil {
		err = errors.New("Token is expired")
		return
	}

	tokenValue := &TokenValue{}
	err = cmd.Scan(tokenValue)
	if err != nil {
		return
	}

	// 2. parse token
	key, _ := paseto.V4SymmetricKeyFromBytes(tokenValue.Key[:])
	pToken, err := ta.parser.ParseV4Local(
		key,
		tokenValue.Authorization,
		nil,
	)
	if err != nil {
		return
	}
	data = pToken.Claims()

	// 3. judge token is need to be refresh or not
	t := time.Now()
	tRefresh := tokenValue.IssuedAt.Add(time.Second * time.Duration(tokenValue.Refresh))
	tTimeout := tokenValue.IssuedAt.Add(time.Second * time.Duration(tokenValue.Timeout))
	if t.After(tRefresh) && t.Before(tTimeout) {
		go ta.refresh(tokenValue, pToken)
	}
	return
}

func (ta *STokenAuth) refresh(oldTokenValue *TokenValue,
	oldPToken *paseto.Token) (newPToken *paseto.Token, err error) {

	duration := time.Duration(oldTokenValue.Timeout * int(time.Second))

	// 1. gen new token
	dataClaims := oldPToken.Claims()
	newTokenValue, newPToken, err := ta.newToken(
		oldTokenValue.Refresh,
		oldTokenValue.Timeout,
		dataClaims,
	)
	if err != nil {
		return
	}

	// 2. replace token
	err = ta.redis.Set(
		context.Background(),
		ta.cacheKey+":"+oldTokenValue.Authorization,
		*newTokenValue,
		duration,
	).Err()

	if err == nil && ta.onRefresh != nil {
		expiredAt, _ := newPToken.GetExpiration()
		ta.onRefresh(newTokenValue, dataClaims, expiredAt)
	}
	return
}

func (ta *STokenAuth) Delete(authorization string) (err error) {
	return ta.redis.Del(context.Background(), ta.cacheKey+":"+authorization).Err()
}
