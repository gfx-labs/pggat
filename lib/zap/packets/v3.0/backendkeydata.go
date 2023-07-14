package packets

import (
	"pggat2/lib/zap"
)

func ReadBackendKeyData(in zap.In) ([8]byte, bool) {
	in.Reset()
	if in.Type() != BackendKeyData {
		return [8]byte{}, false
	}
	var cancellationKey [8]byte
	ok := in.Bytes(cancellationKey[:])
	if !ok {
		return cancellationKey, false
	}
	return cancellationKey, true
}

func WriteBackendKeyData(out zap.Out, cancellationKey [8]byte) {
	out.Reset()
	out.Type(BackendKeyData)
	out.Bytes(cancellationKey[:])
}
