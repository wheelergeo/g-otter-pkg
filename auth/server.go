package auth

import (
	"sync"

	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	redisadapter "github.com/casbin/redis-adapter/v3"
	"gorm.io/gorm"
)

type AuthServer struct {
	e            [2]*casbin.Enforcer
	userPrefix   string
	rolePrefix   string
	policyPrefix string
	action       string
}

var once sync.Once
var s *AuthServer

func NewServer(modelPath string, policy_db *gorm.DB,
	policy_table string, policy_redis string) {

	once.Do(func() {
		s = new(AuthServer)
		s.userPrefix = "u_"
		s.rolePrefix = "r_"
		s.policyPrefix = "p_"
		s.action = "Allow"
		d, err := gormadapter.NewAdapterByDBUseTableName(
			policy_db, "", policy_table)
		if err != nil {
			panic(err)
		}

		s.e[0], err = casbin.NewEnforcer(modelPath, d)
		if err != nil {
			panic(err)
		}

		r, err := redisadapter.NewAdapter("tcp", policy_redis)
		if err != nil {
			panic(err)
		}
		s.e[1], err = casbin.NewEnforcer(modelPath, r)
		if err != nil {
			panic(err)
		}

		// Synchronize data between redis and mysql
		s.Sync()
	})
}

func Server() *AuthServer {
	return s
}

func (a *AuthServer) Sync() {
	// 1. Clear redis policy
	a.e[1].LoadPolicy()
	roles := a.e[1].GetAllRoles()
	for _, v := range roles {
		posts, err := a.e[1].GetUsersForRole(v)
		if err != nil {
			panic(err)
		}
		for _, v1 := range posts {
			_, err = a.e[1].DeleteRoleForUser(v1, v)
			if err != nil {
				panic(err)
			}
		}
	}
	a.e[1].RemovePolicies(a.e[1].GetPolicy())

	// 2. Synchronize roles
	roles = a.e[0].GetAllRoles()
	for _, v := range roles {
		posts, err := a.e[0].GetUsersForRole(v)
		if err != nil {
			panic(err)
		}
		for _, v1 := range posts {
			_, err = a.e[1].AddRoleForUser(v1, v)
			if err != nil {
				panic(err)
			}
		}
	}

	// 3. Synchronize policies
	a.e[1].LoadPolicy()
	policies := a.e[0].GetPolicy()
	if len(policies) == 0 {
		return
	}
	_, err := a.e[1].AddPolicies(policies)
	if err != nil {
		panic(err)
	}
}

func (a *AuthServer) AddRolesForUser(user string, roles []string) (err error) {
	for k, v := range roles {
		roles[k] = a.rolePrefix + v
	}

	for _, v := range a.e {
		_, err = v.AddRolesForUser(a.userPrefix+user, roles)
		if err != nil {
			a.Sync()
			return
		}
	}
	return
}

func (a *AuthServer) DeleteRolesForUser(user string, roles []string) (err error) {
	for k, v := range roles {
		roles[k] = a.rolePrefix + v
	}

	for _, v := range a.e {
		for _, v1 := range roles {
			_, err = v.DeleteRoleForUser(a.userPrefix+user, v1)
			if err != nil {
				a.Sync()
				return
			}
		}
	}
	return
}

func (a *AuthServer) GetRolesForUser(user string) (roles []string, err error) {
	user = a.userPrefix + user
	roles, err = a.e[1].GetRolesForUser(user)
	prefixLen := len(a.rolePrefix)
	if err != nil {
		roles, err = a.e[0].GetRolesForUser(user)
		a.Sync()
	}

	for k, v := range roles {
		roles[k] = v[prefixLen:]
	}
	return
}

func (a *AuthServer) AddPoliciesForRole(role string, objs []string) (err error) {
	var policies [][]string
	for _, v := range objs {
		policies = append(policies, []string{
			a.rolePrefix + role,
			a.policyPrefix + v,
			a.action,
		})
	}
	for _, v := range a.e {
		_, err = v.AddPolicies(policies)
		if err != nil {
			a.Sync()
			return
		}
	}
	return
}

func (a *AuthServer) DeletePoliciesForRole(role string, objs []string) (err error) {
	var policies [][]string
	for _, v := range objs {
		policies = append(policies, []string{
			a.rolePrefix + role,
			a.policyPrefix + v,
			a.action,
		})
	}
	for _, v := range a.e {
		_, err = v.RemovePolicies(policies)
		if err != nil {
			a.Sync()
			return
		}
	}
	return
}

func (a *AuthServer) GetPoliciesForRole(role string) (objs []string) {
	prefixLen := len(a.policyPrefix)
	role = a.rolePrefix + role

	policies := a.e[1].GetFilteredPolicy(0, role)
	if len(policies) == 0 {
		policies = a.e[0].GetFilteredPolicy(0, role)
		a.Sync()
	}
	for _, v := range policies {
		objs = append(objs, v[1][prefixLen:])
	}
	return
}

func (a *AuthServer) GetPoliciesForUser(user string) (objs []string) {
	roles, err := a.GetRolesForUser(user)
	if err != nil || len(roles) == 0 {
		return
	}

	for _, v := range roles {
		objs = append(objs, a.GetPoliciesForRole(v)...)
	}

	return
}

func (a *AuthServer) Enforce(user string, obj string) (b bool) {
	user = a.userPrefix + user
	obj = a.policyPrefix + obj

	b, err := a.e[1].Enforce(user, obj, a.action)
	if err != nil {
		b, _ = a.e[0].Enforce(user, obj, a.action)
		return
	}
	return
}
