package main

import (
	"io"

	"pggat2/lib/frontend/frontends/v0"
	"pggat2/lib/pnet"
	"pggat2/lib/pnet/packet"
)

type testPacket struct {
	typ   packet.Type
	bytes []byte
}

type TestReader struct {
	packets []testPacket
	idx     int
	buf     packet.InBuf
}

func (T *TestReader) Read() (packet.In, error) {
	if T.idx >= len(T.packets) {
		return packet.In{}, io.EOF
	}
	pkt := T.packets[T.idx]
	T.idx++
	if pkt.typ == packet.None {
		panic("expected typed packet")
	}
	T.buf.Reset(pkt.typ, pkt.bytes)
	return packet.MakeIn(&T.buf), nil
}

func (T *TestReader) ReadUntyped() (packet.In, error) {
	if T.idx >= len(T.packets) {
		return packet.In{}, io.EOF
	}
	pkt := T.packets[T.idx]
	T.idx++
	if pkt.typ != packet.None {
		panic("expected untyped packet")
	}
	T.buf.Reset(pkt.typ, pkt.bytes)
	return packet.MakeIn(&T.buf), nil
}

var _ pnet.Reader = (*TestReader)(nil)

type LogWriter struct {
	buf packet.OutBuf
}

func (T *LogWriter) Write() packet.Out {
	if !T.buf.Initialized() {
		T.buf.Initialize(func(t packet.Type, bytes []byte) error {
			// log.Printf("recv %c %v\n", t, bytes)
			return nil
		})
	}
	T.buf.Reset()
	return packet.MakeOut(&T.buf)
}

var _ pnet.Writer = (*LogWriter)(nil)

func main() {
	frontend, err := frontends.NewListener()
	if err != nil {
		panic(err)
	}
	err = frontend.Listen()
	if err != nil {
		panic(err)
	}
	/*
		conn, err := net.Dial("tcp", "localhost:5432")
		if err != nil {
			panic(err)
		}
		server, err := backends.NewServer(conn)
		if err != nil {
			panic(err)
		}
		readWriter := pnet.JoinedReadWriter{
			Reader: &TestReader{
				packets: []testPacket{
					{
						typ:   packet.Query,
						bytes: []byte("select 1\x00"),
					},
					{
						typ:   packet.Query,
						bytes: []byte("set TimeZone = \"America/Denver\"\x00"),
					},
					{
						typ:   packet.Query,
						bytes: []byte("reset all\x00"),
					},
				},
			},
			Writer: &LogWriter{},
		}
		err = server.Transaction(readWriter)
		if err != nil {
			panic(err)
		}
		err = server.Transaction(readWriter)
		if err != nil {
			panic(err)
		}
		err = server.Transaction(readWriter)
		if err != nil {
			panic(err)
		}
		// log.Println(server)
		_ = server
		_ = conn.Close()
	*/
}
