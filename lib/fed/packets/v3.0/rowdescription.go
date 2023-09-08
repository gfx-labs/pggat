package packets

import (
	"pggat/lib/fed"
	"pggat/lib/util/slices"
)

type RowDescriptionField struct {
	Name         string
	TableID      int32
	ColumnID     int16
	Type         int32
	TypeLength   int16
	TypeModifier int32
	FormatCode   int16
}

type RowDescription struct {
	Fields []RowDescriptionField
}

func (T *RowDescription) ReadFromPacket(packet fed.Packet) bool {
	if packet.Type() != TypeRowDescription {
		return false
	}

	var fieldsPerRow uint16
	p := packet.ReadUint16(&fieldsPerRow)
	T.Fields = slices.Resize(T.Fields, int(fieldsPerRow))
	for i := 0; i < int(fieldsPerRow); i++ {
		p = p.ReadString(&T.Fields[i].Name)
		p = p.ReadInt32(&T.Fields[i].TableID)
		p = p.ReadInt16(&T.Fields[i].ColumnID)
		p = p.ReadInt32(&T.Fields[i].Type)
		p = p.ReadInt16(&T.Fields[i].TypeLength)
		p = p.ReadInt32(&T.Fields[i].TypeModifier)
		p = p.ReadInt16(&T.Fields[i].FormatCode)
	}

	return true
}

func (T *RowDescription) IntoPacket() fed.Packet {
	size := 2
	for _, v := range T.Fields {
		size += len(v.Name) + 1
		size += 4 + 2 + 4 + 2 + 4 + 2
	}

	packet := fed.NewPacket(TypeRowDescription, size)
	packet = packet.AppendUint16(uint16(len(T.Fields)))
	for _, v := range T.Fields {
		packet = packet.AppendString(v.Name)
		packet = packet.AppendInt32(v.TableID)
		packet = packet.AppendInt16(v.ColumnID)
		packet = packet.AppendInt32(v.Type)
		packet = packet.AppendInt16(v.TypeLength)
		packet = packet.AppendInt32(v.TypeModifier)
		packet = packet.AppendInt16(v.FormatCode)
	}

	return packet
}
