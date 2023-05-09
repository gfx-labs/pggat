package main

import (
	"io"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"

	"pggat2/lib/backend/backends/v0"
	"pggat2/lib/frontend/frontends/v0"
	"pggat2/lib/pnet"
	"pggat2/lib/pnet/packet"
	"pggat2/lib/rob"
	"pggat2/lib/rob/schedulers/v2"
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

type job struct {
	rw   pnet.ReadWriter
	done chan<- struct{}
}

func testServer(r rob.Scheduler) {
	conn, err := net.Dial("tcp", "localhost:5432")
	if err != nil {
		panic(err)
	}
	server := backends.NewServer(conn)
	if server == nil {
		panic("failed to connect to server")
	}

	sink := r.NewSink(0)
	for {
		j := sink.Read().(job)
		server.Handle(j.rw)
		select {
		case j.done <- struct{}{}:
		default:
		}
	}
}

func main() {
	go func() {
		panic(http.ListenAndServe(":8080", nil))
	}()

	r := schedulers.MakeScheduler()
	go testServer(&r)

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
			done := make(chan struct{})
			for {
				reader, err := pnet.PreRead(client)
				if err != nil {
					log.Println("failed", err)
					break
				}
				source.Schedule(job{
					rw: pnet.JoinedReadWriter{
						Reader: reader,
						Writer: client,
					},
					done: done,
				}, 0)
				<-done
			}
		}()
	}
}
