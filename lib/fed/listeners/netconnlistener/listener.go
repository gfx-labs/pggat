package netconnlistener

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"os"
	"path/filepath"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/fed/codecs/netconncodec"
	"gfx.cafe/gfx/pggat/lib/gat"
	"github.com/caddyserver/caddy/v2"
)

type Listener struct {
	Listener net.Listener
}

func init() {
	gat.RegisterNetwork("default", ListenerFunc)
}

func ListenerFunc(ctx context.Context, addr caddy.NetworkAddress, config *tls.Config) (fed.Listener, error) {
	if addr.Network == "unix" {
		if err := os.MkdirAll(filepath.Dir(addr.Host), 0o660); err != nil {
			return nil, err
		}
	}
	listener, err := addr.Listen(context.Background(), 0, net.ListenConfig{})
	if err != nil {
		return nil, err
	}
	log.Println("got fed conn")
	ncn := &Listener{Listener: listener.(net.Listener)}
	return ncn, nil
}

func (listener *Listener) Accept(fn func(*fed.Conn)) error {
	raw, err := listener.Listener.Accept()
	if err != nil {
		return err
	}
	fedConn := fed.NewConn(netconncodec.NewCodec(raw))
	go func() {
		fn(fedConn)
	}()
	return nil
}
func (l *Listener) Close() error {
	return l.Listener.Close()
}
