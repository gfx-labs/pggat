package server

import (
	"context"
	"git.tuxpa.in/a/zlog/log"
	"testing"

	"gfx.cafe/gfx/pggat/lib/config"
)

var test_address = "localhost:5432"

var test_user = config.User{
	Name:             "postgres",
	Password:         "password",
	PoolSize:         4,
	StatementTimeout: 250,
}

func TestServerDial(t *testing.T) {
	srv, err := Dial(context.TODO(), test_address, &test_user, "postgres", test_user.Name, test_user.Password, nil)
	if err != nil {
		t.Error(err)
	}
	log.Println(srv)
}
