package packets

import (
	"pggat2/lib/util/slices"
	"pggat2/lib/zap"
)

type SASLInitialResponse struct {
	Mechanism       string
	InitialResponse []byte
}

func (T *SASLInitialResponse) ReadFromPacket(packet zap.Packet) bool {
	if packet.Type() != TypeAuthenticationResponse {
		return false
	}

	p := packet.ReadString(&T.Mechanism)

	var initialResponseSize int32
	p = p.ReadInt32(&initialResponseSize)

	T.InitialResponse = slices.Resize(T.InitialResponse, int(initialResponseSize))
	p = p.ReadBytes(T.InitialResponse[:])

	return true
}

func (T *SASLInitialResponse) IntoPacket() zap.Packet {
	packet := zap.NewPacket(TypeAuthenticationResponse)
	packet = packet.AppendString(T.Mechanism)
	packet = packet.AppendInt32(int32(len(T.InitialResponse)))
	packet = packet.AppendBytes(T.InitialResponse)
	return packet
}
