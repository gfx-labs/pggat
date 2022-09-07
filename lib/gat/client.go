package gat

import (
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
)

type Client interface {
	Send(pkt protocol.Packet) error
	Recv() <-chan protocol.Packet
}
