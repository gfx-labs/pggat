package bouncers

import (
	"gfx.cafe/gfx/pggat/lib/bouncer/backends/v0"
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/perror"
)

func clientFail(client *fed.Conn, err perror.Error) {
	// send fatal error to client
	resp := perror.ToPacket(err)
	_ = client.WritePacket(resp)
}

func Bounce(client, server *fed.Conn, initialPacket fed.Packet) (clientError error, serverError error) {
	serverError, clientError = backends.Transaction(server, client, initialPacket)

	if clientError != nil {
		clientFail(client, perror.Wrap(clientError))
	} else if serverError != nil {
		clientFail(client, perror.Wrap(serverError))
	}

	return
}
