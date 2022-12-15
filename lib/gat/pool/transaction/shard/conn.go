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

type conn struct {
	conn      gat.Connection
	conf      *config.Server
	replicaId int
	s         *Shard
}

func (s *conn) GetServerInfo() []*protocol.ParameterStatus {
	return s.conn.GetServerInfo()
}

func (s *conn) GetDatabase() string {
	return s.conn.GetDatabase()
}

func (s *conn) GetState() gat.ConnectionState {
	return s.conn.GetState()
}

func (s *conn) GetHost() string {
	return s.conn.GetHost()
}

func (s *conn) GetPort() int {
	return s.conn.GetPort()
}

func (s *conn) GetAddress() net.Addr {
	return s.conn.GetAddress()
}

func (s *conn) GetLocalAddress() net.Addr {
	return s.conn.GetLocalAddress()
}

func (s *conn) GetConnectTime() time.Time {
	return s.conn.GetConnectTime()
}

func (s *conn) GetRequestTime() time.Time {
	return s.conn.GetRequestTime()
}

func (s *conn) GetClient() gat.Client {
	return s.conn.GetClient()
}

func (s *conn) SetClient(client gat.Client) {
	s.conn.SetClient(client)
}

func (s *conn) GetRemotePid() int {
	return s.conn.GetRemotePid()
}

func (s *conn) GetTLS() string {
	return s.conn.GetTLS()
}

func (s *conn) IsCloseNeeded() bool {
	return s.conn.IsCloseNeeded()
}

func (s *conn) Close() error {
	return s.conn.Close()
}

func (s *conn) Describe(ctx context.Context, client gat.Client, payload *protocol.Describe) error {
	return s.conn.Describe(ctx, client, payload)
}

func (s *conn) Execute(ctx context.Context, client gat.Client, payload *protocol.Execute) error {
	return s.conn.Execute(ctx, client, payload)
}

func (s *conn) CallFunction(ctx context.Context, client gat.Client, payload *protocol.FunctionCall) error {
	return s.conn.CallFunction(ctx, client, payload)
}

func (s *conn) SimpleQuery(ctx context.Context, client gat.Client, payload string) error {
	return s.conn.SimpleQuery(ctx, client, payload)
}

func (s *conn) Transaction(ctx context.Context, client gat.Client, payload string) error {
	return s.conn.Transaction(ctx, client, payload)
}

func (s *conn) Cancel() error {
	return s.conn.Cancel()
}

func (s *conn) connect() {
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

func (s *conn) acquire() *conn {
	if s.conn == nil || s.conn.IsCloseNeeded() {
		s.connect()
	}
	return s
}
