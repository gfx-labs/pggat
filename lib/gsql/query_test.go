package gsql

import (
	"net"
	"testing"

	"tuxpa.in/a/zlog/log"

	"pggat2/lib/auth/credentials"
	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/bouncer/bouncers/v2"
	"pggat2/lib/fed"
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
	err = client.ExtendedQuery(&res, "SELECT $1 as usename, $2 as passwd", "username", "test")
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
