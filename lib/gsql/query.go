package gsql

import (
	"pggat2/lib/fed"
	packets "pggat2/lib/fed/packets/v3.0"
)

func (T *Client) Query(query string, results ...any) {
	T.mu.Lock()
	defer T.mu.Unlock()

	var q = packets.Query(query)

	T.queueResults(NewQueryWriter(results...))
	T.queuePackets(q.IntoPacket())
}

type QueryWriter struct {
	writers   []RowWriter
	writerNum int
	done      bool
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
	if packet.Type() == packets.TypeReadyForQuery {
		T.done = true
		return nil
	}

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

func (T *QueryWriter) Done() bool {
	return T.done
}

var _ ResultWriter = (*QueryWriter)(nil)
