package server

import (
	"bufio"
	"fmt"
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

	healthy bool

	log zlog.Logger

	mu sync.Mutex
}

func Dial(ctx context.Context,
	addr string,
	port uint16,
	user *config.User,
	db string, dbuser string, dbpass string,
) (*Server, error) {
	s := &Server{
		addr: addr,
		port: port,

		state: gat.ConnectionNew,

		boundPreparedStatments: make(map[string]*protocol.Parse),
		boundPortals:           make(map[string]*protocol.Bind),

		dbuser: dbuser,
		dbpass: dbpass,
	}
	var err error
	s.conn, err = net.Dial("tcp", fmt.Sprintf("%s:%d", addr, port))
	if err != nil {
		return nil, err
	}
	s.r = bufio.NewReader(s.conn)
	s.wr = bufio.NewWriter(s.conn)
	s.user = *user
	s.db = db

	s.log = log.With().
		Stringer("addr", s.conn.RemoteAddr()).
		Str("user", user.Name).
		Str("db", db).
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
	s.healthy = false
}

func (s *Server) healthCheck() {
	check := new(protocol.Query)
	check.Fields.Query = "select 1"
	err := s.writePacket(check)
	if err != nil {
		s.failHealthCheck(err)
		return
	}
	err = s.flush()
	if err != nil {
		s.failHealthCheck(err)
		return
	}

	// read until we get a ready for query
	for {
		var recv protocol.Packet
		recv, err = s.readPacket()
		if err != nil {
			s.failHealthCheck(err)
			return
		}

		switch r := recv.(type) {
		case *protocol.ReadyForQuery:
			if r.Fields.Status != 'I' {
				s.failHealthCheck(fmt.Errorf("expected server to be in command mode but it isn't"))
			}
			return
		case *protocol.DataRow, *protocol.RowDescription, *protocol.CommandComplete:
		default:
			s.failHealthCheck(fmt.Errorf("expected a Simple Query packet but server sent %#v", recv))
			return
		}
	}
}

func (s *Server) IsCloseNeeded() bool {
	return !s.healthy
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
	start.Fields.Parameters = []protocol.FieldsStartupMessageParameters{
		{
			Name:  "user",
			Value: s.dbuser,
		},
		{
			Name:  "database",
			Value: s.db,
		},
		{},
	}
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
			return nil
		}
	}
}

func (s *Server) forwardTo(client gat.Client, predicate func(pkt protocol.Packet) (forward bool, finish bool, err error)) error {
	var e error
	for {
		var rsp protocol.Packet
		rsp, err := s.readPacket()
		if err != nil {
			return err
		}
		//log.Printf("backend packet(%s) %+v", reflect.TypeOf(rsp), rsp)
		var forward, finish bool
		forward, finish, e = predicate(rsp)
		if forward && e == nil {
			e = client.Send(rsp)
		}
		if finish {
			return e
		}
	}
}

func (s *Server) writePacket(pkt protocol.Packet) error {
	//log.Printf("out %#v", pkt)
	_, err := pkt.Write(s.wr)
	return err
}

func (s *Server) flush() error {
	return s.wr.Flush()
}

func (s *Server) readPacket() (protocol.Packet, error) {
	p, err := protocol.ReadBackend(s.r)
	//log.Printf("in %#v", p)
	return p, err
}

func (s *Server) ensurePreparedStatement(client gat.Client, name string) error {
	// send prepared statement
	stmt := client.GetPreparedStatement(name)
	if stmt == nil {
		return &pg_error.Error{
			Severity: pg_error.Err,
			Code:     pg_error.ProtocolViolation,
			Message:  fmt.Sprintf("prepared statement '%s' does not exist", name),
		}
	}

	// test if prepared statement is the same
	if prev, ok := s.boundPreparedStatments[name]; ok {
		if reflect.DeepEqual(prev, stmt) {
			// we don't need to bind, we're good
			return nil
		}

		// there is a statement bound that needs to be unbound
		s.destructPreparedStatement(name)
	}

	s.boundPreparedStatments[name] = stmt

	// send prepared statement to server
	return s.writePacket(stmt)
}

func (s *Server) ensurePortal(client gat.Client, name string) error {
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

func (s *Server) Describe(client gat.Client, d *protocol.Describe) error {
	switch d.Fields.Which {
	case 'S': // prepared statement
		err := s.ensurePreparedStatement(client, d.Fields.Name)
		if err != nil {
			return err
		}
	case 'P': // portal
		err := s.ensurePortal(client, d.Fields.Name)
		if err != nil {
			return err
		}
	default:
		return &pg_error.Error{
			Severity: pg_error.Err,
			Code:     pg_error.ProtocolViolation,
			Message:  fmt.Sprintf("expected 'S' or 'P' for describe target, got '%c'", d.Fields.Which),
		}
	}

	// now we actually execute the thing the client wants
	err := s.writePacket(d)
	if err != nil {
		return err
	}
	err = s.writePacket(new(protocol.Sync))
	if err != nil {
		return err
	}
	err = s.flush()
	if err != nil {
		return err
	}

	return s.forwardTo(client, func(pkt protocol.Packet) (forward bool, finish bool, err error) {
		//log.Println("forward packet(%s) %+v", reflect.TypeOf(pkt), pkt)
		switch pkt.(type) {
		case *protocol.BindComplete, *protocol.ParseComplete:
		case *protocol.ReadyForQuery:
			finish = true
		default:
			forward = true
		}
		return
	})
}

func (s *Server) Execute(client gat.Client, e *protocol.Execute) error {
	err := s.ensurePortal(client, e.Fields.Name)
	if err != nil {
		return err
	}

	err = s.writePacket(e)
	if err != nil {
		return err
	}
	err = s.writePacket(new(protocol.Sync))
	if err != nil {
		return err
	}
	err = s.flush()
	if err != nil {
		return err
	}

	return s.forwardTo(client, func(pkt protocol.Packet) (forward bool, finish bool, err error) {
		//log.Println("forward packet(%s) %+v", reflect.TypeOf(pkt), pkt)
		switch pkt.(type) {
		case *protocol.BindComplete, *protocol.ParseComplete:
		case *protocol.ReadyForQuery:
			finish = true
		default:
			forward = true
		}
		return
	})
}

func (s *Server) SimpleQuery(ctx context.Context, client gat.Client, query string) error {
	// send to server
	q := new(protocol.Query)
	q.Fields.Query = query
	err := s.writePacket(q)
	if err != nil {
		return err
	}
	err = s.flush()
	if err != nil {
		return err
	}

	// this function seems wild but it has to be the way it is so we read the whole response, even if the
	// client fails midway
	// read responses
	return s.forwardTo(client, func(pkt protocol.Packet) (forward bool, finish bool, err error) {
		//log.Printf("forwarding pkt pkt(%s): %+v ", reflect.TypeOf(pkt), pkt)
		switch pkt.(type) {
		case *protocol.ReadyForQuery:
			// all ReadyForQuery packets end a simple query, regardless of type
			finish = true
		case *protocol.CopyInResponse:
			_ = client.Send(pkt)
			err = s.CopyIn(ctx, client)
		default:
			forward = true
		}
		return
	})
}

func (s *Server) Transaction(ctx context.Context, client gat.Client, query string) error {
	q := new(protocol.Query)
	q.Fields.Query = query
	err := s.writePacket(q)
	if err != nil {
		return err
	}
	err = s.flush()
	if err != nil {
		return err
	}
	return s.forwardTo(client, func(pkt protocol.Packet) (forward bool, finish bool, err error) {
		//log.Printf("got server pkt pkt(%s): %+v ", reflect.TypeOf(pkt), pkt)
		switch p := pkt.(type) {
		case *protocol.ReadyForQuery:
			// all ReadyForQuery packets end a simple query, regardless of type
			if p.Fields.Status != 'I' {
				// send to client and wait for next query
				err = client.Send(pkt)

				if err == nil {
					err = client.Flush()
					if err == nil {
						select {
						case r := <-client.Recv():
							//log.Printf("got client pkt pkt(%s): %+v", reflect.TypeOf(r), r)
							switch r.(type) {
							case *protocol.Query:
								//forward to server
								_ = s.writePacket(r)
								_ = s.flush()
							default:
								err = fmt.Errorf("expected an error in transaction state but got something else")
							}
						case <-ctx.Done():
							err = ctx.Err()
						}
					}
				}

				if err != nil {
					end := new(protocol.Query)
					end.Fields.Query = "END"
					_ = s.writePacket(end)
					_ = s.flush()
				}
			} else {
				finish = true
			}
		case *protocol.CopyInResponse:
			_ = client.Send(pkt)
			err = s.CopyIn(ctx, client)
		default:
			forward = true
		}
		return
	})
}

func (s *Server) CopyIn(ctx context.Context, client gat.Client) error {
	err := client.Flush()
	if err != nil {
		return err
	}
	for {
		// detect a disconneted /hanging client by waiting 30 seoncds, else timeout
		// otherwise, just keep reading packets until a done or error is received
		cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		var pkt protocol.Packet
		// receive a packet, or done if the ctx gets canceled
		select {
		case pkt = <-client.Recv():
		case <-cctx.Done():
			_ = s.writePacket(new(protocol.CopyFail))
			_ = s.flush()
			return cctx.Err()
		}
		cancel()
		err = s.writePacket(pkt)
		if err != nil {
			return err
		}

		switch pkt.(type) {
		case *protocol.CopyDone, *protocol.CopyFail:
			// don't error on copyfail because the client is the one that errored, it already knows
			return s.flush()
		}
	}
}

func (s *Server) CallFunction(client gat.Client, payload *protocol.FunctionCall) error {
	err := s.writePacket(payload)
	if err != nil {
		return err
	}
	err = s.flush()
	if err != nil {
		return err
	}
	// read responses
	return s.forwardTo(client, func(pkt protocol.Packet) (forward bool, finish bool, err error) {
		switch pkt.(type) {
		case *protocol.ReadyForQuery: // status 'I' should only be encountered here
			finish = true
		default:
			forward = true
		}
		return
	})
}

func (s *Server) Close(ctx context.Context) error {
	<-ctx.Done()
	return nil
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
