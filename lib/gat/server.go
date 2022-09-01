package gat

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
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

	server_info []byte

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
	s.server_info = []byte{}
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
	//TODO: grow / bufpool
	buf := new(bytes.Buffer)
	err := binary.Write(buf, ENDIAN, int32(196608))
	if err != nil {
		return err
	}
	_, err = buf.WriteString("user\000")
	if err != nil {
		return err
	}
	_, err = buf.WriteString(s.user.Name)
	if err != nil {
		return err
	}
	err = buf.WriteByte(0)
	if err != nil {
		return err
	}
	_, err = buf.WriteString("database\000")
	if err != nil {
		return err
	}
	_, err = buf.WriteString(s.db)
	if err != nil {
		return err
	}
	err = buf.WriteByte(0)
	if err != nil {
		return err
	}
	err = buf.WriteByte(0)
	if err != nil {
		return err
	}
	//TODO: grow / bufpool
	buf2 := new(bytes.Buffer)
	buf2.Grow(buf.Len() + 4)
	err = binary.Write(buf2, ENDIAN, int32(buf.Len())+4)
	if err != nil {
		return err
	}
	buf2.Write(buf.Bytes())
	_, err = s.wr.Write(buf2.Bytes())
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
		code, err := s.r.ReadByte()
		if err != nil {
			return err
		}
		var msglen32 int32
		err = binary.Read(s.r, ENDIAN, &msglen32)
		if err != nil {
			return err
		}
		msglen := int(msglen32)
		s.log.Debug().Str("code", string(code)).Int("len", msglen).Msg("startup msg")
		switch code {
		case 'R':
			var auth_code int32
			err = binary.Read(s.r, ENDIAN, &auth_code)
			if err != nil {
				return err
			}
			//TODO: move these into constants
			switch auth_code {
			case 5: //MD5_ENCRYPTED_PASSWORD
				salt := make([]byte, 4)
				_, err := io.ReadFull(s.r, salt)
				if err != nil {
					return err
				}
			case 0: // AUTH SUCCESS
			case 10: // SASL
				s.log.Debug().Msg("starting sasl auth")
				sasl_len := (msglen) - 8
				sasl_auth := make([]byte, sasl_len)
				_, err := io.ReadFull(s.r, sasl_auth)
				if err != nil {
					return err
				}
				sasl_type := string(sasl_auth[:len(sasl_auth)-2])
				switch sasl_type {
				case scram.SHA256.Name():
					s.log.Debug().Str("method", "scram256").Msg("valid protocol")
					scrm, err = scram.Mechanism(scram.SHA256, s.user.Name, s.user.Password)
					if err != nil {
						return err
					}
					var bts []byte
					sm, bts, err = scrm.Start(ctx)
					if err != nil {
						return err
					}
					resp := new(bytes.Buffer)
					//TODO: grow buffer or use bufpool
					err = resp.WriteByte('p')
					if err != nil {
						return err
					}
					err = binary.Write(resp, ENDIAN, int32(4+len(scrm.Name())+1+4+len(bts)))
					if err != nil {
						return err
					}
					// write header
					_, err = resp.WriteString(scrm.Name())
					if err != nil {
						return err
					}
					err = resp.WriteByte(0)
					if err != nil {
						return err
					}
					// write length
					err = binary.Write(resp, ENDIAN, int32(len(bts)))
					if err != nil {
						return err
					}
					_, err = resp.Write(bts)
					if err != nil {
						return err
					}
					_, err = resp.WriteTo(s.wr)
					if err != nil {
						return err
					}
				default:
					return fmt.Errorf("unsupported scram version: %s", sasl_type)
				}
			case 11: // SASL_CONTINUE
				s.log.Debug().Str("method", "scram256").Msg("sasl continue")
				sasl_data := make([]byte, msglen-8)
				_, err := io.ReadFull(s.r, sasl_data)
				if err != nil {
					return err
				}
				_, bts, err := sm.Next(ctx, sasl_data)
				if err != nil {
					return err
				}
				sbuf := new(bytes.Buffer)
				//TODO: grow buffer or use bufpool
				sbuf.WriteByte('p')
				if err != nil {
					return err
				}
				err = binary.Write(sbuf, ENDIAN, int32(4+len(bts)))
				if err != nil {
					return err
				}
				_, err = sbuf.Write(bts)
				if err != nil {
					return err
				}
				_, err = sbuf.WriteTo(s.wr)
				if err != nil {
					return err
				}
			case 12: // SASL_FINAL
				s.log.Debug().Str("method", "scram256").Msg("sasl final")
				sasl_final := make([]byte, msglen-8)
				_, err := io.ReadFull(s.r, sasl_final)
				if err != nil {
					return err
				}
				done, _, err := sm.Next(ctx, sasl_final)
				if err != nil {
					return err
				}
				if !done {
					return fmt.Errorf("sasl authentication failed")
				}

				s.log.Debug().Str("method", "scram256").Msg("sasl success")
			}
		case 'E':
			var error_code int32
			err = binary.Read(s.r, ENDIAN, &error_code)
			if err != nil {
				return err
			}
			switch error_code {
			case 0: //msg terminator
			default:
				err_data := make([]byte, msglen-4-1)
				_, err := io.ReadFull(s.r, err_data)
				if err != nil {
					return err
				}
				return fmt.Errorf("pg error: %s", string(err_data))
			}
		case 'S':
			param_data := make([]byte, msglen-4)
			_, err := io.ReadFull(s.r, param_data)
			if err != nil {
				return err
			}
			s.server_info = append(s.server_info, 'S')
			s.server_info = ENDIAN.AppendUint32(s.server_info, uint32(msglen))
			s.server_info = append(s.server_info, param_data...)
		case 'K':
			err = binary.Read(s.r, ENDIAN, &s.process_id)
			if err != nil {
				return err
			}
			err = binary.Read(s.r, ENDIAN, &s.secret_key)
			if err != nil {
				return err
			}
		case 'Z':
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
