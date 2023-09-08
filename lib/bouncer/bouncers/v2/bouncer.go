package bouncers

import (
	"pggat/lib/bouncer/backends/v0"
	"pggat/lib/fed"
	packets "pggat/lib/fed/packets/v3.0"
	"pggat/lib/perror"
)

func clientFail(client fed.ReadWriter, err perror.Error) {
	// send fatal error to client
	resp := packets.ErrorResponse{
		Error: err,
	}
	_ = client.WritePacket(resp.IntoPacket())
}

func Bounce(client, server fed.ReadWriter, initialPacket fed.Packet) (clientError error, serverError error) {
	ctx := backends.Context{
		Peer: client,
	}
	serverError = backends.Transaction(&ctx, server, initialPacket)
	clientError = ctx.PeerError

	if clientError != nil {
		clientFail(client, perror.Wrap(clientError))
	} else if serverError != nil {
		clientFail(client, perror.Wrap(serverError))
	}

	return
}
