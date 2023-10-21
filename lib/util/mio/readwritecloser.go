package mio

import (
	"context"
	"io"
	"net"
	"sync"
	"time"
)

type ReadWriteCloser struct {
	buf               []byte
	r                 sync.Cond
	closed            bool
	readDeadline      time.Time
	readDeadlineTimer *time.Timer
	writeDeadline     time.Time
	mu                sync.Mutex
}

func NewReadWriteCloserSize(size int) *ReadWriteCloser {
	return &ReadWriteCloser{
		buf: make([]byte, 0, size),
	}
}

func (T *ReadWriteCloser) Read(b []byte) (n int, err error) {
	T.mu.Lock()
	defer T.mu.Unlock()

	for len(T.buf) == 0 {
		if T.closed {
			return 0, io.EOF
		}

		if T.readDeadline != (time.Time{}) && time.Now().After(T.readDeadline) {
			return 0, context.DeadlineExceeded
		}

		if T.r.L == nil {
			T.r.L = &T.mu
		}
		T.r.Wait()
	}

	n = copy(b, T.buf)
	copy(T.buf, T.buf[n:])
	T.buf = T.buf[:len(T.buf)-n]

	return
}

func (T *ReadWriteCloser) Write(b []byte) (n int, err error) {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.closed {
		return 0, net.ErrClosed
	}

	if T.writeDeadline != (time.Time{}) && time.Now().After(T.writeDeadline) {
		return 0, context.DeadlineExceeded
	}

	T.buf = append(T.buf, b...)
	n = len(b)

	if T.r.L == nil {
		T.r.L = &T.mu
	}
	T.r.Broadcast()

	return
}

func (T *ReadWriteCloser) Close() error {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.closed {
		return net.ErrClosed
	}
	T.closed = true

	if T.readDeadlineTimer != nil {
		T.readDeadlineTimer.Stop()
	}

	if T.r.L == nil {
		T.r.L = &T.mu
	}
	T.r.Broadcast()

	return nil
}

func (T *ReadWriteCloser) readDeadlineExceeded() {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.r.L == nil {
		T.r.L = &T.mu
	}
	T.r.Broadcast()
}

func (T *ReadWriteCloser) setReadDeadline(t time.Time) error {
	if T.closed {
		return net.ErrClosed
	}

	if t == (time.Time{}) {
		if T.readDeadlineTimer != nil {
			T.readDeadlineTimer.Stop()
		}
	} else {
		if T.readDeadlineTimer == nil {
			T.readDeadlineTimer = time.AfterFunc(time.Until(t), T.readDeadlineExceeded)
		} else {
			T.readDeadlineTimer.Reset(time.Until(t))
		}
	}

	return nil
}

func (T *ReadWriteCloser) SetReadDeadline(t time.Time) error {
	T.mu.Lock()
	defer T.mu.Unlock()

	return T.setReadDeadline(t)
}

func (T *ReadWriteCloser) setWriteDeadline(t time.Time) error {
	T.writeDeadline = t
	return nil
}

func (T *ReadWriteCloser) SetWriteDeadline(t time.Time) error {
	T.mu.Lock()
	defer T.mu.Unlock()

	return T.setWriteDeadline(t)
}

func (T *ReadWriteCloser) SetDeadline(t time.Time) error {
	T.mu.Lock()
	defer T.mu.Unlock()

	if err := T.setReadDeadline(t); err != nil {
		return err
	}
	return T.setWriteDeadline(t)
}
