package gat

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
	"gfx.cafe/gfx/pggat/lib/util"
	"io"
	"net"
	"time"

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

	bad bool

	in_txn bool
	csm    map[ClientKey]ClientInfo

	connected_at time.Time

	stats any // TODO: stats

	application_name string

	last_activity time.Time

	db   string
	user config.User

	log zlog.Logger
}

var ENDIAN = binary.BigEndian

func DialServer(ctx context.Context, addr string, user *config.User, db string, csm map[ClientInfo]ClientInfo, stats any) (*Server, error) {
	s := &Server{}
	var err error
	s.conn, err = net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	s.remote = s.conn.RemoteAddr()
	s.r = bufio.NewReader(s.conn)
	s.wr = s.conn
	s.server_info = []*protocol.ParameterStatus{}
	s.user = *user
	s.db = db

	s.log = log.With().
		Stringer("addr", s.conn.RemoteAddr()).
		Str("user", user.Name).
		Str("db", db).
		Logger()
	return s, s.connect(ctx)
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
				if util.Contains(p.Fields.SASLMechanism, scram.SHA256.Name()) {
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

				rsp := new(protocol.AuthenticationResponse)
				buf := new(bytes.Buffer)
				_, _ = protocol.WriteString(buf, scrm.Name())
				_, _ = protocol.WriteInt32(buf, int32(len(bts)))
				buf.Write(bts)
				rsp.Fields.Data = buf.Bytes()
				_, err = rsp.Write(s.wr)
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
			pgErr := new(PostgresError)
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

// TODO: implement drop - we should rename to close.
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
