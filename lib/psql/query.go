package psql

import (
	"crypto/tls"
	"errors"
	"io"
	"log"

	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

type resultReader struct{}

func (T *resultReader) EnableSSLClient(_ *tls.Config) error {
	return errors.New("ssl not supported")
}

func (T *resultReader) EnableSSLServer(_ *tls.Config) error {
	return errors.New("ssl not supported")
}

func (T *resultReader) ReadByte() (byte, error) {
	return 0, io.EOF
}

func (T *resultReader) ReadPacket(_ bool) (zap.Packet, error) {
	return nil, io.EOF
}

func (T *resultReader) WriteByte(_ byte) error {
	return nil
}

func (T *resultReader) WritePacket(packet zap.Packet) error {
	switch packet.Type() {
	case packets.TypeRowDescription:
		var rd packets.RowDescription
		if !rd.ReadFromPacket(packet) {
			return errors.New("invalid format")
		}
		log.Printf("row description: %#v", rd)
	case packets.TypeDataRow:
		var dr packets.DataRow
		if !dr.ReadFromPacket(packet) {
			return errors.New("invalid format")
		}
		log.Printf("data row: %#v", dr)
	}
	return nil
}

func (T *resultReader) Close() error {
	return nil
}

var _ zap.ReadWriter = (*resultReader)(nil)

func Query(server zap.ReadWriter, query string) error {
	var res resultReader
	ctx := backends.Context{
		Peer: &res,
	}
	if err := backends.QueryString(&ctx, server, query); err != nil {
		return err
	}

	return nil
}
