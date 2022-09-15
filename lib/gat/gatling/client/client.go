package client

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/tls"
	"errors"
	"fmt"
	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/gatling/messages"
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
	"gfx.cafe/gfx/pggat/lib/gat/protocol/pg_error"
	"gfx.cafe/gfx/pggat/lib/parse"
	"git.tuxpa.in/a/zlog"
	"git.tuxpa.in/a/zlog/log"
	"io"
	"math"
	"math/big"
	"net"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type CountReader[T io.Reader] struct {
	BytesRead atomic.Int64
	Reader    T
}

func NewCountReader[T io.Reader](reader T) *CountReader[T] {
	return &CountReader[T]{
		Reader: reader,
	}
}

func (C *CountReader[T]) Read(p []byte) (n int, err error) {
	n, err = C.Reader.Read(p)
	C.BytesRead.Add(int64(n))
	return
}

type CountWriter[T io.Writer] struct {
	BytesWritten atomic.Int64
	Writer       T
}

func NewCountWriter[T io.Writer](writer T) *CountWriter[T] {
	return &CountWriter[T]{
		Writer: writer,
	}
}

func (C *CountWriter[T]) Write(p []byte) (n int, err error) {
	n, err = C.Writer.Write(p)
	C.BytesWritten.Add(int64(n))
	return
}

// / client state, one per client
type Client struct {
	conn net.Conn
	r    *CountReader[*bufio.Reader]
	wr   *CountWriter[*bufio.Writer]

	recv chan protocol.Packet

	pid       int32
	secretKey int32

	connectTime time.Time

	server gat.ConnectionPool

	poolName string
	username string

	gatling     gat.Gat
	currentConn gat.Connection
	statements  map[string]*protocol.Parse
	portals     map[string]*protocol.Bind
	conf        *config.Global
	status      rune

	log zlog.Logger

	mu sync.Mutex
}

func (c *Client) State() string {
	return "TODO" // TODO
}

func (c *Client) Addr() string {
	addr, _, _ := net.SplitHostPort(c.conn.RemoteAddr().String())
	return addr
}

func (c *Client) Port() int {
	// ignore the errors cuz 0 is fine, just for stats
	_, port, _ := net.SplitHostPort(c.conn.RemoteAddr().String())
	p, _ := strconv.Atoi(port)
	return p
}

func (c *Client) LocalAddr() string {
	addr, _, _ := net.SplitHostPort(c.conn.LocalAddr().String())
	return addr
}

func (c *Client) LocalPort() int {
	_, port, _ := net.SplitHostPort(c.conn.LocalAddr().String())
	p, _ := strconv.Atoi(port)
	return p
}

func (c *Client) ConnectTime() time.Time {
	return c.connectTime
}

func (c *Client) RequestTime() time.Time {
	return c.currentConn.RequestTime()
}

func (c *Client) Wait() time.Duration {
	return c.currentConn.Wait()
}

func (c *Client) RemotePid() int {
	return int(c.pid)
}

func (c *Client) GetConnectionPool() gat.ConnectionPool {
	return c.server
}

func NewClient(
	gatling gat.Gat,
	conf *config.Global,
	conn net.Conn,
	admin_only bool,
) *Client {
	pid, _ := rand.Int(rand.Reader, big.NewInt(math.MaxInt32))
	skey, _ := rand.Int(rand.Reader, big.NewInt(math.MaxInt32))

	c := &Client{
		conn:       conn,
		r:          NewCountReader(bufio.NewReader(conn)),
		wr:         NewCountWriter(bufio.NewWriter(conn)),
		recv:       make(chan protocol.Packet),
		pid:        int32(pid.Int64()),
		secretKey:  int32(skey.Int64()),
		gatling:    gatling,
		statements: make(map[string]*protocol.Parse),
		portals:    make(map[string]*protocol.Bind),
		status:     'I',
		conf:       conf,
	}
	c.log = log.With().
		Stringer("clientaddr", c.conn.RemoteAddr()).Logger()
	return c
}

func (c *Client) Id() gat.ClientID {
	return gat.ClientID{
		PID:       c.pid,
		SecretKey: c.secretKey,
	}
}

func (c *Client) GetCurrentConn() (gat.Connection, error) {
	if c.currentConn == nil {
		return nil, errors.New("not connected to a server")
	}
	return c.currentConn, nil
}

func (c *Client) SetCurrentConn(conn gat.Connection) {
	c.currentConn = conn
}

func (c *Client) Accept(ctx context.Context) error {
	// read a packet
	startup := new(protocol.StartupMessage)
	err := startup.Read(c.r)
	if err != nil {
		return err
	}
	switch startup.Fields.ProtocolVersionNumber {
	case 196608:
	case 80877102:
		return c.handle_cancel(ctx, startup)
	case 80877103:
		// ssl stuff now
		useSsl := (c.conf.General.TlsCertificate != "")
		if !useSsl {
			_, err = protocol.WriteByte(c.wr, 'N')
			if err != nil {
				return err
			}
			err = c.wr.Writer.Flush()
			if err != nil {
				return err
			}
			startup = new(protocol.StartupMessage)
			err = startup.Read(c.r)
			if err != nil {
				return err
			}
		} else {
			_, err = protocol.WriteByte(c.wr, 'S')
			if err != nil {
				return err
			}
			err = c.wr.Writer.Flush()
			if err != nil {
				return err
			}
			//TODO: we need to do an ssl handshake here.
			var cert tls.Certificate
			cert, err = tls.LoadX509KeyPair(c.conf.General.TlsCertificate, c.conf.General.TlsPrivateKey)
			if err != nil {
				return err
			}
			cfg := &tls.Config{
				Certificates:       []tls.Certificate{cert},
				InsecureSkipVerify: true,
			}
			c.conn = tls.Server(c.conn, cfg)
			c.r.Reader = bufio.NewReader(c.conn)
			c.wr.Writer = bufio.NewWriter(c.conn)
			err = startup.Read(c.r)
			if err != nil {
				return err
			}
		}
	}
	params := make(map[string]string)
	for _, v := range startup.Fields.Parameters {
		params[v.Name] = v.Value
	}

	var ok bool
	c.poolName, ok = params["database"]
	if !ok {
		return &pg_error.Error{
			Severity: pg_error.Fatal,
			Code:     pg_error.InvalidAuthorizationSpecification,
			Message:  "param database required",
		}
	}

	c.username, ok = params["user"]
	if !ok {
		return &pg_error.Error{
			Severity: pg_error.Fatal,
			Code:     pg_error.InvalidAuthorizationSpecification,
			Message:  "param user required",
		}
	}

	admin := (c.poolName == "pgcat" || c.poolName == "pgbouncer")

	if c.conf.General.AdminOnly && !admin {
		c.log.Debug().Msg("rejected non admin, since admin only mode")
		return &pg_error.Error{
			Severity: pg_error.Fatal,
			Code:     pg_error.InvalidAuthorizationSpecification,
			Message:  "rejected non admin",
		}
	}

	// TODO: Add SASL support.

	// Perform MD5 authentication.
	pkt, salt, err := messages.CreateMd5Challenge()
	if err != nil {
		return err
	}
	err = c.Send(pkt)
	if err != nil {
		return err
	}
	err = c.Flush()
	if err != nil {
		return err
	}

	var rsp protocol.Packet
	rsp, err = protocol.ReadFrontend(c.r)
	if err != nil {
		return err
	}
	var passwordResponse []byte
	switch r := rsp.(type) {
	case *protocol.AuthenticationResponse:
		passwordResponse = r.Fields.Data
	default:
		return &pg_error.Error{
			Severity: pg_error.Fatal,
			Code:     pg_error.InvalidAuthorizationSpecification,
			Message:  fmt.Sprintf("wanted AuthenticationResponse packet, got '%+v'", rsp),
		}
	}

	var pool gat.Pool
	pool, err = c.gatling.GetPool(c.poolName)
	if err != nil {
		return err
	}

	// get user
	var user *config.User
	user, err = pool.GetUser(c.username)
	if err != nil {
		return err
	}

	// Authenticate admin user.
	if admin {
		pw_hash := messages.Md5HashPassword(c.conf.General.AdminUsername, c.conf.General.AdminPassword, salt[:])
		if !reflect.DeepEqual(pw_hash, passwordResponse) {
			return &pg_error.Error{
				Severity: pg_error.Fatal,
				Code:     pg_error.InvalidPassword,
				Message:  "invalid password",
			}
		}
	} else {
		pw_hash := messages.Md5HashPassword(c.username, user.Password, salt[:])
		if !reflect.DeepEqual(pw_hash, passwordResponse) {
			return &pg_error.Error{
				Severity: pg_error.Fatal,
				Code:     pg_error.InvalidPassword,
				Message:  "invalid password",
			}
		}
	}

	c.server, err = pool.WithUser(c.username)
	if err != nil {
		return err
	}

	authOk := new(protocol.Authentication)
	authOk.Fields.Code = 0
	err = c.Send(authOk)
	if err != nil {
		return err
	}

	//
	info := c.server.GetServerInfo()
	for _, inf := range info {
		err = c.Send(inf)
		if err != nil {
			return err
		}
	}
	backendKeyData := new(protocol.BackendKeyData)
	backendKeyData.Fields.ProcessID = c.pid
	backendKeyData.Fields.SecretKey = c.secretKey
	err = c.Send(backendKeyData)
	if err != nil {
		return err
	}
	readyForQuery := new(protocol.ReadyForQuery)
	readyForQuery.Fields.Status = byte('I')
	err = c.Send(readyForQuery)
	if err != nil {
		return err
	}
	go c.recvLoop()
	open := true
	for open {
		err = c.Flush()
		if err != nil {
			return err
		}
		open, err = c.tick(ctx)
		// add send and recv to pool
		stats := c.server.GetPool().GetStats()
		if stats != nil {
			stats.AddTotalSent(int(c.wr.BytesWritten.Swap(0)))
			stats.AddTotalReceived(int(c.r.BytesRead.Swap(0)))
		}
		if !open {
			break
		}
		if err != nil {
			err = c.Send(pg_error.IntoPacket(err))
			if err != nil {
				return err
			}
		}
		if c.status == 'I' {
			rq := new(protocol.ReadyForQuery)
			rq.Fields.Status = 'I'
			err = c.Send(rq)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Client) recvLoop() {
	for {
		recv, err := protocol.ReadFrontend(c.r)
		if err != nil {
			if !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
				log.Err(err)
			}
			break
		}
		//log.Printf("got packet(%s) %+v", reflect.TypeOf(recv), recv)
		c.recv <- recv
	}
}

func (c *Client) handle_cancel(ctx context.Context, p *protocol.StartupMessage) error {
	cl, err := c.gatling.GetClient(gat.ClientID{
		PID:       p.Fields.ProcessKey,
		SecretKey: p.Fields.SecretKey,
	})
	if err != nil {
		return err
	}
	var conn gat.Connection
	conn, err = cl.GetCurrentConn()
	if err != nil {
		return err
	}
	return conn.Cancel()
}

// reads a packet from stream and handles it
func (c *Client) tick(ctx context.Context) (bool, error) {
	var rsp protocol.Packet
	select {
	case rsp = <-c.recv:
	case <-ctx.Done():
		return false, ctx.Err()
	}
	switch cast := rsp.(type) {
	case *protocol.Parse:
		return true, c.parse(ctx, cast)
	case *protocol.Bind:
		return true, c.bind(ctx, cast)
	case *protocol.Describe:
		return true, c.handle_describe(ctx, cast)
	case *protocol.Execute:
		return true, c.handle_execute(ctx, cast)
	case *protocol.Sync:
		c.status = 'I'
		return true, nil
	case *protocol.Query:
		return true, c.handle_query(ctx, cast)
	case *protocol.FunctionCall:
		return true, c.handle_function(ctx, cast)
	case *protocol.Terminate:
		return false, nil
	default:
		log.Printf("unhandled packet %#v", rsp)
	}
	return true, nil
}

func (c *Client) parse(ctx context.Context, q *protocol.Parse) error {
	c.statements[q.Fields.PreparedStatement] = q
	c.status = 'T'
	return c.Send(new(protocol.ParseComplete))
}

func (c *Client) bind(ctx context.Context, b *protocol.Bind) error {
	c.portals[b.Fields.Destination] = b
	c.status = 'T'
	return c.Send(new(protocol.BindComplete))
}

func (c *Client) handle_describe(ctx context.Context, d *protocol.Describe) error {
	//log.Println("describe")
	c.status = 'T'
	return c.server.Describe(ctx, c, d)
}

func (c *Client) handle_execute(ctx context.Context, e *protocol.Execute) error {
	//log.Println("execute")
	c.status = 'T'
	return c.server.Execute(ctx, c, e)
}

func (c *Client) handle_query(ctx context.Context, q *protocol.Query) error {
	parsed, err := parse.Parse(q.Fields.Query)
	if err != nil {
		return err
	}

	// we can handle empty queries here
	if len(parsed) == 0 {
		err = c.Send(&protocol.EmptyQueryResponse{})
		if err != nil {
			return err
		}
		ready := new(protocol.ReadyForQuery)
		ready.Fields.Status = 'I'
		return c.Send(ready)
	}

	prev := 0
	transaction := false
	for idx, cmd := range parsed {
		switch strings.ToUpper(cmd.Command) {
		case "START":
			if len(cmd.Arguments) < 1 || strings.ToUpper(cmd.Arguments[0]) != "TRANSACTION" {
				break
			}
			fallthrough
		case "BEGIN":
			// begin transaction
			if prev != cmd.Index {
				query := q.Fields.Query[prev:cmd.Index]
				err = c.handle_simple_query(ctx, query)
				prev = cmd.Index
				if err != nil {
					return err
				}
			}
			transaction = true
		case "END":
			// end transaction block
			var query string
			if idx+1 >= len(parsed) {
				query = q.Fields.Query[prev:]
			} else {
				query = q.Fields.Query[prev:parsed[idx+1].Index]
			}
			if query != "" {
				err = c.handle_transaction(ctx, query)
				prev = cmd.Index
				if err != nil {
					return err
				}
			}
			transaction = false

		}
	}
	query := q.Fields.Query[prev:]
	if transaction {
		err = c.handle_transaction(ctx, query)
	} else {
		err = c.handle_simple_query(ctx, query)
	}
	return err
}

func (c *Client) handle_simple_query(ctx context.Context, q string) error {
	//log.Println("query:", q)
	return c.server.SimpleQuery(ctx, c, q)
}

func (c *Client) handle_transaction(ctx context.Context, q string) error {
	//log.Println("transaction:", q)
	return c.server.Transaction(ctx, c, q)
}

func (c *Client) handle_function(ctx context.Context, f *protocol.FunctionCall) error {
	err := c.server.CallFunction(ctx, c, f)
	if err != nil {
		return err
	}
	return err
}

func (c *Client) GetPreparedStatement(name string) *protocol.Parse {
	return c.statements[name]
}

func (c *Client) GetPortal(name string) *protocol.Bind {
	return c.portals[name]
}

func (c *Client) Send(pkt protocol.Packet) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	//log.Printf("sent packet(%s) %+v", reflect.TypeOf(pkt), pkt)
	_, err := pkt.Write(c.wr)
	return err
}

func (c *Client) Flush() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.wr.Writer.Flush()
}

func (c *Client) Recv() <-chan protocol.Packet {
	return c.recv
}

var _ gat.Client = (*Client)(nil)
