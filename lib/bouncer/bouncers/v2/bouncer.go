package bouncers

import (
	"gfx.cafe/gfx/pggat/lib/bouncer/backends/v0"
	"gfx.cafe/gfx/pggat/lib/fed"
)

func Bounce(client, server *fed.Conn, initialPacket fed.Packet) (clientError error, serverError error) {
	serverError, clientError = backends.Transaction(server, client, initialPacket)
	return
}
