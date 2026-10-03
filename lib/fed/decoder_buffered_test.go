package fed

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
)

func frame(typ byte, length uint32, body []byte) []byte {
	b := make([]byte, 5+len(body))
	b[0] = typ
	binary.BigEndian.PutUint32(b[1:5], length)
	copy(b[5:], body)
	return b
}

func packetOf(typ byte, body []byte) []byte {
	return frame(typ, uint32(len(body))+4, body)
}

// A truncated header or body must not count as a buffered packet, or a relay would
// block on the transport while holding output. Cases run with the current packet empty or unread.
func TestDecoderHasBufferedTypedPacket(t *testing.T) {
	body := []byte("SELECT 1\x00")
	whole := packetOf('C', body)

	tails := []struct {
		name string
		data []byte
		want bool
	}{
		{"nothing", nil, false},
		{"one header byte", whole[:1], false},
		{"four header bytes", whole[:4], false},
		{"header only", whole[:5], false},
		{"partial body", whole[:len(whole)-1], false},
		{"whole packet", whole, true},
		{"whole packet and a partial one", append(append([]byte{}, whole...), whole[:3]...), true},
		{"empty body", packetOf('Z', nil), true},
		{"length 0", frame('C', 0, nil), false},
		{"length 3", frame('C', 3, nil), false},
		{"length max uint32", frame('C', math.MaxUint32, body), false},
	}
	currents := []struct {
		name string
		data []byte
	}{
		{"empty current", packetOf('Z', nil)},
		{"unread current body", packetOf('D', []byte("abcdef"))},
	}

	for _, cur := range currents {
		for _, tail := range tails {
			t.Run(cur.name+"/"+tail.name, func(t *testing.T) {
				d := NewDecoder(bytes.NewReader(append(append([]byte{}, cur.data...), tail.data...)))
				if err := d.Next(true); err != nil {
					t.Fatal(err)
				}
				if got := d.HasBufferedTypedPacket(); got != tail.want {
					t.Fatalf("HasBufferedTypedPacket = %v, want %v", got, tail.want)
				}
			})
		}
	}
}
