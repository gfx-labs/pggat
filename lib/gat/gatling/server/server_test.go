package server

import (
	"context"
	"git.tuxpa.in/a/zlog/log"
	"testing"

	"gfx.cafe/gfx/pggat/lib/config"
)

var test_server = config.Server{
	Host: "localhost",
	Port: 5432,
}

var test_shard = config.Shard{}

var test_user = config.User{
	Name:             "postgres",
	Password:         "password",
	PoolSize:         4,
	StatementTimeout: 250,
}

func TestServerDial(t *testing.T) {
	srv, err := Dial(context.TODO(), &test_user, &test_shard, &test_server)
	if err != nil {
		t.Error(err)
	}
	log.Println(srv)
}
