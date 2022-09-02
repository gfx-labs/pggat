package protocol

import "io"

// codegen: modify for debug only

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
