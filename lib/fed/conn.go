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

	Middleware []Middleware

	SSL bool

	User              string
	Database          string
	InitialParameters map[strutil.CIString]string
	BackendKey        BackendKey

	Authenticated bool
	Ready         bool
}

func NewConn(codec PacketCodec) *Conn {
	c := &Conn{
		codec: codec,
	}
	return c
}

func (T *Conn) Flush(ctx context.Context) error {
	return T.codec.Flush(ctx)
}

func (T *Conn) readPacket(ctx context.Context, typed bool) (Packet, error) {
	return T.codec.ReadPacket(ctx, typed)
}

func (T *Conn) ReadPacket(ctx context.Context, typed bool) (Packet, error) {
	if err := T.Flush(ctx); err != nil {
		return nil, err
	}

	for {
		// try doing PreRead
		for i := 0; i < len(T.Middleware); i++ {
			middleware := T.Middleware[i]
			for {
				packet, err := middleware.PreRead(ctx, typed)
				if err != nil {
					return nil, err
				}

				if packet == nil {
					break
				}

				for j := i; j < len(T.Middleware); j++ {
					packet, err = T.Middleware[j].ReadPacket(ctx, packet)
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

		packet, err := T.readPacket(ctx, typed)
		if err != nil {
			return nil, err
		}
		for _, middleware := range T.Middleware {
			packet, err = middleware.ReadPacket(ctx, packet)
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

func (T *Conn) writePacket(ctx context.Context, packet Packet) error {
	return T.codec.WritePacket(ctx, packet)
}

func (T *Conn) WritePacket(ctx context.Context, packet Packet) error {
	for i := len(T.Middleware) - 1; i >= 0; i-- {
		middleware := T.Middleware[i]

		var err error
		packet, err = middleware.WritePacket(ctx, packet)
		if err != nil {
			return err
		}
		if packet == nil {
			break
		}
	}
	if packet != nil {
		if err := T.writePacket(ctx, packet); err != nil {
			return err
		}
	}

	// try doing PostWrite
	for i := len(T.Middleware) - 1; i >= 0; i-- {
		middleware := T.Middleware[i]

		for {
			var err error
			packet, err = middleware.PostWrite(ctx)
			if err != nil {
				return err
			}

			if packet == nil {
				break
			}

			for j := i; j >= 0; j-- {
				packet, err = T.Middleware[j].WritePacket(ctx, packet)
				if err != nil {
					return err
				}
				if packet == nil {
					break
				}
			}

			if packet != nil {
				if err = T.writePacket(ctx, packet); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (T *Conn) WriteByte(ctx context.Context, b byte) error {
	return T.codec.WriteByte(ctx, b)
}

func (T *Conn) LocalAddr() net.Addr {
	return T.codec.LocalAddr()

}

func (T *Conn) ReadByte(ctx context.Context) (byte, error) {
	return T.codec.ReadByte(ctx)
}

func (T *Conn) EnableSSL(ctx context.Context, config *tls.Config, isClient bool) error {
	return T.codec.EnableSSL(ctx, config, isClient)
}

func (T *Conn) Close(ctx context.Context) error {
	return T.codec.Close(ctx)
}
