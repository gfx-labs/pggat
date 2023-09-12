package gsql

import (
	"net"
	"testing"

	"tuxpa.in/a/zlog/log"

	"pggat/lib/auth/credentials"
	"pggat/lib/bouncer/backends/v0"
	"pggat/lib/bouncer/bouncers/v2"
	"pggat/lib/fed"
)

type Result struct {
	Username string  `sql:"0"`
	Password *string `sql:"1"`
}

func TestQuery(t *testing.T) {
	// open server
	s, err := net.Dial("tcp", "localhost:5432")
	if err != nil {
		t.Error(err)
		return
	}
	server := fed.WrapNetConn(s)
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
	client := new(Client)
	err = ExtendedQuery(client, &res, "SELECT usename, passwd FROM pg_shadow WHERE usename=$1", "bob")
	if err != nil {
		t.Error(err)
		return
	}
	err = client.Close()
	if err != nil {
		t.Error(err)
	}

	initial, err := client.ReadPacket(true)
	if err != nil {
		t.Error(err)
	}
	clientErr, serverErr := bouncers.Bounce(client, server, initial)
	if clientErr != nil {
		t.Error(clientErr)
	}
	if serverErr != nil {
		t.Error(serverErr)
	}

	log.Printf("%#v", res)
}
