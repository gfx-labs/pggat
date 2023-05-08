package packet

func Proxy(out Out, in In) {
	out.Type(in.Type())
	out.Bytes(in.Full())
}
