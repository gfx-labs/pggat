package bouncers

import (
	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/perror"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

func clientFail(client zap.ReadWriter, err perror.Error) {
	// send fatal error to client
	resp := packets.ErrorResponse{
		Error: err,
	}
	_ = client.WritePacket(resp.IntoPacket())
}

func Bounce(client, server zap.ReadWriter, initialPacket zap.Packet) (clientError error, serverError error) {
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
