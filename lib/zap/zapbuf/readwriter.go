package zapbuf

import (
	"errors"

	"pggat2/lib/zap"
)

var ErrAlreadyBuffered = errors.New("already buffered")

type Buffer struct {
	zap.ReadWriter
	buffered  *zap.Packet
	hasBuffer bool
}

func NewBuffer(rw zap.ReadWriter) *Buffer {
	return &Buffer{
		ReadWriter: rw,
	}
}

func (T *Buffer) Buffer() error {
	if T.hasBuffer {
		return ErrAlreadyBuffered
	}

	T.hasBuffer = true

	if T.buffered == nil {
		T.buffered = zap.NewPacket()
	}

	return T.ReadWriter.Read(T.buffered)
}

func (T *Buffer) Read(packet *zap.Packet) error {
	if T.hasBuffer {
		// swap buffers
		packet.PacketWriter, T.buffered.PacketWriter = T.buffered.PacketWriter, packet.PacketWriter
		T.hasBuffer = false
		return nil
	}

	return T.ReadWriter.Read(packet)
}

func (T *Buffer) Done() {
	if T.buffered != nil {
		T.buffered.Done()
	}
}

var _ zap.ReadWriter = (*Buffer)(nil)
