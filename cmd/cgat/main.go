package main

import (
	"io"
	"net"
	"sync"

	"pggat2/lib/backend/backends/v0"
	"pggat2/lib/pnet"
	"pggat2/lib/pnet/packet"
	"pggat2/lib/router"
	"pggat2/lib/router/routers/v0"
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

func makeTestServer(r router.Router, wg *sync.WaitGroup) {
	conn, err := net.Dial("tcp", "localhost:5432")
	if err != nil {
		panic(err)
	}
	server := backends.NewServer(conn)
	if server == nil {
		panic("failed to connect to server")
	}
	go func() {
		handler := r.NewHandler(true)
		for {
			peer := handler.Next()
			server.Handle(peer)
			wg.Done()
		}
	}()
}

func main() {
	/*
		frontend, err := frontends.NewListener()
		if err != nil {
			panic(err)
		}
		err = frontend.Listen()
		if err != nil {
			panic(err)
		}
	*/
	r := routers.MakeRouter()
	var wg sync.WaitGroup
	makeTestServer(&r, &wg)
	// makeTestServer(&r, &wg)

	src := r.NewSource()
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
	wg.Add(3)
	src.Handle(readWriter, true)
	src.Handle(readWriter, true)
	src.Handle(readWriter, true)
	wg.Wait()
	// log.Println(server)
}
