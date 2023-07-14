package packets

import (
	"pggat2/lib/zap"
)

func ReadExecute(in zap.In) (target string, maxRows int32, ok bool) {
	in.Reset()
	if in.Type() != Execute {
		return
	}
	target, ok = in.String()
	if !ok {
		return
	}
	maxRows, ok = in.Int32()
	if !ok {
		return
	}
	return
}

func WriteExecute(out zap.Out, target string, maxRows int32) {
	out.Reset()
	out.Type(Execute)
	out.String(target)
	out.Int32(maxRows)
}
