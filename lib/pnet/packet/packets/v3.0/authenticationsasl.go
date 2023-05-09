package packets

import "pggat2/lib/pnet/packet"

func ReadAuthenticationSASL(in packet.In) ([]string, bool) {
	in.Reset()
	if in.Type() != packet.Authentication {
		return nil, false
	}

	method, ok := in.Int32()
	if !ok {
		return nil, false
	}

	if method != 10 {
		return nil, false
	}

	// get count first to prevent reallocating the slice a bunch
	var mechanismCount int
	for {
		mechanism, ok := in.String()
		if !ok {
			return nil, false
		}
		if mechanism == "" {
			break
		}
		mechanismCount++
	}

	mechanisms := make([]string, 0, mechanismCount)
	in.Reset()
	in.Int32()
	for i := 0; i < mechanismCount; i++ {
		mechanism, ok := in.String()
		if !ok {
			return nil, false
		}
		mechanisms = append(mechanisms, mechanism)
	}

	return mechanisms, true
}

func WriteAuthenticationSASL(out packet.Out, mechanisms []string) {
	out.Reset()
	out.Type(packet.Authentication)
	out.Int32(10)
	for _, mechanism := range mechanisms {
		out.String(mechanism)
	}
	out.Uint8(0)
}
