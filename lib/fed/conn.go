package fed

import (
	"context"
	"crypto/tls"
	"io"
	"net"

	"gfx.cafe/gfx/pggat/lib/util/decorator"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

type Listener interface {
	Accept(fn func(*Conn)) error
	io.Closer
}

type Conn struct {
	noCopy decorator.NoCopy

	codec PacketCodec
	Ctx   context.Context

	Middleware []Middleware

	SSL bool

	User              string
	Database          string
	InitialParameters map[strutil.CIString]string
	BackendKey        BackendKey

	Authenticated bool
	Ready         bool
}

func NewConn(ctx context.Context, codec PacketCodec) *Conn {
	c := &Conn{
		Ctx:   ctx,
		codec: codec,
	}
	return c
}

func (T *Conn) Flush() error {
	return T.codec.Flush()
}

func (T *Conn) readPacket(typed bool) (Packet, error) {
	return T.codec.ReadPacket(typed)
}

func (T *Conn) ReadPacket(typed bool) (Packet, error) {
	if err := T.Flush(); err != nil {
		return nil, err
	}

	for {
		// try doing PreRead
		for i := 0; i < len(T.Middleware); i++ {
			middleware := T.Middleware[i]
			for {
				packet, err := middleware.PreRead(T.Ctx, typed)
				if err != nil {
					return nil, err
				}

				if packet == nil {
					break
				}

				for j := i; j < len(T.Middleware); j++ {
					packet, err = T.Middleware[j].ReadPacket(T.Ctx, packet)
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

		packet, err := T.readPacket(typed)
		if err != nil {
			return nil, err
		}
		for _, middleware := range T.Middleware {
			packet, err = middleware.ReadPacket(T.Ctx, packet)
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

func (T *Conn) writePacket(packet Packet) error {
	return T.codec.WritePacket(packet)
}

func (T *Conn) WritePacket(packet Packet) error {
	for i := len(T.Middleware) - 1; i >= 0; i-- {
		middleware := T.Middleware[i]

		var err error
		packet, err = middleware.WritePacket(T.Ctx, packet)
		if err != nil {
			return err
		}
		if packet == nil {
			break
		}
	}
	if packet != nil {
		if err := T.writePacket(packet); err != nil {
			return err
		}
	}

	// try doing PostWrite
	for i := len(T.Middleware) - 1; i >= 0; i-- {
		middleware := T.Middleware[i]

		for {
			var err error
			packet, err = middleware.PostWrite(T.Ctx)
			if err != nil {
				return err
			}

			if packet == nil {
				break
			}

			for j := i; j >= 0; j-- {
				packet, err = T.Middleware[j].WritePacket(T.Ctx, packet)
				if err != nil {
					return err
				}
				if packet == nil {
					break
				}
			}

			if packet != nil {
				if err = T.writePacket(packet); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (T *Conn) WriteByte(b byte) error {
	return T.codec.WriteByte(b)
}

func (T *Conn) LocalAddr() net.Addr {
	return T.codec.LocalAddr()

}

func (T *Conn) ReadByte() (byte, error) {
	return T.codec.ReadByte()
}

func (T *Conn) EnableSSL(config *tls.Config, isClient bool) error {
	return T.codec.EnableSSL(config, isClient)
}

func (T *Conn) Close() error {
	return T.codec.Close()
}
