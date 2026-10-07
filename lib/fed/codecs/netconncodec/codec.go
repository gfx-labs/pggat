package netconncodec

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

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

func (c *Codec) ReadPacket(ctx context.Context, typed bool) (fed.Packet, error) {
	if err := c.decoder.Next(typed); err != nil {
		return nil, err
	}
	return fed.PendingPacket{
		Decoder: &c.decoder,
	}, nil
}

func (c *Codec) WritePacket(ctx context.Context, packet fed.Packet) error {
	err := c.encoder.Next(packet.Type(), packet.Length())
	if err != nil {
		return err
	}

	return packet.WriteTo(&c.encoder)
}
func (c *Codec) WriteByte(ctx context.Context, b byte) error {
	return c.encoder.WriteByte(b)
}

func (c *Codec) ReadByte(ctx context.Context) (byte, error) {
	if err := c.Flush(ctx); err != nil {
		return 0, err
	}

	return c.decoder.ReadByte()
}

func (c *Codec) Flush(ctx context.Context) error {
	return c.encoder.Flush()
}

func (c *Codec) Close(ctx context.Context) error {
	// Always close the socket, even if the flush fails on a dead connection.
	return errors.Join(c.encoder.Flush(), c.conn.Close())
}

func (c *Codec) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *Codec) SSL() bool {
	return c.ssl
}

func (c *Codec) EnableSSL(ctx context.Context, config *tls.Config, isClient bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ssl {
		return errors.New("SSL is already enabled")
	}
	c.ssl = true

	// Flush buffers
	if err := c.Flush(ctx); err != nil {
		return err
	}
	if c.decoder.Buffered() > 0 {
		return errors.New("expected empty read buffer")
	}

	var sslConn *tls.Conn
	obs := &observedConn{Conn: c.conn}
	if isClient {
		sslConn = tls.Client(obs, config)
	} else {
		sslConn = tls.Server(obs, config)
	}
	c.encoder.Reset(sslConn)
	c.decoder.Reset(sslConn)
	c.conn = sslConn
	err := sslConn.Handshake()
	if err != nil {
		return fmt.Errorf("ssl handshake fail client(%v): %w", isClient, err)
	}
	return nil
}

// IdleUsable reports false if an idle connection has unread input or is closed. True is best-effort.
func (c *Codec) IdleUsable() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Every packet of the last response must be fully consumed. An unread tail would
	// be indistinguishable from unsolicited input in the socket, so fail closed.
	if c.decoder.Buffered() != 0 || c.decoder.Length() != c.decoder.Position() {
		return false
	}

	if tc, ok := c.conn.(*tls.Conn); ok {
		return tlsIdleUsable(tc)
	}
	return !peekPending(c.conn)
}

// observedConn records transport write failures. crypto/tls can write on Read (a KeyUpdate
// response) and keeps that failure in its own write state without returning it.
type observedConn struct {
	net.Conn
	writeFailed atomic.Bool
}

func (o *observedConn) Write(b []byte) (int, error) {
	n, err := o.Conn.Write(b)
	if err != nil {
		o.writeFailed.Store(true)
	}
	return n, err
}

// tlsIdleUsable checks data crypto/tls already holds, then the socket under it.
func tlsIdleUsable(conn *tls.Conn) bool {
	obs, ok := conn.NetConn().(*observedConn)
	if !ok {
		return false
	}

	// Expired deadlines make the probe unable to block, even if Read answers a KeyUpdate.
	// Read still returns data crypto/tls buffered earlier, and a timeout is not a sticky TLS error.
	if err := conn.SetDeadline(time.Unix(1, 0)); err != nil {
		return false
	}
	var b [1]byte
	n, err := conn.Read(b[:])
	// The codec sets no other deadlines, so clearing restores the previous state.
	if resetErr := conn.SetDeadline(time.Time{}); resetErr != nil {
		return false
	}
	var ne net.Error
	if n != 0 || !errors.As(err, &ne) || !ne.Timeout() || obs.writeFailed.Load() {
		return false
	}
	return !peekPending(obs.Conn)
}
