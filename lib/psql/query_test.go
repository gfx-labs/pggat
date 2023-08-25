package psql

import (
	"net"
	"testing"

	"tuxpa.in/a/zlog/log"

	"pggat2/lib/auth/credentials"
	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/zap"
)

type Result struct {
	Username string  `sql:"usename"`
	Password *string `sql:"passwd"`
}

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

	var res Result

	err = Query(server, &res, "SELECT $1 as usename, $2 as passwd", "postgres", "password")
	if err != nil {
		t.Error(err)
		return
	}

	log.Printf("%#v", res)
}
