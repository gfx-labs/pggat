package zap

func InToOut(in In) Out {
	return Out{
		buf: in.buf,
		rev: in.rev,
	}
}

func OutToIn(out Out) In {
	return In{
		buf: out.buf,
		rev: out.rev,
	}
}
