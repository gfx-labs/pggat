package packets

import "pggat2/lib/pnet/packet"

func ReadBackendKeyData(in packet.In) ([8]byte, bool) {
	in.Reset()
	if in.Type() != packet.BackendKeyData {
		return [8]byte{}, false
	}
	var cancellationKey [8]byte
	ok := in.Bytes(cancellationKey[:])
	if !ok {
		return cancellationKey, false
	}
	return cancellationKey, true
}

func WriteBackendKeyData(out packet.Out, cancellationKey [8]byte) {
	out.Reset()
	out.Type(packet.BackendKeyData)
	out.Bytes(cancellationKey[:])
}
