package protocol

import (
	"bytes"
	"io"
)

// codegen: modify for debug only

var _ bytes.Buffer
var _ io.Reader

type FieldsCopyData struct {
	Data []byte
}

func (T *FieldsCopyData) Read(payloadLength int, reader io.Reader) (err error) {
	DataLength := payloadLength
	T.Data = make([]byte, int(DataLength))
	for i := 0; i < int(DataLength); i++ {
		T.Data[i], err = ReadByte(reader)
		if err != nil {
			return
		}
	}
	return
}

func (T *FieldsCopyData) Write(writer io.Writer) (length int, err error) {
	var temp int
	for _, v := range T.Data {
		temp, err = WriteByte(writer, v)
		if err != nil {
			return
		}
		length += temp
	}
	_ = temp
	return
}

type CopyData struct {
	fields FieldsCopyData
}

// Read reads all but the packet identifier. Be sure to read that beforehand (if it exists)
func (T *CopyData) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.fields.Read(int(length-4), reader)
}

func (T *CopyData) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('d'))
	if err != nil {
		length = 1
		return
	}
	_, err = WriteInt32(writer, int32(length))
	if err != nil {
		length = 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
}

type FieldsCopyDone struct {
}

func (T *FieldsCopyDone) Read(payloadLength int, reader io.Reader) (err error) {
	return
}

func (T *FieldsCopyDone) Write(writer io.Writer) (length int, err error) {
	var temp int
	_ = temp
	return
}

type CopyDone struct {
	fields FieldsCopyDone
}

// Read reads all but the packet identifier. Be sure to read that beforehand (if it exists)
func (T *CopyDone) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.fields.Read(int(length-4), reader)
}

func (T *CopyDone) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('c'))
	if err != nil {
		length = 1
		return
	}
	_, err = WriteInt32(writer, int32(length))
	if err != nil {
		length = 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
}
