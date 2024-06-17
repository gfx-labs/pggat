package netconnlistener

import (
	"net"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/fed/codecs/netconncodec"
)

type Listener struct {
	Listener net.Listener
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
	return l.Close()
}
