package gsql_test

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"testing"

	"gfx.cafe/gfx/pggat/lib/auth/credentials"
	"gfx.cafe/gfx/pggat/lib/bouncer/backends/v0"
	"gfx.cafe/gfx/pggat/lib/bouncer/bouncers/v2"
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/fed/codecs/netconncodec"
	"gfx.cafe/gfx/pggat/lib/gsql"
	"gfx.cafe/gfx/pggat/lib/util/flip"
)

type Result struct {
	Username string  `sql:"0"`
	Password *string `sql:"1"`
}

func TestQuery(t *testing.T) {
	go func() {
		panic(http.ListenAndServe(":8080", nil))
	}()

	s, err := net.Dial("tcp", "localhost:5432")
	if err != nil {
		t.Error(err)
		return
	}
	server := fed.NewConn(context.Background(), netconncodec.NewCodec(s))
	err = backends.Accept(
		server,
		"disable",
		&tls.Config{},
		"postgres",
		credentials.Cleartext{
			Username: "postgres",
			Password: "postgres",
		},
		"postgres",
		nil,
	)
	if err != nil {
		t.Error(err)
		return
	}

	inward, outward, _, _ := gsql.NewPair()

	var res Result

	var b flip.Bank
	b.Queue(func() error {
		if err := gsql.ExtendedQuery(inward, &res, "SELECT usename, passwd FROM pg_shadow WHERE usename=$1", "postgres"); err != nil {
			return err
		}
		return nil
	})

	b.Queue(func() error {
		initial, err := outward.ReadPacket(true)
		if err != nil {
			return err
		}
		clientErr, serverErr := bouncers.Bounce(outward, server, initial)
		if clientErr != nil {
			return clientErr
		}
		if serverErr != nil {
			return serverErr
		}
		if err := outward.Close(); err != nil {
			return err
		}
		return nil
	})

	if err = b.Wait(); err != nil {
		t.Error(err)
	}

	log.Printf("%#v", res)
}
