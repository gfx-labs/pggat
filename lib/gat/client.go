package gat

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net"

	"gfx.cafe/gfx/pggat/lib/config"
)

type ClientConnectionType interface {
}

var _ []ClientConnectionType = []ClientConnectionType{
	&StartupConnection{},
	&TLSConnection{},
	&CancelQueryConnection{},
}

type StartupConnection struct {
}

type TLSConnection struct {
}

type CancelQueryConnection struct {
}

type ClientKey [2]int

type ClientInfo struct {
	A int
	B int
	C string
	D uint16
}

// / client state, one per client
type Client struct {
	conn net.Conn
	r    io.Reader
	wr   io.Writer

	buf bytes.Buffer

	addr net.Addr

	cancel_mode bool
	txn_mode    bool

	pid        int
	secret_key int

	csm        map[ClientKey]ClientInfo
	parameters map[string]string
	stats      any // TODO: Reporter
	admin      bool

	last_addr_id int
	last_srv_id  int

	connected_to_server bool
	pool_name           string
	username            string
}

func NewClient(
	conf *config.Global,
	conn net.Conn,
	csm map[ClientKey]ClientInfo,
	admin_only bool,
) *Client {
	c := &Client{
		conn: conn,
		r:    bufio.NewReader(conn),
		wr:   conn,
		addr: conn.RemoteAddr(),
		csm:  csm,
	}
	return c
}

func (c *Client) Accept(ctx context.Context) error {
	return nil
}
