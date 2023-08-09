package gat

import (
	"pggat2/lib/auth"
	"pggat2/lib/util/maps"
)

type User struct {
	credentials auth.Credentials

	pools maps.RWLocked[string, *Pool]
}

func NewUser(credentials auth.Credentials) *User {
	return &User{
		credentials: credentials,
	}
}

func (T *User) GetCredentials() auth.Credentials {
	return T.credentials
}

func (T *User) AddPool(name string, pool *Pool) {
	T.pools.Store(name, pool)
}

func (T *User) RemovePool(name string) {
	T.pools.Delete(name)
}

func (T *User) GetPool(name string) *Pool {
	pool, _ := T.pools.Load(name)
	return pool
}
