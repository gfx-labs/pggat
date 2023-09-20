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
	ctx := backends.AcceptContext{
		Conn: server,
		Options: backends.AcceptOptions{
			Username: "postgres",
			Credentials: credentials.Cleartext{
				Username: "postgres",
				Password: "password",
			},
			Database: "postgres",
		},
	}
	_, err = backends.Accept(&ctx)
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

	var initial fed.Packet
	initial, err = client.ReadPacket(true, initial)
	if err != nil {
		t.Error(err)
	}
	_, clientErr, serverErr := bouncers.Bounce(client, server, initial)
	if clientErr != nil {
		t.Error(clientErr)
	}
	if serverErr != nil {
		t.Error(serverErr)
	}

	log.Printf("%#v", res)
}
