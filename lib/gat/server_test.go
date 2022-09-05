package gat

import (
	"context"
	"git.tuxpa.in/a/zlog/log"
	"testing"

	"gfx.cafe/gfx/pggat/lib/config"
)

var test_address = "localhost:5444"

var test_user = config.User{
	Name:             "postgres",
	Password:         "test",
	PoolSize:         4,
	StatementTimeout: 250,
}

func TestServerDial(t *testing.T) {
	csm := make(map[ClientInfo]ClientInfo)
	srv, err := DialServer(context.TODO(), test_address, &test_user, "postgres", csm, nil)
	if err != nil {
		t.Error(err)
	}
	log.Println(srv)
}
