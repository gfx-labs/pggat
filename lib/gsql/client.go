package gsql

import (
	"net"
	"sync"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/util/ring"
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

	readC  *sync.Cond
	writeC *sync.Cond
}

func (T *Client) Do(result ResultWriter, packets ...fed.Packet) {
	T.mu.Lock()
	defer T.mu.Unlock()

	T.queue.PushBack(batch{
		result:  result,
		packets: packets,
	})

	if T.readC != nil {
		T.readC.Broadcast()
	}
}

func (T *Client) queueNext() bool {
	b, ok := T.queue.PopFront()
	if ok {
		for _, packet := range b.packets {
			T.read.PushBack(packet)
		}
		T.write = b.result
		if T.writeC != nil {
			T.writeC.Broadcast()
		}
		return true
	}

	return false
}

func (T *Client) Read(b []byte) (int, error) {
	panic("TODO")
}

func (T *Client) Write(b []byte) (int, error) {
	panic("TODO")
}

func (T *Client) Close() error {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.closed {
		return net.ErrClosed
	}

	T.closed = true

	if T.writeC != nil {
		T.writeC.Broadcast()
	}
	if T.readC != nil {
		T.readC.Broadcast()
	}
	return nil
}
