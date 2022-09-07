package server

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"gfx.cafe/gfx/pggat/lib/gat/protocol/pg_error"
	"io"
	"net"
	"time"

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
	remote net.Addr
	conn   net.Conn
	r      *bufio.Reader
	wr     io.Writer

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

	db   string
	user config.User

	log zlog.Logger
}

var ENDIAN = binary.BigEndian

func Dial(ctx context.Context, addr string, user *config.User, db string, stats any) (*Server, error) {
	s := &Server{}
	var err error
	s.conn, err = net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	s.remote = s.conn.RemoteAddr()
	s.r = bufio.NewReader(s.conn)
	s.wr = s.conn
	s.user = *user
	s.db = db

	s.log = log.With().
		Stringer("addr", s.conn.RemoteAddr()).
		Str("user", user.Name).
		Str("db", db).
		Logger()
	return s, s.connect(ctx)
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
			Value: s.user.Name,
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
				s.log.Debug().Msg("starting sasl auth")
				if slices.Contains(p.Fields.SASLMechanism, scram.SHA256.Name()) {
					s.log.Debug().Str("method", "scram256").Msg("valid protocol")
				} else {
					return fmt.Errorf("unsupported scram version: %s", p.Fields.SASLMechanism)
				}

				scrm, err = scram.Mechanism(scram.SHA256, s.user.Name, s.user.Password)
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

func (s *Server) Query(query string, rep chan<- protocol.Packet) error {
	// send to server
	q := new(protocol.Query)
	q.Fields.Query = query
	_, err := q.Write(s.wr)
	if err != nil {
		return err
	}

	// read responses
	for {
		var rsp protocol.Packet
		rsp, err = protocol.ReadBackend(s.r)
		if err != nil {
			return err
		}
		switch r := rsp.(type) {
		case *protocol.ReadyForQuery:
			if r.Fields.Status == 'I' {
				rep <- rsp
				return nil
			}
		case *protocol.CopyInResponse, *protocol.CopyOutResponse, *protocol.CopyBothResponse:
			return fmt.Errorf("unsuported")
		}
		rep <- rsp
	}
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
