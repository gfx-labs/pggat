package gsql

import (
	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
)

func Query(client *Client, results []any, query string) {
	var q = packets.Query(query)

	client.Do(NewQueryWriter(results...), q.IntoPacket(nil))
}

type QueryWriter struct {
	writers   []RowWriter
	writerNum int
}

func NewQueryWriter(results ...any) *QueryWriter {
	var writers = make([]RowWriter, 0, len(results))
	for _, result := range results {
		writers = append(writers, MakeRowWriter(result))
	}

	return &QueryWriter{
		writers: writers,
	}
}

func (T *QueryWriter) WritePacket(packet fed.Packet) error {
	if T.writerNum >= len(T.writers) {
		// ignore
		return nil
	}

	result := &T.writers[T.writerNum]
	if err := result.WritePacket(packet); err != nil {
		return err
	}

	if result.Done() {
		T.writerNum++
	}

	return nil
}

var _ ResultWriter = (*QueryWriter)(nil)
