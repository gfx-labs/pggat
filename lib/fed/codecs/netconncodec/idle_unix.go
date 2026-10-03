//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly

package netconncodec

import (
	"crypto/tls"
	"net"
	"syscall"
)

// peekPending reports whether the socket has unread bytes, or has been closed or reset by the peer.
// It peeks without consuming. For TLS it looks at the underlying socket, where any
// ciphertext (data, alert, or close_notify) also means the connection is not idle.
func peekPending(conn net.Conn) bool {
	if t, ok := conn.(*tls.Conn); ok {
		conn = t.NetConn()
	}
	sc, ok := conn.(syscall.Conn)
	if !ok {
		return false
	}
	raw, err := sc.SyscallConn()
	if err != nil {
		return true
	}

	var (
		buf  [1]byte
		serr error
	)
	err = raw.Read(func(fd uintptr) bool {
		for {
			// n == 0 with no error is EOF, n > 0 is unread data.
			_, _, serr = syscall.Recvfrom(int(fd), buf[:], syscall.MSG_PEEK|syscall.MSG_DONTWAIT)
			if serr != syscall.EINTR {
				break
			}
		}
		// Returning true stops raw.Read from waiting for readability.
		return true
	})
	if err != nil {
		return true
	}
	// EAGAIN means nothing is pending. Any other error is a dead socket.
	return serr != syscall.EAGAIN
}
