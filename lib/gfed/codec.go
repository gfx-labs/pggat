package gfed

import (
	"bufio"
	"crypto/tls"
	"io"
	"net"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/util/decorator"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
	"github.com/panjf2000/gnet/v2"
)

type Codec struct {
	localAddr net.Addr
	noCopy    decorator.NoCopy

	in *io.PipeWriter

	encoder fed.Encoder
	decoder fed.Decoder

	gnetConn gnet.Conn

	middleware []fed.Middleware

	ssl bool

	user              string
	database          string
	initialParameters map[strutil.CIString]string
	backendKey        fed.BackendKey

	authenticated bool
	ready         bool
}

func (c *Codec) SetUser(u string)                                   { c.user = u }
func (c *Codec) SetDatabase(d string)                               { c.database = d }
func (c *Codec) SetReady(b bool)                                    { c.ready = b }
func (c *Codec) SetAuthenticated(b bool)                            { c.authenticated = b }
func (c *Codec) Authenticated() bool                                { return c.authenticated }
func (c *Codec) Ready() bool                                        { return c.ready }
func (c *Codec) User() string                                       { return c.user }
func (c *Codec) Database() string                                   { return c.database }
func (c *Codec) LocalAddr() net.Addr                                { return c.localAddr }
func (c *Codec) SetBackendKey(i fed.BackendKey)                     { c.backendKey = i }
func (c *Codec) BackendKey() fed.BackendKey                         { return c.backendKey }
func (c *Codec) SSL() bool                                          { return c.ssl }
func (c *Codec) AddMiddleware(xs ...fed.Middleware)                 { c.middleware = append(c.middleware, xs...) }
func (c *Codec) Middleware() []fed.Middleware                       { return c.middleware }
func (c *Codec) InitialParameters() map[strutil.CIString]string     { return c.initialParameters }
func (c *Codec) SetInitialParameters(i map[strutil.CIString]string) { c.initialParameters = i }

func NewCodec() *Codec {
	// TODO: maybe use something copy free in the future?
	// this is complicated, and requires changing underlying
	// interfaces
	out, in := io.Pipe()
	//TODO: pool buffers maybe? idk. weird use case.
	rd := bufio.NewReader(out)
	c := &Codec{
		in: in,
	}
	c.decoder.Reset(rd)
	return c
}

func (T *Codec) OnData(buf []byte) error {
	_, err := T.in.Write(buf)
	if err != nil {
		return err
	}
	return nil
}

func (T *Codec) Flush() error {
	return T.encoder.Flush()
}

func (T *Codec) readPacket(typed bool) (fed.Packet, error) {
	if err := T.decoder.Next(typed); err != nil {
		return nil, err
	}
	return fed.PendingPacket{
		Decoder: &T.decoder,
	}, nil
}

func (T *Codec) ReadPacket(typed bool) (fed.Packet, error) {
	if err := T.Flush(); err != nil {
		return nil, err
	}
	mw := T.Middleware()
	for {
		// try doing PreRead
		for i := 0; i < len(mw); i++ {
			middleware := mw[i]
			for {
				packet, err := middleware.PreRead(typed)
				if err != nil {
					return nil, err
				}

				if packet == nil {
					break
				}

				for j := i; j < len(mw); j++ {
					packet, err = mw[j].ReadPacket(packet)
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
		for _, middleware := range mw {
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

func (T *Codec) writePacket(packet fed.Packet) error {
	err := T.encoder.Next(packet.Type(), packet.Length())
	if err != nil {
		return err
	}

	return packet.WriteTo(&T.encoder)
}

func (T *Codec) WritePacket(packet fed.Packet) error {
	mw := T.Middleware()
	for i := len(mw) - 1; i >= 0; i-- {
		middleware := mw[i]

		var err error
		packet, err = middleware.WritePacket(packet)
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
	for i := len(mw) - 1; i >= 0; i-- {
		middleware := mw[i]

		for {
			var err error
			packet, err = middleware.PostWrite()
			if err != nil {
				return err
			}

			if packet == nil {
				break
			}

			for j := i; j >= 0; j-- {
				packet, err = mw[j].WritePacket(packet)
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

func (T *Codec) WriteByte(b byte) error {
	return T.encoder.WriteByte(b)
}

func (T *Codec) ReadByte() (byte, error) {
	if err := T.Flush(); err != nil {
		return 0, err
	}

	return T.decoder.ReadByte()
}

func (T *Codec) EnableSSL(config *tls.Config, isClient bool) error {
	panic("ssl not supported")
}

func (T *Codec) Close() error {
	if err := T.encoder.Flush(); err != nil {
		return err
	}

	return T.gnetConn.Close()
}
