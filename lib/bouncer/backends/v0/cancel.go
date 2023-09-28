package backends

import "gfx.cafe/gfx/pggat/lib/fed"

func Cancel(server *fed.Conn, key [8]byte) error {
	packet := fed.NewPacket(0, 12)
	packet = packet.AppendUint16(1234)
	packet = packet.AppendUint16(5678)
	packet = packet.AppendBytes(key[:])
	return server.WritePacket(packet)
}
