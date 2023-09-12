package gsql

import (
	"io"
	"net"
	"sync"

	"pggat/lib/fed"
	"pggat/lib/util/ring"
)

type batch struct {
	result  ResultWriter
	packets []fed.Packet
}

type Client struct {
	write ResultWriter
	read  ring.Ring[fed.Packet]

	queue ring.Ring[batch]

	closed bool
	mu     sync.Mutex

	readQueue  chan struct{}
	writeQueue chan struct{}
}

func (T *Client) Do(result ResultWriter, packets ...fed.Packet) {
	T.mu.Lock()
	defer T.mu.Unlock()

	T.queue.PushBack(batch{
		result:  result,
		packets: packets,
	})

	if T.readQueue != nil {
		for {
			select {
			case T.readQueue <- struct{}{}:
			default:
				return
			}
		}
	}
}

func (T *Client) ReadPacket(typed bool) (fed.Packet, error) {
	T.mu.Lock()
	defer T.mu.Unlock()

	var p fed.Packet
	for {
		var ok bool
		p, ok = T.read.PopFront()
		if ok {
			break
		}

		// try to add next in queue
		b, ok := T.queue.PopFront()
		if ok {
			for _, packet := range b.packets {
				T.read.PushBack(packet)
			}
			T.write = b.result
		outer:
			for {
				select {
				case T.writeQueue <- struct{}{}:
				default:
					break outer
				}
			}
			continue
		}

		if T.closed {
			return nil, io.EOF
		}

		func() {
			if T.readQueue == nil {
				T.readQueue = make(chan struct{})
			}
			q := T.readQueue

			T.mu.Unlock()
			defer T.mu.Lock()

			<-q
		}()
	}

	if (p.Type() == 0 && typed) || (p.Type() != 0 && !typed) {
		return nil, ErrTypedMismatch
	}

	return p, nil
}

func (T *Client) WritePacket(packet fed.Packet) error {
	T.mu.Lock()
	defer T.mu.Unlock()

	for T.write == nil {
		if T.closed {
			return io.EOF
		}

		func() {
			if T.writeQueue == nil {
				T.writeQueue = make(chan struct{})
			}
			q := T.writeQueue

			T.mu.Unlock()
			defer T.mu.Lock()

			<-q
		}()
	}

	if err := T.write.WritePacket(packet); err != nil {
		return err
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

	if T.writeQueue != nil {
		close(T.writeQueue)
	}
	if T.readQueue != nil {
		close(T.readQueue)
	}
	return nil
}

var _ fed.Conn = (*Client)(nil)
