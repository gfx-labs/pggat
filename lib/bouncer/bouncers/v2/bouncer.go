package bouncers

import (
	"context"
	"gfx.cafe/gfx/pggat/lib/bouncer/backends/v0"
	"gfx.cafe/gfx/pggat/lib/fed"
)

func Bounce(ctx context.Context, client, server *fed.Conn, initialPacket fed.Packet) (clientError error, serverError error) {
	serverError, clientError = backends.Transaction(ctx, server, client, initialPacket)
	return
}
