package backends

import "pggat2/lib/zap"

func Cancel(server zap.ReadWriter, key [8]byte) error {
	packet := zap.NewPacket(0)
	packet = packet.AppendUint16(1234)
	packet = packet.AppendUint16(5678)
	packet = packet.AppendBytes(key[:])
	return server.WritePacket(packet)
}
