package packets

import "pggat2/lib/zap"

func ReadAuthenticationSASL(in zap.ReadablePacket) ([]string, bool) {
	if in.ReadType() != Authentication {
		return nil, false
	}

	method, ok := in.ReadInt32()
	if !ok {
		return nil, false
	}

	if method != 10 {
		return nil, false
	}

	in2 := in

	// get count first to prevent reallocating the slice a bunch
	var mechanismCount int
	for {
		mechanism, ok := in2.ReadString()
		if !ok {
			return nil, false
		}
		if mechanism == "" {
			break
		}
		mechanismCount++
	}

	mechanisms := make([]string, 0, mechanismCount)
	for i := 0; i < mechanismCount; i++ {
		mechanism, ok := in.ReadString()
		if !ok {
			return nil, false
		}
		mechanisms = append(mechanisms, mechanism)
	}

	return mechanisms, true
}

func WriteAuthenticationSASL(out *zap.Packet, mechanisms []string) {
	out.WriteType(Authentication)
	out.WriteInt32(10)
	for _, mechanism := range mechanisms {
		out.WriteString(mechanism)
	}
	out.WriteUint8(0)
}
