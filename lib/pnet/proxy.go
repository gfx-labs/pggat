package pnet

import "pggat2/lib/pnet/packet"

func ProxyPacket(writer Writer, in packet.In) error {
	out := writer.Write()
	packet.Proxy(out, in)
	return out.Send()
}

func Proxy(writer Writer, reader Reader) error {
	in, err := reader.Read()
	if err != nil {
		return err
	}
	return ProxyPacket(writer, in)
}
