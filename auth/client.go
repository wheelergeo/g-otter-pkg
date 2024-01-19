package auth

import (
	"github.com/casbin/casbin/v2"
	redisadapter "github.com/casbin/redis-adapter/v3"
)

type AuthClient struct {
	e            *casbin.Enforcer
	userPrefix   string
	rolePrefix   string
	policyPrefix string
	action       string
}

var c *AuthClient

func NewClient(modelPath string, policy_redis string) {
	once.Do(func() {
		c = new(AuthClient)
		c.userPrefix = "u_"
		c.rolePrefix = "r_"
		c.policyPrefix = "p_"
		c.action = "Allow"

		r, err := redisadapter.NewAdapter("tcp", policy_redis)
		if err != nil {
			panic(err)
		}
		c.e, err = casbin.NewEnforcer(modelPath, r)
		if err != nil {
			panic(err)
		}
	})
}

func Client() *AuthClient {
	return c
}

func (a *AuthClient) GetRolesForUser(user string) (roles []string, err error) {
	user = a.userPrefix + user
	roles, err = a.e.GetRolesForUser(user)
	prefixLen := len(a.rolePrefix)
	if err != nil {
		return
	}

	for k, v := range roles {
		roles[k] = v[prefixLen:]
	}
	return
}

func (a *AuthClient) GetPoliciesForRole(role string) (objs []string) {
	prefixLen := len(a.policyPrefix)
	role = a.rolePrefix + role

	policies := a.e.GetFilteredPolicy(0, role)
	if len(policies) == 0 {
		return
	}
	for _, v := range policies {
		objs = append(objs, v[1][prefixLen:])
	}
	return
}

func (a *AuthClient) GetPoliciesForUser(user string) (objs []string) {
	roles, err := a.GetRolesForUser(user)
	if err != nil || len(roles) == 0 {
		return
	}

	for _, v := range roles {
		objs = append(objs, a.GetPoliciesForRole(v)...)
	}

	return
}

func (a *AuthClient) Enforce(user string, obj string) (b bool) {
	user = a.userPrefix + user
	obj = a.policyPrefix + obj

	b, _ = a.e.Enforce(user, obj, a.action)
	return
}
