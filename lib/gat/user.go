package gat

import "pggat2/lib/util/maps"

type User struct {
	password string

	pools maps.RWLocked[string, *Pool]
}

func NewUser(password string) *User {
	return &User{
		password: password,
	}
}

func (T *User) GetPassword() string {
	return T.password
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
