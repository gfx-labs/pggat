package fed

import (
	"crypto/tls"
	"errors"
	"io"
	"net"

	"gfx.cafe/gfx/pggat/lib/util/decorator"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

type Conn struct {
	noCopy decorator.NoCopy

	encoder Encoder
	decoder Decoder

	ReadWriter io.ReadWriteCloser

	Middleware []Middleware
	SSL        bool

	User              string
	Database          string
	InitialParameters map[strutil.CIString]string
	Authenticated     bool
	BackendKey        BackendKey
}

func NewConn(rw io.ReadWriteCloser) *Conn {
	c := &Conn{
		ReadWriter: rw,
	}
	c.encoder.Writer.Reset(rw)
	c.decoder.Reader.Reset(rw)
	return c
}

func (T *Conn) flush() error {
	return T.encoder.Flush()
}

func (T *Conn) ReadPacket(typed bool) (Packet, error) {
	if err := T.flush(); err != nil {
		return nil, err
	}

	for {
		if err := T.decoder.Next(typed); err != nil {
			return nil, err
		}
		var packet Packet
		packet = PendingPacket{
			Decoder: &T.decoder,
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

	err := T.encoder.Next(packet.Type(), packet.Length())
	if err != nil {
		return err
	}

	return packet.WriteTo(&T.encoder)
}

func (T *Conn) WriteByte(b byte) error {
	return T.encoder.WriteByte(b)
}

func (T *Conn) ReadByte() (byte, error) {
	if err := T.flush(); err != nil {
		return 0, err
	}

	return T.decoder.ReadByte()
}

func (T *Conn) EnableSSL(config *tls.Config, isClient bool) error {
	if T.SSL {
		return errors.New("SSL is already enabled")
	}
	T.SSL = true

	// flush buffers
	if err := T.flush(); err != nil {
		return err
	}
	if T.decoder.Reader.Buffered() > 0 {
		return errors.New("expected empty read buffer")
	}

	conn, ok := T.ReadWriter.(net.Conn)
	if !ok {
		return errors.New("ssl not supported for this read writer")
	}

	var sslConn *tls.Conn
	if isClient {
		sslConn = tls.Client(conn, config)
	} else {
		sslConn = tls.Server(conn, config)
	}
	T.encoder.Writer.Reset(sslConn)
	T.decoder.Reader.Reset(sslConn)
	T.ReadWriter = sslConn
	return sslConn.Handshake()
}

func (T *Conn) Close() error {
	if err := T.encoder.Flush(); err != nil {
		return err
	}

	return T.ReadWriter.Close()
}
