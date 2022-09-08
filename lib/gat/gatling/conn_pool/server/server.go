package server

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
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
	addr   string
	remote net.Addr
	conn   net.Conn
	r      *bufio.Reader
	wr     io.Writer
	bufwr  *bufio.Writer

	buf bytes.Buffer

	server_info []*protocol.ParameterStatus

	process_id int32
	secret_key int32

	bad    bool
	in_txn bool

	connected_at     time.Time
	stats            any // TODO: stats
	application_name string

	last_activity time.Time

	db     string
	dbuser string
	dbpass string
	user   config.User

	log zlog.Logger
}

func Dial(ctx context.Context,
	addr string,
	user *config.User,
	db string, dbuser string, dbpass string,
	stats any,
) (*Server, error) {
	s := &Server{
		addr:   addr,
		dbuser: dbuser,
		dbpass: dbpass,
	}
	var err error
	s.conn, err = net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	s.remote = s.conn.RemoteAddr()
	s.r = bufio.NewReader(s.conn)
	s.wr = s.conn
	s.bufwr = bufio.NewWriter(s.wr)
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
	cancel.Fields.ProcessKey = s.process_id
	cancel.Fields.SecretKey = s.secret_key
	_, err = cancel.Write(conn)
	_ = conn.Close()
	return err
}

func (s *Server) GetServerInfo() []*protocol.ParameterStatus {
	return s.server_info
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
	_, err := start.Write(s.wr)
	if err != nil {
		return err
	}
	return nil
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
		pkt, err = protocol.ReadBackend(s.r)
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
				func() {
					rsp := new(protocol.AuthenticationResponse)
					buf := bufpool.Get(len(scrm.Name()) + 1 + 4 + len(bts))
					buf.Reset()
					defer bufpool.Put(buf)
					_, _ = protocol.WriteString(buf, scrm.Name())
					_, _ = protocol.WriteInt32(buf, int32(len(bts)))
					buf.Write(bts)
					rsp.Fields.Data = buf.Bytes()
					_, err = rsp.Write(s.wr)
				}()
				if err != nil {
					return err
				}
			case 11: // SASL_CONTINUE
				s.log.Debug().Str("method", "scram256").Msg("sasl continue")
				var bts []byte
				_, bts, err = sm.Next(ctx, p.Fields.SASLChallenge)

				rsp := new(protocol.AuthenticationResponse)
				rsp.Fields.Data = bts
				_, err = rsp.Write(s.wr)
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
			s.server_info = append(s.server_info, p)
		case *protocol.BackendKeyData:
			s.process_id = p.Fields.ProcessID
			s.secret_key = p.Fields.SecretKey
		case *protocol.ReadyForQuery:
			s.last_activity = time.Now()
			s.connected_at = time.Now().UTC()
			s.bad = false
			s.in_txn = false
			return nil
		}
	}
}

func (s *Server) forwardTo(client gat.Client, predicate func(pkt protocol.Packet) (forward bool, finish bool)) error {
	for {
		var rsp protocol.Packet
		rsp, err := protocol.ReadBackend(s.r)
		if err != nil {
			return err
		}
		forward, finish := predicate(rsp)
		if forward {
			err = client.Send(rsp)
			if err != nil {
				return err
			}
		}
		if finish {
			return nil
		}
	}
}

func (s *Server) writePacket(pkt protocol.Packet) error {
	_, err := pkt.Write(s.bufwr)
	if err != nil {
		s.bufwr.Reset(s.wr)
		return err
	}
	return s.bufwr.Flush()
}

func (s *Server) SimpleQuery(ctx context.Context, client gat.Client, query string) error {
	// send to server
	q := new(protocol.Query)
	q.Fields.Query = query
	err := s.writePacket(q)
	if err != nil {
		return err
	}
	if strings.Contains(query, "pg_sleep") {
		go func() {
			time.Sleep(1 * time.Second)
			log.Println("cancel: ", s.Cancel())
		}()
	}
	// this function seems wild but it has to be the way it is so we read the whole response, even if the
	// client fails midway
	// read responses
	e := s.forwardTo(client, func(pkt protocol.Packet) (forward bool, finish bool) {
		//log.Printf("forwarding pkt pkt(%s): %+v ", reflect.TypeOf(pkt), pkt)
		switch pkt.(type) {
		case *protocol.ReadyForQuery:
			// all ReadyForQuery packets end a simple query, regardless of type
			return err == nil, true
		case *protocol.CopyInResponse:
			err = client.Send(pkt)
			if err != nil {
				return false, false
			}
			err = s.CopyIn(ctx, client)
			if err != nil {
				return false, false
			}
			return false, false
		default:
			return err == nil, false
		}
	})
	if e != nil {
		return e
	}
	return err
}

func (s *Server) Transaction(ctx context.Context, client gat.Client, query string) error {
	q := new(protocol.Query)
	q.Fields.Query = query
	err := s.writePacket(q)
	if err != nil {
		return err
	}
	e := s.forwardTo(client, func(pkt protocol.Packet) (forward bool, finish bool) {
		//log.Printf("got server pkt pkt(%s): %+v ", reflect.TypeOf(pkt), pkt)
		switch p := pkt.(type) {
		case *protocol.ReadyForQuery:
			// all ReadyForQuery packets end a simple query, regardless of type
			if p.Fields.Status != 'I' {
				// send to client and wait for next query
				if err == nil {
					err = client.Send(pkt)
				}

				if err == nil {
					select {
					case r := <-client.Recv():
						//log.Printf("got client pkt pkt(%s): %+v", reflect.TypeOf(r), r)
						switch r.(type) {
						case *protocol.Query:
							//forward to server
							_, _ = r.Write(s.bufwr)
							err = s.bufwr.Flush()
						default:
							err = fmt.Errorf("expected an error in transaction state but got something else")
						}
					case <-ctx.Done():
						err = ctx.Err()
					}
				}

				if err != nil {
					end := new(protocol.Query)
					end.Fields.Query = "END;"
					_, _ = end.Write(s.bufwr)
					_ = s.bufwr.Flush()
				}
			}
			return p.Fields.Status == 'I', p.Fields.Status == 'I'
		case *protocol.CopyInResponse:
			err = client.Send(pkt)
			if err != nil {
				return false, false
			}
			err = s.CopyIn(ctx, client)
			if err != nil {
				return false, false
			}
			return false, false
		default:
			return err == nil, false
		}
	})
	if e != nil {
		return e
	}
	return err
}

func (s *Server) CopyIn(ctx context.Context, client gat.Client) error {
	for {
		// detect a disconneted /hanging client by waiting 30 seoncds, else timeout
		// otherwise, just keep reading packets until a done or error is received
		cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		var pkt protocol.Packet
		// receive a packet, or done if the ctx gets canceled
		select {
		case pkt = <-client.Recv():
		case <-cctx.Done():
			_, _ = new(protocol.CopyFail).Write(s.bufwr)
			_ = s.bufwr.Flush()
			rfq := new(protocol.ReadyForQuery)
			rfq.Fields.Status = 'I'
			return client.Send(rfq)
		}
		cancel()
		err := s.writePacket(pkt)
		if err != nil {
			return err
		}

		switch p := pkt.(type) {
		case *protocol.CopyDone:
			return nil
		case *protocol.CopyFail:
			return errors.New(p.Fields.Cause)
		}
	}
}

func (s *Server) CallFunction(client gat.Client, payload *protocol.FunctionCall) error {
	_, err := payload.Write(s.wr)
	if err != nil {
		return err
	}
	// read responses
	return s.forwardTo(client, func(pkt protocol.Packet) (forward bool, finish bool) {
		switch r := pkt.(type) {
		case *protocol.ReadyForQuery:
			return true, r.Fields.Status == 'I'
		default:
			return true, false
		}
	})
}

func (s *Server) Close(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

//impl Drop for Server {
//    /// Try to do a clean shut down. Best effort because
//    /// the socket is in non-blocking mode, so it may not be ready
//    /// for a write.
//    fn drop(&mut self) {
//        self.stats
//            .server_disconnecting(self.process_id(), self.address.id);
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
//        let duration = now - self.connected_at;
//
//        info!(
//            "Server connection closed, session duration: {}",
//            crate::format_duration(&duration)
//        );
//    }
//}
