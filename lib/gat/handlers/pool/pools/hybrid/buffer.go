package hybrid

import "io"

type Buffer struct {
	buf []byte
	pos int
}

func (T *Buffer) Read(b []byte) (int, error) {
	if T.Buffered() == 0 {
		return 0, io.EOF
	}
	n := copy(b, T.buf[T.pos:])
	T.pos += n
	return n, nil
}

func (T *Buffer) Write(b []byte) (int, error) {
	T.buf = append(T.buf, b...)
	return len(b), nil
}

func (T *Buffer) Buffered() int {
	return len(T.buf) - T.pos
}

func (T *Buffer) Reset() {
	T.buf = T.buf[:0]
	T.pos = 0
}

func (T *Buffer) ResetRead() {
	T.pos = 0
}
