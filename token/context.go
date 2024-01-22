package token

import (
	"context"
)

type TokenContext struct {
	UserCtx *UserContext
}

type UserContext struct {
	Id    int64
	UUID  int64
	Phone string
	Dept  int32
	Post  int32
}

const TokenCtxKey = "TokenCtx"

func NewClaimByUserContext(u UserContext) ClaimData {
	return ClaimData{
		"Id":    u.Id,
		"UUID":  u.UUID,
		"Phone": u.Phone,
		"Dept":  u.Dept,
		"Post":  u.Post,
	}
}

func ParseClaimAsUserContext(c ClaimData) UserContext {
	return UserContext{
		Id:    int64(c["Id"].(float64)),
		UUID:  int64(c["UUID"].(float64)),
		Phone: c["Phone"].(string),
		Dept:  int32(c["Dept"].(float64)),
		Post:  int32(c["Post"].(float64)),
	}
}

func SetContext(ctx *context.Context, content *TokenContext) {
	*ctx = context.WithValue(*ctx, TokenCtxKey, content)
}

func GetContext(ctx context.Context) *TokenContext {
	v := ctx.Value(TokenCtxKey)
	if v == nil {
		return &TokenContext{}
	}

	tokenCtx, ok := v.(*TokenContext)
	if ok {
		return tokenCtx
	}
	return &TokenContext{}
}
