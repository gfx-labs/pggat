package bouncers

import (
	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/perror"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

func clientFail(client zap.ReadWriter, err perror.Error) {
	// send fatal error to client
	packet := zap.NewPacket()
	defer packet.Done()
	packets.WriteErrorResponse(packet, err)
	_ = client.Write(packet)
}

func Bounce(client, server zap.ReadWriter) (clientError error, serverError error) {
	packet := zap.NewPacket()
	defer packet.Done()
	if clientError = client.Read(packet); clientError != nil {
		return
	}
	ctx := backends.Context{
		Peer: client,
	}
	serverError = backends.Transaction(&ctx, server, packet)
	clientError = ctx.PeerError

	if clientError != nil {
		clientFail(client, perror.Wrap(clientError))
	} else if serverError != nil {
		clientFail(client, perror.Wrap(serverError))
	}

	return
}
