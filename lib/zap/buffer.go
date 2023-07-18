package zap

import (
	"encoding/binary"
	"io"

	"pggat2/lib/util/slices"
)

type buffered struct {
	offset int
	typed  bool
}

type Buffer struct {
	primary []byte
	items   []buffered
}

func (T *Buffer) ReadFrom(reader io.Reader, typed bool) error {
	offset := len(T.primary)
	if typed {
		T.primary = append(T.primary, 0, 0, 0, 0, 0)
	} else {
		T.primary = append(T.primary, 0, 0, 0, 0)
	}

	_, err := io.ReadFull(reader, T.primary[offset:])
	if err != nil {
		T.primary = T.primary[:offset]
		return err
	}

	var length uint32
	if typed {
		length = binary.BigEndian.Uint32(T.primary[offset+1:])
	} else {
		length = binary.BigEndian.Uint32(T.primary[offset:])
	}

	T.primary = slices.Resize(T.primary, len(T.primary)+int(length)-4)

	var payload []byte
	if typed {
		payload = T.primary[offset+5:]
	} else {
		payload = T.primary[offset+4:]
	}

	_, err = io.ReadFull(reader, payload)
	return err
}

func (T *Buffer) WriteInto(writer io.Writer) error {
	_, err := writer.Write(T.primary)
	return err
}

func (T *Buffer) Full() []byte {
	return T.primary
}

func (T *Buffer) Reset() {
	T.primary = T.primary[:0]
	T.items = T.items[:0]
}

// Count returns the number of packets in the buffer
func (T *Buffer) Count() int {
	return len(T.items)
}

func (T *Buffer) Inspect(i int) Inspector {
	item := T.items[i]
	inspector := Inspector{
		buffer: T,
		offset: item.offset,
		typed:  item.typed,
	}
	inspector.Reset()
	return inspector
}

func (T *Buffer) Build(typed bool) Builder {
	offset := len(T.primary)
	T.items = append(T.items, buffered{
		offset: offset,
		typed:  typed,
	})
	if typed {
		T.primary = append(T.primary, 0, 0, 0, 0, 4)
	} else {
		T.primary = append(T.primary, 0, 0, 0, 4)
	}
	return Builder{
		buffer: T,
		offset: offset,
		typed:  typed,
	}
}
