package server

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"reflect"
	"sync"
	"time"

	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/protocol/pg_error"

	"gfx.cafe/gfx/pggat/lib/gat/protocol"
	"gfx.cafe/gfx/pggat/lib/util/slices"
	"gfx.cafe/util/go/bufpool"

	"gfx.cafe/gfx/pggat/lib/auth/sasl"
	"gfx.cafe/gfx/pggat/lib/auth/scram"
	"gfx.cafe/gfx/pggat/lib/config"
	"git.tuxpa.in/a/zlog"
	"git.tuxpa.in/a/zlog/log"
	"golang.org/x/net/context"
)

type Server struct {
	addr string
	port uint16
	conn net.Conn
	r    *bufio.Reader
	wr   *bufio.Writer

	client gat.Client
	state  gat.ConnectionState

	options []protocol.FieldsStartupMessageParameters

	serverInfo []*protocol.ParameterStatus

	processId int32
	secretKey int32

	connectedAt  time.Time
	lastActivity time.Time

	boundPreparedStatments map[string]*protocol.Parse
	boundPortals           map[string]*protocol.Bind

	// constants
	db     string
	dbuser string
	dbpass string
	user   config.User

	awaitingSync  bool
	readyForQuery bool
	copying       bool

	log zlog.Logger

	closed chan struct{}
	mu     sync.Mutex
}

func Dial(ctx context.Context, options []protocol.FieldsStartupMessageParameters, user *config.User, shard *config.Shard, server *config.Server) (gat.Connection, error) {
	s := &Server{
		addr: server.Host,
		port: server.Port,

		state: gat.ConnectionNew,

		options: options,

		boundPreparedStatments: make(map[string]*protocol.Parse),
		boundPortals:           make(map[string]*protocol.Bind),

		dbuser: server.Username,
		dbpass: server.Password,

		closed: make(chan struct{}),
	}
	var err error
	s.conn, err = net.Dial("tcp", fmt.Sprintf("%s:%d", server.Host, server.Port))
	if err != nil {
		return nil, err
	}
	s.r = bufio.NewReader(s.conn)
	s.wr = bufio.NewWriter(s.conn)
	s.user = *user
	s.db = shard.Database

	s.log = log.With().
		Stringer("addr", s.conn.RemoteAddr()).
		Str("user", user.Name).
		Str("db", shard.Database).
		Logger()
	return s, s.connect(ctx)
}

func (s *Server) Cancel() error {
	conn, err := net.Dial("tcp", s.addr)
	if err != nil {
		return err
	}
	cancel := new(protocol.StartupMessage)
	cancel.Fields.ProtocolVersionNumber = 80877102
	cancel.Fields.ProcessKey = s.processId
	cancel.Fields.SecretKey = s.secretKey
	_, err = cancel.Write(conn)
	_ = conn.Close()
	return err
}

func (s *Server) GetDatabase() string {
	return s.db
}

func (s *Server) GetState() gat.ConnectionState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

func (s *Server) GetHost() string {
	return s.addr
}

func (s *Server) GetPort() int {
	return int(s.port)
}

func (s *Server) GetAddress() net.Addr {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conn.RemoteAddr()
}

func (s *Server) GetLocalAddress() net.Addr {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conn.LocalAddr()
}

func (s *Server) GetConnectTime() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.connectedAt
}

func (s *Server) GetRequestTime() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastActivity
}

func (s *Server) failHealthCheck(err error) {
	log.Println("Server failed a health check!!!!", err)
	_ = s.Close()
}

func (s *Server) healthCheck() {
	if !s.readyForQuery {
		s.failHealthCheck(errors.New("expected server to be ready for query"))
	}
}

func (s *Server) IsCloseNeeded() bool {
	select {
	case <-s.closed:
		return true
	default:
		return !s.readyForQuery
	}
}

func (s *Server) GetClient() gat.Client {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.client
}

func (s *Server) SetClient(client gat.Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastActivity = time.Now()
	if client != nil {
		s.state = gat.ConnectionActive
	} else {
		// client no longer needs this connection, perform a health check
		s.healthCheck()
		s.state = gat.ConnectionIdle
	}
	s.client = client
}

func (s *Server) GetRemotePid() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return int(s.processId)
}

func (s *Server) GetTLS() string {
	return "" // TODO
}

func (s *Server) GetServerInfo() []*protocol.ParameterStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.serverInfo
}

func (s *Server) startup(ctx context.Context) error {
	s.log.Debug().Msg("sending startup")
	start := new(protocol.StartupMessage)
	start.Fields.ProtocolVersionNumber = 196608
	start.Fields.Parameters = append(
		s.options,
		protocol.FieldsStartupMessageParameters{
			Name:  "user",
			Value: s.dbuser,
		},
		protocol.FieldsStartupMessageParameters{
			Name:  "database",
			Value: s.db,
		},
		protocol.FieldsStartupMessageParameters{},
	)
	err := s.writePacket(start)
	if err != nil {
		return err
	}
	return s.flush()
}

func (s *Server) connect(ctx context.Context) error {
	err := s.startup(ctx)
	if err != nil {
		return err
	}
	s.log.Debug().Msg("beginning connection")
	var scrm sasl.Mechanism
	var sm sasl.StateMachine
	for {
		var pkt protocol.Packet
		pkt, err = s.readPacket()
		if err != nil {
			return err
		}
		switch p := pkt.(type) {
		case *protocol.Authentication:
			switch p.Fields.Code {
			case 5: //MD5_ENCRYPTED_PASSWORD
			case 0: // AUTH SUCCESS
			case 10: // SASL
				if slices.Contains(p.Fields.SASLMechanism, scram.SHA256.Name()) {
					s.log.Debug().Str("method", "scram256").Msg("valid protocol")
				} else {
					return fmt.Errorf("unsupported scram version: %s", p.Fields.SASLMechanism)
				}
				scrm, err = scram.Mechanism(scram.SHA256, s.dbuser, s.dbpass)
				if err != nil {
					return err
				}
				var bts []byte
				sm, bts, err = scrm.Start(ctx)
				if err != nil {
					return err
				}

				rsp := new(protocol.AuthenticationResponse)
				buf := bufpool.Get(len(scrm.Name()) + 1 + 4 + len(bts))
				buf.Reset()
				_, _ = protocol.WriteString(buf, scrm.Name())
				_, _ = protocol.WriteInt32(buf, int32(len(bts)))
				buf.Write(bts)
				rsp.Fields.Data = buf.Bytes()
				err = s.writePacket(rsp)
				bufpool.Put(buf)
				if err != nil {
					return err
				}
				err = s.flush()
				if err != nil {
					return err
				}
			case 11: // SASL_CONTINUE
				s.log.Debug().Str("method", "scram256").Msg("sasl continue")
				var bts []byte
				_, bts, err = sm.Next(ctx, p.Fields.SASLChallenge)

				rsp := new(protocol.AuthenticationResponse)
				rsp.Fields.Data = bts
				err = s.writePacket(rsp)
				if err != nil {
					return err
				}
				err = s.flush()
				if err != nil {
					return err
				}
			case 12: // SASL_FINAL
				s.log.Debug().Str("method", "scram256").Msg("sasl final")
				var done bool
				done, _, err = sm.Next(ctx, p.Fields.SASLAdditionalData)
				if err != nil {
					return err
				}
				if !done {
					return fmt.Errorf("sasl authentication failed")
				}

				s.log.Debug().Str("method", "scram256").Msg("sasl success")
			}
		case *protocol.ErrorResponse:
			pgErr := new(pg_error.Error)
			pgErr.Read(p)
			return pgErr
		case *protocol.ParameterStatus:
			s.serverInfo = append(s.serverInfo, p)
		case *protocol.BackendKeyData:
			s.processId = p.Fields.ProcessID
			s.secretKey = p.Fields.SecretKey
		case *protocol.ReadyForQuery:
			s.lastActivity = time.Now()
			s.connectedAt = time.Now().UTC()
			s.state = "idle"
			s.readyForQuery = true
			return nil
		}
	}
}

func (s *Server) writePacket(pkt protocol.Packet) error {
	//log.Printf("out %#v", pkt)
	select {
	case <-s.closed:
		return io.ErrClosedPipe
	default:
		_, err := pkt.Write(s.wr)
		if err != nil {
			_ = s.Close()
		}
		return err
	}
}

func (s *Server) flush() error {
	select {
	case <-s.closed:
		return io.ErrClosedPipe
	default:
		err := s.wr.Flush()
		if err != nil {
			_ = s.Close()
		}
		return err
	}
}

func (s *Server) readPacket() (protocol.Packet, error) {
	p, err := protocol.ReadBackend(s.r)
	if err != nil {
		_ = s.Close()
	}
	//log.Printf("in %#v", p)
	return p, err
}

func (s *Server) stabilize() {
	if s.readyForQuery {
		return
	}
	//log.Println("connection is unstable, attempting to restabilize it")
	if s.copying {
		//log.Println("failing copy")
		s.copying = false
		err := s.writePacket(new(protocol.CopyFail))
		if err != nil {
			return
		}
	}
	if s.awaitingSync {
		//log.Println("syncing")
		s.awaitingSync = false
		err := s.writePacket(new(protocol.Sync))
		if err != nil {
			return
		}
	}
	err := s.flush()
	if err != nil {
		return
	}

	for {
		var pkt protocol.Packet
		pkt, err = s.readPacket()
		if err != nil {
			return
		}

		//log.Printf("received %+v", pkt)

		switch pk := pkt.(type) {
		case *protocol.ReadyForQuery:
			if pk.Fields.Status == 'I' {
				s.readyForQuery = true
				return
			} else {
				query := new(protocol.Query)
				query.Fields.Query = "end"
				err = s.writePacket(query)
				if err != nil {
					return
				}
				err = s.flush()
				if err != nil {
					return
				}
			}
		case *protocol.CopyInResponse, *protocol.CopyBothResponse:
			fail := new(protocol.CopyFail)
			err = s.writePacket(fail)
			if err != nil {
				return
			}
			err = s.flush()
			if err != nil {
				return
			}
		}
	}
}

func (s *Server) ensurePreparedStatement(client gat.Client, name string) error {
	s.awaitingSync = true
	// send prepared statement
	stmt := client.GetPreparedStatement(name)
	if stmt == nil {
		return &pg_error.Error{
			Severity: pg_error.Err,
			Code:     pg_error.ProtocolViolation,
			Message:  fmt.Sprintf("prepared statement '%s' does not exist", name),
		}
	}

	if name != "" {
		// test if prepared statement is the same
		if prev, ok := s.boundPreparedStatments[name]; ok {
			if reflect.DeepEqual(prev, stmt) {
				// we don't need to bind, we're good
				return nil
			}

			// there is a statement bound that needs to be unbound
			s.destructPreparedStatement(name)
		}
	}

	s.boundPreparedStatments[name] = stmt

	// send prepared statement to server
	return s.writePacket(stmt)
}

func (s *Server) ensurePortal(client gat.Client, name string) error {
	s.awaitingSync = true
	portal := client.GetPortal(name)
	if portal == nil {
		return &pg_error.Error{
			Severity: pg_error.Err,
			Code:     pg_error.ProtocolViolation,
			Message:  fmt.Sprintf("portal '%s' does not exist", name),
		}
	}

	err := s.ensurePreparedStatement(client, portal.Fields.PreparedStatement)
	if err != nil {
		return err
	}

	if name != "" {
		if prev, ok := s.boundPortals[name]; ok {
			if reflect.DeepEqual(prev, portal) {
				return nil
			}
		}
	}

	s.boundPortals[name] = portal
	return s.writePacket(portal)
}

func (s *Server) destructPreparedStatement(name string) {
	if name == "" {
		return
	}
	delete(s.boundPreparedStatments, name)
	query := new(protocol.Query)
	query.Fields.Query = fmt.Sprintf("DEALLOCATE \"%s\"", name)
	_ = s.writePacket(query)
	_ = s.flush()
	// await server ready
	for {
		r, _ := s.readPacket()
		if _, ok := r.(*protocol.ReadyForQuery); ok {
			return
		}
	}
}

func (s *Server) destructPortal(name string) {
	portal, ok := s.boundPortals[name]
	if !ok {
		return
	}
	delete(s.boundPortals, name)
	s.destructPreparedStatement(portal.Fields.PreparedStatement)
}

func (s *Server) handleRecv(client gat.Client, packet protocol.Packet) error {
	switch pkt := packet.(type) {
	case *protocol.FunctionCall, *protocol.Query:
		err := s.writePacket(packet)
		if err != nil {
			return err
		}
		err = s.flush()
		if err != nil {
			return err
		}
	case *protocol.Describe:
		s.awaitingSync = true
		switch pkt.Fields.Which {
		case 'S': // prepared statement
			err := s.ensurePreparedStatement(client, pkt.Fields.Name)
			if err != nil {
				return err
			}
		case 'P': // portal
			err := s.ensurePortal(client, pkt.Fields.Name)
			if err != nil {
				return err
			}
		default:
			return &pg_error.Error{
				Severity: pg_error.Err,
				Code:     pg_error.ProtocolViolation,
				Message:  fmt.Sprintf("expected 'S' or 'P' for describe target, got '%c'", pkt.Fields.Which),
			}
		}

		// now we actually execute the thing the client wants
		err := s.writePacket(packet)
		if err != nil {
			return err
		}
	case *protocol.Execute:
		s.awaitingSync = true
		err := s.ensurePortal(client, pkt.Fields.Name)
		if err != nil {
			return err
		}

		err = s.writePacket(pkt)
		if err != nil {
			return err
		}
	case *protocol.Sync:
		s.awaitingSync = false
		err := s.writePacket(packet)
		if err != nil {
			return err
		}
		err = s.flush()
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("don't know how to handle %T", packet)
	}
	return nil
}

func (s *Server) sendAndLink(ctx context.Context, client gat.Client, initial protocol.Packet) error {
	s.readyForQuery = false
	err := s.handleRecv(client, initial)
	if err != nil {
		return err
	}
	err = s.awaitSync(ctx, client)
	if err != nil {
		return err
	}
	return s.link(ctx, client)
}

func (s *Server) link(ctx context.Context, client gat.Client) error {
	defer s.stabilize()
	for {
		pkt, err := s.readPacket()
		if err != nil {
			return err
		}

		switch p := pkt.(type) {
		case *protocol.BindComplete, *protocol.ParseComplete:
			// ignore, it is because we bound stuff
		case *protocol.ReadyForQuery:
			if p.Fields.Status == 'I' {
				// this client is done
				s.awaitingSync = false
				s.copying = false
				s.readyForQuery = true
				return nil
			}

			err = client.Send(p)
			if err != nil {
				return err
			}
			err = client.Flush()
			if err != nil {
				return err
			}

			err = s.handleClientPacket(ctx, client)
			if err != nil {
				return err
			}
			err = s.awaitSync(ctx, client)
			if err != nil {
				return err
			}
		case *protocol.CopyInResponse, *protocol.CopyBothResponse:
			err = client.Send(p)
			if err != nil {
				return err
			}
			err = client.Flush()
			if err != nil {
				return err
			}
			err = s.CopyIn(ctx, client)
			if err != nil {
				return err
			}
		default:
			err = client.Send(p)
			if err != nil {
				return err
			}
		}
	}
}

func (s *Server) handleClientPacket(ctx context.Context, client gat.Client) error {
	select {
	case pkt := <-client.Recv():
		return s.handleRecv(client, pkt)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Server) awaitSync(ctx context.Context, client gat.Client) error {
	for s.awaitingSync {
		err := s.handleClientPacket(ctx, client)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) Describe(ctx context.Context, client gat.Client, d *protocol.Describe) error {
	return s.sendAndLink(ctx, client, d)
}

func (s *Server) Execute(ctx context.Context, client gat.Client, e *protocol.Execute) error {
	return s.sendAndLink(ctx, client, e)
}

func (s *Server) SimpleQuery(ctx context.Context, client gat.Client, query string) error {
	// send to server
	q := new(protocol.Query)
	q.Fields.Query = query
	return s.sendAndLink(ctx, client, q)
}

func (s *Server) Transaction(ctx context.Context, client gat.Client, query string) error {
	q := new(protocol.Query)
	q.Fields.Query = query
	return s.sendAndLink(ctx, client, q)
}

func (s *Server) CopyIn(ctx context.Context, client gat.Client) error {
	s.copying = true
	err := client.Flush()
	if err != nil {
		return err
	}
	for {
		var pkt protocol.Packet
		// receive a packet, or done if the ctx gets canceled
		select {
		case pkt = <-client.Recv():
		case <-ctx.Done():
			return ctx.Err()
		}
		err = s.writePacket(pkt)
		if err != nil {
			return err
		}

		switch pkt.(type) {
		case *protocol.CopyDone, *protocol.CopyFail:
			s.copying = false
			// don't error on copyfail because the client is the one that errored, it already knows
			return s.flush()
		}
	}
}

func (s *Server) CallFunction(ctx context.Context, client gat.Client, payload *protocol.FunctionCall) error {
	return s.sendAndLink(ctx, client, payload)
}

func (s *Server) Close() error {
	select {
	case <-s.closed:
		return io.ErrClosedPipe
	default:
		s.readyForQuery = false
		close(s.closed)
		_ = s.writePacket(&protocol.Close{})
		return s.conn.Close()
	}
}

var _ gat.Connection = (*Server)(nil)

//impl Drop for Server {
//    /// Try to do a clean shut down. Best effort because
//    /// the socket is in non-blocking mode, so it may not be ready
//    /// for a write.
//    fn drop(&mut self) {
//        self.stats
//            .server_disconnecting(self.processId(), self.address.id);
//
//        let mut bytes = BytesMut::with_capacity(4);
//        bytes.put_u8(b'X');
//        bytes.put_i32(4);
//
//        match self.write.try_write(&bytes) {
//            Ok(_) => (),
//            Err(_) => debug!("Dirty shutdown"),
//        };
//
//        // Should not matter.
//        self.bad = true;
//
//        let now = chrono::offset::Utc::now().naive_utc();
//        let duration = now - self.connectedAt;
//
//        info!(
//            "Server connection closed, session duration: {}",
//            crate::format_duration(&duration)
//        );
//    }
//}
