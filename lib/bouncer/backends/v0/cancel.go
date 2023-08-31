package backends

import "pggat2/lib/fed"

func Cancel(server fed.ReadWriter, key [8]byte) error {
	packet := fed.NewPacket(0, 12)
	packet = packet.AppendUint16(1234)
	packet = packet.AppendUint16(5678)
	packet = packet.AppendBytes(key[:])
	return server.WritePacket(packet)
}
