package bouncers

import (
	"gfx.cafe/gfx/pggat/lib/bouncer/backends/v0"
	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/lib/perror"
)

func clientFail(packet fed.Packet, client *fed.Conn, err perror.Error) fed.Packet {
	// send fatal error to client
	resp := packets.ErrorResponse{
		Error: err,
	}
	packet = resp.IntoPacket(packet)
	_ = client.WritePacket(packet)
	return packet
}

func Bounce(client, server *fed.Conn, initialPacket fed.Packet) (packet fed.Packet, clientError error, serverError error) {
	serverError, clientError, packet = backends.Transaction(server, client, initialPacket)

	if clientError != nil {
		packet = clientFail(packet, client, perror.Wrap(clientError))
	} else if serverError != nil {
		packet = clientFail(packet, client, perror.Wrap(serverError))
	}

	return
}
