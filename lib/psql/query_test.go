package psql

import (
	"net"
	"testing"

	"pggat2/lib/auth/credentials"
	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/zap"
)

func TestQuery(t *testing.T) {
	// open server
	s, err := net.Dial("tcp", "localhost:5432")
	if err != nil {
		t.Error(err)
		return
	}
	server := zap.WrapNetConn(s)
	_, err = backends.Accept(server, backends.AcceptOptions{
		Credentials: credentials.Cleartext{
			Username: "postgres",
			Password: "password",
		},
		Database: "postgres",
	})
	if err != nil {
		t.Error(err)
		return
	}

	err = Query(server, "SELECT usename, passwd FROM pg_shadow WHERE usename='postgres'")
	if err != nil {
		t.Error(err)
		return
	}
}
