package gatutil

type Type interface {
	OID() int32
	Len() int16
}

type Text struct{}

func (Text) OID() int32 {
	return 25
}

func (Text) Len() int16 {
	return -1
}

type Int16 struct{}

func (Int16) OID() int32 {
	return 21
}

func (Int16) Len() int16 {
	return 2
}

type Int32 struct{}

func (Int32) OID() int32 {
	return 23
}

func (Int32) Len() int16 {
	return 4
}

type Int64 struct{}

func (Int64) OID() int32 {
	return 20
}

func (Int64) Len() int16 {
	return 8
}

type Char struct{}

func (Char) OID() int32 {
	return 18
}

func (Char) Len() int16 {
	return 1
}

type Bool struct{}

func (Bool) OID() int32 {
	return 16
}

func (Bool) Len() int16 {
	return 1
}

type Float32 struct{}

func (Float32) OID() int32 {
	return 700
}

func (Float32) Len() int16 {
	return 4
}

type Float64 struct{}

func (Float64) OID() int32 {
	return 701
}

func (Float64) Len() int16 {
	return 8
}
