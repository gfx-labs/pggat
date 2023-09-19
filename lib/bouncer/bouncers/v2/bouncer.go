package bouncers

import (
	"pggat/lib/bouncer/backends/v0"
	"pggat/lib/fed"
	packets "pggat/lib/fed/packets/v3.0"
	"pggat/lib/perror"
)

func clientFail(ctx *backends.Context, client fed.ReadWriter, err perror.Error) {
	// send fatal error to client
	resp := packets.ErrorResponse{
		Error: err,
	}
	ctx.Packet = resp.IntoPacket(ctx.Packet)
	_ = client.WritePacket(ctx.Packet)
}

func Bounce(client, server fed.ReadWriter, initialPacket fed.Packet) (packet fed.Packet, clientError error, serverError error) {
	ctx := backends.Context{
		Server: server,
		Packet: initialPacket,
		Peer:   client,
	}
	serverError = backends.Transaction(&ctx)
	clientError = ctx.PeerError

	if clientError != nil {
		clientFail(&ctx, client, perror.Wrap(clientError))
	} else if serverError != nil {
		clientFail(&ctx, client, perror.Wrap(serverError))
	}

	packet = ctx.Packet

	return
}
