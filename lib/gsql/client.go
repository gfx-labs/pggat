package gsql

import (
	"crypto/tls"
	"net"
	"sync"

	"pggat2/lib/fed"
	"pggat2/lib/util/ring"
)

type Client struct {
	writeQ ring.Ring[ResultWriter]
	writeC *sync.Cond
	write  ResultWriter

	readQ ring.Ring[fed.Packet]
	readC *sync.Cond

	closed bool
	mu     sync.Mutex
}

func (*Client) EnableSSLClient(_ *tls.Config) error {
	panic("not implemented")
}

func (*Client) EnableSSLServer(_ *tls.Config) error {
	panic("not implemented")
}

func (*Client) ReadByte() (byte, error) {
	panic("not implemented")
}

func (T *Client) queuePackets(packets ...fed.Packet) {
	for _, packet := range packets {
		T.readQ.PushBack(packet)

		if T.readC != nil {
			T.readC.Signal()
		}
	}
}

func (T *Client) queueResults(results ...ResultWriter) {
	for _, result := range results {
		T.writeQ.PushBack(result)

		if T.writeC != nil {
			T.writeC.Signal()
		}
	}
}

func (T *Client) ReadPacket(typed bool) (fed.Packet, error) {
	T.mu.Lock()
	defer T.mu.Unlock()

	p, ok := T.readQ.PopFront()
	for !ok {
		if T.closed {
			return nil, net.ErrClosed
		}
		if T.readC == nil {
			T.readC = sync.NewCond(&T.mu)
		}
		T.readC.Wait()
		p, ok = T.readQ.PopFront()
	}

	if (p.Type() == 0 && typed) || (p.Type() != 0 && !typed) {
		panic("tried to read typed as untyped or untyped as typed")
	}

	return p, nil
}

func (*Client) WriteByte(_ byte) error {
	panic("not implemented")
}

func (T *Client) WritePacket(packet fed.Packet) error {
	if T.write == nil {
		T.write, _ = T.writeQ.PopFront()
		for T.write == nil {
			if T.closed {
				return net.ErrClosed
			}
			if T.writeC == nil {
				T.writeC = sync.NewCond(&T.mu)
			}
			T.writeC.Wait()
			T.write, _ = T.writeQ.PopFront()
		}
	}

	if err := T.write.WritePacket(packet); err != nil {
		return err
	}

	if T.write.Done() {
		T.write = nil
	}

	return nil
}

func (T *Client) Close() error {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.closed {
		return net.ErrClosed
	}

	T.closed = true
	return nil
}

var _ fed.Conn = (*Client)(nil)
