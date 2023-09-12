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
		if T.queueNext() {
			continue
		}

		if T.closed {
			return nil, io.EOF
		}

		if T.readC == nil {
			T.readC = sync.NewCond(&T.mu)
		}
		T.readC.Wait()
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
		if T.read.Length() == 0 && T.queueNext() {
			continue
		}

		if T.closed {
			return io.EOF
		}

		if T.writeC == nil {
			T.writeC = sync.NewCond(&T.mu)
		}
		T.writeC.Wait()
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

	if T.writeC != nil {
		T.writeC.Broadcast()
	}
	if T.readC != nil {
		T.readC.Broadcast()
	}
	return nil
}

var _ fed.Conn = (*Client)(nil)
