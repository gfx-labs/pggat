package gnetcodec

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"sync"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/util/decorator"
	"github.com/panjf2000/gnet/v2"
)

type Codec struct {
	noCopy decorator.NoCopy

	conn gnet.Conn
	ssl  bool

	in *io.PipeWriter
	wr io.Writer

	encoder fed.Encoder
	decoder fed.Decoder

	mu sync.RWMutex
}

func NewCodec() *Codec {
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

func (c *Codec) ReadPacket(typed bool) (fed.Packet, error) {
	if err := c.decoder.Next(typed); err != nil {
		return nil, err
	}
	return fed.PendingPacket{
		Decoder: &c.decoder,
	}, nil
}

func (c *Codec) WritePacket(packet fed.Packet) error {
	c.mu.Lock()
	defer c.mu.Lock()
	err := c.encoder.Next(packet.Type(), packet.Length())
	if err != nil {
		return err
	}
	return packet.WriteTo(&c.encoder)
}
func (c *Codec) WriteByte(b byte) error {
	c.mu.Lock()
	defer c.mu.Lock()
	return c.encoder.WriteByte(b)
}

func (c *Codec) Flush() error {
	c.mu.Lock()
	defer c.mu.Lock()
	return c.encoder.Flush()
}

func (c *Codec) ReadByte() (byte, error) {
	if err := c.Flush(); err != nil {
		return 0, err
	}

	return c.decoder.ReadByte()
}

func (c *Codec) Close() error {
	c.mu.Lock()
	defer c.mu.Lock()
	if err := c.encoder.Flush(); err != nil {
		return err
	}
	return c.conn.Close()
}

func (c *Codec) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *Codec) SSL() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ssl
}

func (c *Codec) EnableSSL(config *tls.Config, isClient bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return fmt.Errorf("ssl not supported by gnet backend")
	//if c.ssl {
	//	return errors.New("SSL is already enabled")
	//}
	//c.ssl = true

	//// Flush buffers
	//if err := c.Flush(); err != nil {
	//	return err
	//}
	//if c.decoder.Buffered() > 0 {
	//	return errors.New("expected empty read buffer")
	//}

	//var sslConn *tls.Conn
	//if isClient {
	//	sslConn = tls.Client(c.conn, config)
	//} else {
	//	sslConn = tls.Server(c.conn, config)
	//}
	//c.encoder.Reset(sslConn)
	//c.decoder.Reset(sslConn)
	//c.conn = sslConn
	//err := sslConn.Handshake()
	//if err != nil {
	//	return fmt.Errorf("ssl handshake fail client(%v): %w", isClient, err)
	//}
	return nil
}
