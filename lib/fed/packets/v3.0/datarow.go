package packets

import (
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/util/slices"
)

type DataRow struct {
	Columns [][]byte
}

func (T *DataRow) ReadFromPacket(packet fed.Packet) bool {
	if packet.Type() != TypeDataRow {
		return false
	}

	var columnCount uint16
	p := packet.ReadUint16(&columnCount)
	T.Columns = slices.Resize(T.Columns, int(columnCount))
	for i := 0; i < int(columnCount); i++ {
		var valueLength int32
		p = p.ReadInt32(&valueLength)
		if valueLength == -1 {
			continue
		}
		T.Columns[i] = slices.Resize(T.Columns[i], int(valueLength))
		p = p.ReadBytes(T.Columns[i])
	}

	return true
}

func (T *DataRow) IntoPacket(packet fed.Packet) fed.Packet {
	size := 2
	for _, v := range T.Columns {
		size += len(v) + 4
	}

	packet = packet.Reset(TypeDataRow, size)
	packet = packet.AppendUint16(uint16(len(T.Columns)))
	for _, v := range T.Columns {
		if v == nil {
			packet = packet.AppendInt32(-1)
			continue
		}

		packet = packet.AppendInt32(int32(len(v)))
		packet = packet.AppendBytes(v)
	}

	return packet
}
