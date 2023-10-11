package fed

import (
	"io"

	"gfx.cafe/gfx/pggat/lib/util/decorator"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

type Conn struct {
	noCopy decorator.NoCopy

	ReadWriter io.ReadWriteCloser
	Encoder    Encoder
	Decoder    Decoder

	Middleware []Middleware

	User              string
	Database          string
	InitialParameters map[strutil.CIString]string
	Authenticated     bool
	BackendKey        [8]byte
}

func NewConn(rw io.ReadWriteCloser) *Conn {
	c := &Conn{
		ReadWriter: rw,
	}
	c.Encoder.Writer.Reset(rw)
	c.Decoder.Reader.Reset(rw)
	return c
}

func (T *Conn) Flush() error {
	return T.Encoder.Flush()
}

func (T *Conn) ReadPacket(typed bool) (Packet, error) {
	if err := T.Flush(); err != nil {
		return nil, err
	}

	for {
		if err := T.Decoder.Next(typed); err != nil {
			return nil, err
		}
		var packet Packet
		packet = PendingPacket{
			Decoder: &T.Decoder,
		}
		for _, middleware := range T.Middleware {
			var err error
			packet, err = middleware.ReadPacket(packet)
			if err != nil {
				return nil, err
			}
			if packet == nil {
				break
			}
		}
		if packet != nil {
			return packet, nil
		}
	}
}

func (T *Conn) WritePacket(packet Packet) error {
	for _, middleware := range T.Middleware {
		var err error
		packet, err = middleware.WritePacket(packet)
		if err != nil {
			return err
		}
		if packet == nil {
			break
		}
	}
	if packet == nil {
		return nil
	}

	err := T.Encoder.Next(packet.Type(), packet.Length())
	if err != nil {
		return err
	}

	return packet.WriteTo(&T.Encoder)
}

func (T *Conn) Close() error {
	if err := T.Encoder.Flush(); err != nil {
		return err
	}

	return T.ReadWriter.Close()
}
