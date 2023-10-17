package hybrid

type Buffer struct {
	buf []byte
	pos int
}

func (T *Buffer) Read(b []byte) (int, error) {
	n := copy(b, T.buf[T.pos:])
	T.pos += n
	return n, nil
}

func (T *Buffer) Write(b []byte) (int, error) {
	T.buf = append(T.buf, b...)
	return len(b), nil
}

func (T *Buffer) Full() []byte {
	return T.buf
}

func (T *Buffer) Reset() {
	T.buf = T.buf[:0]
	T.pos = 0
}
