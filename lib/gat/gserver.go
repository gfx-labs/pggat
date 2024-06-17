package gat

import (
	"crypto/tls"
	"errors"
	"io"
	"net"

	"gfx.cafe/gfx/pggat/lib/bouncer/frontends/v0"
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/perror"
	"go.uber.org/zap"
)

func (T *Server) gaccept(listener *GListener, conn *fed.Conn) {
	defer func() {
		_ = conn.Close()
	}()

	var tlsConfig *tls.Config
	if listener.ssl != nil {
		tlsConfig = listener.ssl.ServerTLSConfig()
	}

	var cancelKey fed.BackendKey
	var isCanceling bool
	var err error
	cancelKey, isCanceling, err = frontends.Accept(conn, tlsConfig)
	if err != nil {
		if !errors.Is(err, io.EOF) {
			T.log.Warn("error accepting client", zap.Error(err))
		}
		return
	}

	if isCanceling {
		T.Cancel(cancelKey)
		return
	}

	count := listener.open.Add(1)
	defer listener.open.Add(-1)

	if listener.MaxConnections != 0 && int(count) > listener.MaxConnections {
		_ = conn.WritePacket(
			perror.ToPacket(perror.New(
				perror.FATAL,
				perror.TooManyConnections,
				"Too many connections, sorry",
			)),
		)
		return
	}

	T.Serve(conn)
}

func (T *Server) gacceptFrom(listener *GListener) bool {
	conn, err := listener.accept()
	if err != nil {
		if errors.Is(err, net.ErrClosed) {
			return false
		}
		if netErr, ok := err.(*net.OpError); ok {
			// why can't they just expose this error
			if netErr.Err.Error() == "listener 'closed' 😉" {
				return false
			}
		}
		T.log.Warn("error accepting client", zap.Error(err))
		return true
	}

	go T.gaccept(listener, conn)
	return true
}
