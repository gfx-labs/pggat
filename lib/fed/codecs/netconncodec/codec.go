package netconncodec

import (
	"crypto/tls"
	"errors"
	"net"
	"sync"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/util/decorator"
)

type Codec struct {
	noCopy decorator.NoCopy

	conn net.Conn
	ssl  bool

	encoder fed.Encoder
	decoder fed.Decoder

	mu sync.RWMutex
}

func NewCodec(rw net.Conn) fed.PacketCodec {
	c := &Codec{
		conn: rw,
	}
	c.encoder.Reset(rw)
	c.decoder.Reset(rw)
	return c
}

func (c *Codec) ReadPacket(typed bool) (fed.Packet, error) {
	if err := c.decoder.Next(typed); err != nil {
		return nil, err
	}
	return fed.PendingPacket{
		Decoder: &c.decoder,
	}, nil
}

func (c *Codec) WritePacket(packet fed.Packet) error {
	err := c.encoder.Next(packet.Type(), packet.Length())
	if err != nil {
		return err
	}

	return packet.WriteTo(&c.encoder)
}

func (c *Codec) Flush() error {
	return c.encoder.Flush()
}

func (c *Codec) Close() error {
	if err := c.encoder.Flush(); err != nil {
		return err
	}
	return c.conn.Close()
}

func (c *Codec) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *Codec) SSL() bool {
	return c.ssl
}

func (c *Codec) EnableSSL(config *tls.Config, isClient bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ssl {
		return errors.New("SSL is already enabled")
	}
	c.ssl = true

	// Flush buffers
	if err := c.Flush(); err != nil {
		return err
	}
	if c.decoder.Buffered() > 0 {
		return errors.New("expected empty read buffer")
	}

	var sslConn *tls.Conn
	if isClient {
		sslConn = tls.Client(c.conn, config)
	} else {
		sslConn = tls.Server(c.conn, config)
	}
	c.encoder.Reset(sslConn)
	c.decoder.Reset(sslConn)
	c.conn = sslConn
	return sslConn.Handshake()
}
