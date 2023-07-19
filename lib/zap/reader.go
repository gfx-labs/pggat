package zap

import "io"

type Reader interface {
	ReadByte() (byte, error)
	Read(*Packet) error
	ReadUntyped(*UntypedPacket) error

	Close() error
}

func WrapIOReader(readCloser io.ReadCloser) Reader {
	return ioReader{
		reader: readCloser,
		closer: readCloser,
	}
}

type ioReader struct {
	reader io.Reader
	closer io.Closer
}

func (T ioReader) ReadByte() (byte, error) {
	var res = []byte{0}
	_, err := io.ReadFull(T.reader, res)
	if err != nil {
		return 0, err
	}
	return res[0], err
}

func (T ioReader) Read(packet *Packet) error {
	_, err := packet.ReadFrom(T.reader)
	return err
}

func (T ioReader) ReadUntyped(packet *UntypedPacket) error {
	_, err := packet.ReadFrom(T.reader)
	return err
}

func (T ioReader) Close() error {
	return T.closer.Close()
}

var _ Reader = ioReader{}
