package fed

import (
	"context"
	"crypto/tls"
	"net"
)

type PacketCodec interface {
	ReadPacket(ctx context.Context,typed bool) (Packet, error)
	WritePacket(ctx context.Context,packet Packet) error
	WriteByte(ctx context.Context,b byte) error
	ReadByte(ctx context.Context,) (byte, error)

	LocalAddr() net.Addr
	Flush(ctx context.Context,) error
	Close(ctx context.Context,) error

	SSL() bool
	EnableSSL(ctx context.Context,config *tls.Config, isClient bool) error
}
