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

	state gat.ClientState

	pid       int32
	secretKey int32

	requestTime time.Time
	connectTime time.Time

	server gat.ConnectionPool

	poolName string
	username string

	shardingKey       string
	preferredShard    int
	hasPreferredShard bool

	gatling     gat.Gat
	currentConn gat.Connection
	statements  map[string]*protocol.Parse
	portals     map[string]*protocol.Bind
	conf        *config.Global
	status      rune

	log zlog.Logger

	mu sync.Mutex
}

func (c *Client) GetState() gat.ClientState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

func (c *Client) GetAddress() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *Client) GetLocalAddress() net.Addr {
	return c.conn.LocalAddr()
}

func (c *Client) GetConnectTime() time.Time {
	return c.connectTime
}

func (c *Client) startRequest() {
	c.state = gat.ClientWaiting
	c.requestTime = time.Now()
}

func (c *Client) GetRequestTime() time.Time {
	return c.requestTime
}

func (c *Client) GetRemotePid() int {
	return int(c.pid)
}

func (c *Client) GetConnectionPool() gat.ConnectionPool {
	return c.server
}

func (c *Client) SetRequestedShard(shard int) {
	c.preferredShard = shard
	c.hasPreferredShard = true
}

func (c *Client) UnsetRequestedShard() {
	c.hasPreferredShard = false
}

func (c *Client) GetRequestedShard() (int, bool) {
	return c.preferredShard, c.hasPreferredShard
}

func (c *Client) SetShardingKey(key string) {
	c.shardingKey = key
}

func (c *Client) GetShardingKey() string {
	return c.shardingKey
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
		state:      gat.ClientActive,
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

func (c *Client) GetCurrentConn() gat.Connection {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.currentConn
}

func (c *Client) SetCurrentConn(conn gat.Connection) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = gat.ClientActive
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

	pool := c.gatling.GetPool(c.poolName)
	if pool == nil {
		return fmt.Errorf("pool '%s' not found", c.poolName)
	}

	// get user
	var user *config.User
	user = pool.GetUser(c.username)
	if user == nil {
		return fmt.Errorf("user '%s' not found", c.username)
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

	c.server = pool.WithUser(c.username)
	if c.server == nil {
		return fmt.Errorf("no pool for '%s'", c.username)
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
	cl := c.gatling.GetClient(gat.ClientID{
		PID:       p.Fields.ProcessKey,
		SecretKey: p.Fields.SecretKey,
	})
	if cl == nil {
		return errors.New("user not found")
	}
	conn := cl.GetCurrentConn()
	if conn == nil {
		return errors.New("not connected to a server")
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
	c.startRequest()
	return c.server.Describe(ctx, c, d)
}

func (c *Client) handle_execute(ctx context.Context, e *protocol.Execute) error {
	//log.Println("execute")
	c.status = 'T'
	c.startRequest()
	return c.server.Execute(ctx, c, e)
}

func (c *Client) handle_query(ctx context.Context, q *protocol.Query) error {
	parsed, err := parse.Parse(q.Fields.Query)
	if err != nil {
		return err
	}

	// we can handle empty queries here
	if len(parsed) == 0 {
		return c.Send(&protocol.EmptyQueryResponse{})
	}

	transaction := -1
	for idx, cmd := range parsed {
		var next int
		if idx+1 >= len(parsed) {
			next = len(q.Fields.Query)
		} else {
			next = parsed[idx+1].Index
		}

		cmdUpper := strings.ToUpper(cmd.Command)

		// not in transaction
		if transaction == -1 {
			switch cmdUpper {
			case "START":
				if len(cmd.Arguments) < 1 || strings.ToUpper(cmd.Arguments[0]) != "TRANSACTION" {
					break
				}
				fallthrough
			case "BEGIN":
				transaction = cmd.Index
			}
		}

		if transaction == -1 {
			// this is a simple query
			c.startRequest()
			err = c.handle_simple_query(ctx, q.Fields.Query[cmd.Index:next])
			if err != nil {
				return err
			}
		} else {
			// this command is part of a transaction
			switch cmdUpper {
			case "END":
				c.startRequest()
				err = c.handle_transaction(ctx, q.Fields.Query[transaction:next])
				if err != nil {
					return err
				}
				transaction = -1
			}
		}
	}

	if transaction != -1 {
		c.startRequest()
		err = c.handle_transaction(ctx, q.Fields.Query[transaction:])
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) handle_simple_query(ctx context.Context, q string) error {
	//log.Println("query:", q)
	c.startRequest()
	return c.server.SimpleQuery(ctx, c, q)
}

func (c *Client) handle_transaction(ctx context.Context, q string) error {
	//log.Println("transaction:", q)
	c.startRequest()
	return c.server.Transaction(ctx, c, q)
}

func (c *Client) handle_function(ctx context.Context, f *protocol.FunctionCall) error {
	c.startRequest()
	return c.server.CallFunction(ctx, c, f)
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
