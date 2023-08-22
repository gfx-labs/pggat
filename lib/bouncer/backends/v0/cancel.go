package backends

import "pggat2/lib/zap"

func Cancel(server zap.ReadWriter, key [8]byte) error {
	packet := zap.NewUntypedPacket()
	defer packet.Done()
	packet.WriteUint16(1234)
	packet.WriteUint16(5678)
	packet.WriteBytes(key[:])
	return server.WriteUntyped(packet)
}
