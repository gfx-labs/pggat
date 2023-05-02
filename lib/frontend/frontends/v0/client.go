package frontends

import (
	"encoding/binary"
	"log"
	"net"

	"pggat2/lib/frontend"
	"pggat2/lib/frontend/pnet"
)

type Client struct {
	conn net.Conn
}

func NewClient(conn net.Conn) *Client {
	client := &Client{
		conn: conn,
	}
	go client.read()
	return client
}

func (T *Client) read() {
	reader := pnet.MakeReader(T.conn)
	// read initial packet
	pkt, err := reader.ReadUntyped()
	if err != nil {
		panic(err)
	}
	log.Printf("received packet: %#v", pkt)
	log.Println(binary.BigEndian.Uint32(pkt.Payload))
	for {
		pkt, err = reader.Read()
		if err != nil {
			panic(err)
		}
		log.Printf("received packet: %#v", pkt)
	}
}

var _ frontend.Client = (*Client)(nil)
