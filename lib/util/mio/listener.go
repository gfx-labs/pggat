package mio

import (
	"errors"
	"fmt"
	"net"
	"sync"

	"gfx.cafe/gfx/pggat/lib/util/ring"
)

var (
	listeners   map[string]*Listener
	listenersMu sync.RWMutex
)

type Listener struct {
	address  string
	incoming ring.Ring[*Conn]
	all      []*Conn
	a        sync.Cond
	closed   bool
	mu       sync.Mutex
}

func Listen(address string) (*Listener, error) {
	listenersMu.Lock()
	defer listenersMu.Unlock()

	if _, ok := listeners[address]; ok {
		return nil, errors.New("address already in use")
	}

	l := &Listener{
		address: address,
	}
	if listeners == nil {
		listeners = make(map[string]*Listener)
	}
	listeners[address] = l

	return l, nil
}

func (T *Listener) Accept() (net.Conn, error) {
	T.mu.Lock()
	defer T.mu.Unlock()

	for T.incoming.Length() == 0 {
		if T.closed {
			return nil, net.ErrClosed
		}

		if T.a.L == nil {
			T.a.L = &T.mu
		}
		T.a.Wait()
	}

	c, _ := T.incoming.PopFront()
	return OutwardConn{Conn: c}, nil
}

func (T *Listener) close() error {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.closed {
		return net.ErrClosed
	}
	T.closed = true
	for _, c := range T.all {
		_ = c.Close()
	}

	return nil
}

func (T *Listener) Close() error {
	if err := T.close(); err != nil {
		return err
	}

	listenersMu.Lock()
	defer listenersMu.Unlock()

	delete(listeners, T.address)

	return nil
}

func (T *Listener) Addr() net.Addr {
	return ListenerAddr{
		Listener: T,
	}
}

var _ net.Listener = (*Listener)(nil)

func lookup(address string) *Listener {
	listenersMu.RLock()
	defer listenersMu.RUnlock()

	return listeners[address]
}

func Dial(address string) (net.Conn, error) {
	l := lookup(address)

	if l == nil {
		return nil, errors.New("address not found")
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	c := new(Conn)

	l.all = append(l.all, c)
	l.incoming.PushBack(c)
	if l.a.L == nil {
		l.a.L = &l.mu
	}
	l.a.Signal()

	return OutwardConn{
		Conn: c,
	}, nil
}

type ListenerAddr struct {
	Listener *Listener
}

func (T ListenerAddr) Network() string {
	return "mio"
}

func (T ListenerAddr) String() string {
	return fmt.Sprintf("memory listener(%p)", T.Listener)
}

var _ net.Addr = ListenerAddr{}
