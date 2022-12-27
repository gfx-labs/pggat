package shard

import (
	"context"
	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
	"log"
	"net"
	"time"
)

type Conn struct {
	conn      gat.Connection
	conf      *config.Server
	replicaId int
	s         *Shard
}

func (s *Conn) GetServerInfo() []*protocol.ParameterStatus {
	return s.conn.GetServerInfo()
}

func (s *Conn) GetDatabase() string {
	return s.conn.GetDatabase()
}

func (s *Conn) GetState() gat.ConnectionState {
	return s.conn.GetState()
}

func (s *Conn) GetHost() string {
	return s.conn.GetHost()
}

func (s *Conn) GetPort() int {
	return s.conn.GetPort()
}

func (s *Conn) GetAddress() net.Addr {
	return s.conn.GetAddress()
}

func (s *Conn) GetLocalAddress() net.Addr {
	return s.conn.GetLocalAddress()
}

func (s *Conn) GetConnectTime() time.Time {
	return s.conn.GetConnectTime()
}

func (s *Conn) GetRequestTime() time.Time {
	return s.conn.GetRequestTime()
}

func (s *Conn) GetClient() gat.Client {
	return s.conn.GetClient()
}

func (s *Conn) SetClient(client gat.Client) {
	s.conn.SetClient(client)
}

func (s *Conn) GetRemotePid() int {
	return s.conn.GetRemotePid()
}

func (s *Conn) GetTLS() string {
	return s.conn.GetTLS()
}

func (s *Conn) IsCloseNeeded() bool {
	return s.conn.IsCloseNeeded()
}

func (s *Conn) Close() error {
	return s.conn.Close()
}

func (s *Conn) Describe(ctx context.Context, client gat.Client, payload *protocol.Describe) error {
	return s.conn.Describe(ctx, client, payload)
}

func (s *Conn) Execute(ctx context.Context, client gat.Client, payload *protocol.Execute) error {
	return s.conn.Execute(ctx, client, payload)
}

func (s *Conn) CallFunction(ctx context.Context, client gat.Client, payload *protocol.FunctionCall) error {
	return s.conn.CallFunction(ctx, client, payload)
}

func (s *Conn) SimpleQuery(ctx context.Context, client gat.Client, payload string) error {
	return s.conn.SimpleQuery(ctx, client, payload)
}

func (s *Conn) Transaction(ctx context.Context, client gat.Client, payload string) error {
	return s.conn.Transaction(ctx, client, payload)
}

func (s *Conn) Cancel() error {
	return s.conn.Cancel()
}

func (s *Conn) connect() {
	if s.s == nil || s.conf == nil {
		return
	}
	if s.conn != nil {
		_ = s.conn.Close()
	}
	var err error
	s.conn, err = s.s.dialer(context.TODO(), s.s.options, s.s.user, s.s.conf, s.conf)
	if err != nil {
		log.Println("error connecting to server:", err)
	}
	return
}

func (s *Conn) acquire() *Conn {
	if s.conn == nil || s.conn.IsCloseNeeded() {
		s.connect()
	}
	return s
}
