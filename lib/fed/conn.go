package fed

import (
	"gfx.cafe/gfx/pggat/lib/util/decorator"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

type Conn struct {
	noCopy decorator.NoCopy

	ReadWriteCloser

	Middleware []Middleware

	User              string
	Database          string
	InitialParameters map[strutil.CIString]string
	Authenticated     bool
	BackendKey        [8]byte
}

func NewConn(rw ReadWriteCloser) *Conn {
	return &Conn{
		ReadWriteCloser: rw,
	}
}

func (T *Conn) ReadPacket(typed bool, buffer Packet) (packet Packet, err error) {
	packet = buffer
	for {
		packet, err = T.ReadWriteCloser.ReadPacket(typed, buffer)
		if err != nil {
			return
		}
		for _, middleware := range T.Middleware {
			packet, err = middleware.ReadPacket(packet)
			if err != nil {
				return
			}
			if len(packet) == 0 {
				break
			}
		}
		if len(packet) != 0 {
			return
		}
	}
}

func (T *Conn) WritePacket(packet Packet) (err error) {
	for _, middleware := range T.Middleware {
		packet, err = middleware.ReadPacket(packet)
		if err != nil || len(packet) == 0 {
			return
		}
	}
	err = T.ReadWriteCloser.WritePacket(packet)
	return
}

var _ ReadWriteCloser = (*Conn)(nil)
