package protocol

import (
	"encoding/binary"
	"io"
	"strings"
)

func ReadByte(reader io.Reader) (byte, error) {
	var b [1]byte
	_, err := reader.Read(b[:])
	return b[0], err
}

func ReadInt8(reader io.Reader) (int8, error) {
	b, err := ReadByte(reader)
	return int8(b), err
}

func ReadUint16(reader io.Reader) (uint16, error) {
	var b [2]byte
	_, err := reader.Read(b[:])
	return binary.BigEndian.Uint16(b[:]), err
}

func ReadInt16(reader io.Reader) (int16, error) {
	b, err := ReadUint16(reader)
	return int16(b), err
}

func ReadUint32(reader io.Reader) (uint32, error) {
	var b [4]byte
	_, err := reader.Read(b[:])
	return binary.BigEndian.Uint32(b[:]), err
}

func ReadInt32(reader io.Reader) (int32, error) {
	b, err := ReadUint32(reader)
	return int32(b), err
}

func ReadUint64(reader io.Reader) (uint64, error) {
	var b [8]byte
	_, err := reader.Read(b[:])
	return binary.BigEndian.Uint64(b[:]), err
}

func ReadInt64(reader io.Reader) (int64, error) {
	b, err := ReadUint64(reader)
	return int64(b), err
}

func ReadString(reader io.Reader) (string, error) {
	var builder strings.Builder
	for {
		b, err := ReadByte(reader)
		if err != nil {
			return "", err
		}
		if b == 0 {
			return builder.String(), nil
		}
		builder.WriteByte(b)
	}
}

func WriteByte(writer io.Writer, value byte) (int, error) {
	var b [1]byte
	b[0] = value
	return writer.Write(b[:])
}

func WriteInt8(writer io.Writer, value int8) (int, error) {
	return WriteByte(writer, byte(value))
}

func WriteUint16(writer io.Writer, value uint16) (int, error) {
	var b [2]byte
	binary.BigEndian.PutUint16(b[:], value)
	return writer.Write(b[:])
}

func WriteInt16(writer io.Writer, value int16) (int, error) {
	return WriteUint16(writer, uint16(value))
}

func WriteUint32(writer io.Writer, value uint32) (int, error) {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], value)
	return writer.Write(b[:])
}

func WriteInt32(writer io.Writer, value int32) (int, error) {
	return WriteUint32(writer, uint32(value))
}

func WriteUint64(writer io.Writer, value uint64) (int, error) {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], value)
	return writer.Write(b[:])
}

func WriteInt64(writer io.Writer, value int64) (int, error) {
	return WriteUint64(writer, uint64(value))
}

func WriteString(writer io.Writer, value string) (int, error) {
	_, err := writer.Write([]byte(value))
	if err != nil {
		return 0, err
	}
	_, err = WriteByte(writer, 0)
	if err != nil {
		return len(value), err
	}
	return len(value) + 1, nil
}
