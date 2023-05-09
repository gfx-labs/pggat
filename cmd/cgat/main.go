package main

import (
	"io"
	"net"
	"sync"

	"pggat2/lib/backend/backends/v0"
	"pggat2/lib/frontend/frontends/v0"
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
	r := routers.MakeRouter()
	var wg sync.WaitGroup
	makeTestServer(&r, &wg)

	listener, err := net.Listen("tcp", "0.0.0.0:6432") // TODO(garet) make this configurable
	if err != nil {
		panic(err)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}
		go func() {
			source := r.NewSource()
			client := frontends.NewClient(conn)
			for {
				wg.Add(1)
				source.Handle(client, false)
				wg.Wait()
			}
		}()
	}
}
