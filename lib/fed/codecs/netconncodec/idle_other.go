//go:build !(linux || darwin || freebsd || netbsd || openbsd || dragonfly)

package netconncodec

import "net"

func peekPending(net.Conn) bool {
	return false
}
