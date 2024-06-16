package fed

import (
	"crypto/tls"
	"errors"
	"net"

	"gfx.cafe/gfx/pggat/lib/util/decorator"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

type conn struct {
	noCopy decorator.NoCopy

	encoder Encoder
	decoder Decoder

	NetConn net.Conn

	middleware []Middleware

	ssl bool

	user              string
	database          string
	initialParameters map[strutil.CIString]string
	backendKey        BackendKey

	authenticated bool
	ready         bool
}

func (c *conn) SetUser(u string)                                   { c.user = u }
func (c *conn) SetDatabase(d string)                               { c.database = d }
func (c *conn) SetBackendKey(i BackendKey)                         { c.backendKey = i }
func (c *conn) AddMiddleware(xs ...Middleware)                     { c.middleware = append(c.middleware, xs...) }
func (c *conn) SetReady(b bool)                                    { c.ready = b }
func (c *conn) SetAuthenticated(b bool)                            { c.authenticated = b }
func (c *conn) Authenticated() bool                                { return c.authenticated }
func (c *conn) Ready() bool                                        { return c.ready }
func (c *conn) User() string                                       { return c.user }
func (c *conn) Database() string                                   { return c.database }
func (c *conn) LocalAddr() net.Addr                                { return c.NetConn.LocalAddr() }
func (c *conn) BackendKey() BackendKey                             { return c.backendKey }
func (c *conn) SSL() bool                                          { return c.ssl }
func (c *conn) Middleware() []Middleware                           { return c.middleware }
func (c *conn) InitialParameters() map[strutil.CIString]string     { return c.initialParameters }
func (c *conn) SetInitialParameters(i map[strutil.CIString]string) { c.initialParameters = i }

func NewConn(rw net.Conn) Conn {
	c := &conn{
		NetConn:           rw,
		initialParameters: map[strutil.CIString]string{},
	}
	c.encoder.Reset(rw)
	c.decoder.Reset(rw)
	return c
}

func (T *conn) Flush() error {
	return T.encoder.Flush()
}

func (T *conn) readPacket(typed bool) (Packet, error) {
	if err := T.decoder.Next(typed); err != nil {
		return nil, err
	}
	return PendingPacket{
		Decoder: &T.decoder,
	}, nil
}

func (T *conn) ReadPacket(typed bool) (Packet, error) {
	if err := T.Flush(); err != nil {
		return nil, err
	}

	for {
		// try doing PreRead
		for i := 0; i < len(T.middleware); i++ {
			middleware := T.middleware[i]
			for {
				packet, err := middleware.PreRead(typed)
				if err != nil {
					return nil, err
				}

				if packet == nil {
					break
				}

				for j := i; j < len(T.middleware); j++ {
					packet, err = T.middleware[j].ReadPacket(packet)
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
		for _, middleware := range T.middleware {
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

func (T *conn) writePacket(packet Packet) error {
	err := T.encoder.Next(packet.Type(), packet.Length())
	if err != nil {
		return err
	}

	return packet.WriteTo(&T.encoder)
}

func (T *conn) WritePacket(packet Packet) error {
	for i := len(T.middleware) - 1; i >= 0; i-- {
		middleware := T.middleware[i]

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
	for i := len(T.middleware) - 1; i >= 0; i-- {
		middleware := T.middleware[i]

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
				packet, err = T.middleware[j].WritePacket(packet)
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

func (T *conn) WriteByte(b byte) error {
	return T.encoder.WriteByte(b)
}

func (T *conn) ReadByte() (byte, error) {
	if err := T.Flush(); err != nil {
		return 0, err
	}

	return T.decoder.ReadByte()
}

func (T *conn) EnableSSL(config *tls.Config, isClient bool) error {
	if T.ssl {
		return errors.New("SSL is already enabled")
	}
	T.ssl = true

	// Flush buffers
	if err := T.Flush(); err != nil {
		return err
	}
	if T.decoder.Buffered() > 0 {
		return errors.New("expected empty read buffer")
	}

	var sslConn *tls.Conn
	if isClient {
		sslConn = tls.Client(T.NetConn, config)
	} else {
		sslConn = tls.Server(T.NetConn, config)
	}
	T.encoder.Reset(sslConn)
	T.decoder.Reset(sslConn)
	T.NetConn = sslConn
	return sslConn.Handshake()
}

func (T *conn) Close() error {
	if err := T.encoder.Flush(); err != nil {
		return err
	}

	return T.NetConn.Close()
}
