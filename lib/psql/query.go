package psql

import (
	"errors"
	"io"
	"reflect"
	"strconv"

	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

type packetReader struct {
	packets []zap.Packet
}

func (T *packetReader) ReadByte() (byte, error) {
	return 0, io.EOF
}

func (T *packetReader) ReadPacket(typed bool) (zap.Packet, error) {
	if len(T.packets) == 0 {
		return nil, io.EOF
	}

	packet := T.packets[0]
	packetTyped := packet.Type() != 0

	if packetTyped != typed {
		return nil, io.EOF
	}

	T.packets = T.packets[1:]
	return packet, nil
}

var _ zap.Reader = (*packetReader)(nil)

type eofReader struct{}

func (eofReader) ReadByte() (byte, error) {
	return 0, io.EOF
}

func (eofReader) ReadPacket(_ bool) (zap.Packet, error) {
	return nil, io.EOF
}

var _ zap.Reader = eofReader{}

type resultWriter struct {
	result reflect.Value
	rd     packets.RowDescription
	err    error
	row    int
}

func (T *resultWriter) WriteByte(_ byte) error {
	return nil
}

func (T *resultWriter) set(i int, row []byte) error {
	if i >= len(T.rd.Fields) {
		return ErrExtraFields
	}
	desc := T.rd.Fields[i]

	result := T.result

	// unptr
	for result.Kind() == reflect.Pointer {
		if result.IsNil() {
			if !result.CanSet() {
				return ErrResultMustBeNonNil
			}
			result.Set(reflect.New(result.Type().Elem()))
		}
		result = result.Elem()
	}

	// get row
outer:
	for {
		kind := result.Kind()
		switch kind {
		case reflect.Array:
			if T.row >= result.Len() {
				return ErrResultTooBig
			}
			result = result.Index(T.row)
			break outer
		case reflect.Slice:
			for T.row >= result.Len() {
				if !result.CanSet() {
					return ErrResultTooBig
				}
				result.Set(reflect.Append(result, reflect.Zero(result.Type().Elem())))
			}
			result = result.Index(T.row)
			break outer
		case reflect.Struct, reflect.Map:
			if T.row != 0 {
				return ErrResultTooBig
			}
			break outer
		default:
			return ErrUnexpectedType
		}
	}

	// unptr
	for result.Kind() == reflect.Pointer {
		if result.IsNil() {
			if !result.CanSet() {
				return ErrResultMustBeNonNil
			}
			result.Set(reflect.New(result.Type().Elem()))
		}
		result = result.Elem()
	}

	// get field
	kind := result.Kind()
	typ := result.Type()
outer2:
	switch kind {
	case reflect.Struct:
		for j := 0; j < typ.NumField(); j++ {
			field := typ.Field(j)
			if !field.IsExported() {
				continue
			}

			sqlName, hasSQLName := field.Tag.Lookup("sql")
			if !hasSQLName {
				sqlName = field.Name
			}

			if sqlName == desc.Name {
				result = result.Field(j)
				break outer2
			}
		}

		// ignore field
		return nil
	case reflect.Map:
		key := typ.Key()
		if key.Kind() != reflect.String {
			return ErrUnexpectedType
		}

		if result.IsNil() {
			if !result.CanSet() {
				return ErrResultMustBeNonNil
			}

			result.Set(reflect.MakeMap(typ))
		}

		k := reflect.New(key).Elem()
		k.SetString(desc.Name)
		value := typ.Elem()
		v := reflect.New(value).Elem()
		m := result
		result = v
		defer func() {
			m.SetMapIndex(k, v)
		}()
	default:
		return ErrUnexpectedType
	}

	if !result.CanSet() {
		return ErrUnexpectedType
	}

	if row == nil {
		if result.Kind() == reflect.Pointer {
			if result.IsNil() {
				return nil
			}
			if !result.CanSet() {
				return ErrUnexpectedType
			}
			result.Set(reflect.Zero(result.Type()))
			return nil
		} else {
			return ErrUnexpectedType
		}
	}

	// unptr
	for result.Kind() == reflect.Pointer {
		if result.IsNil() {
			if !result.CanSet() {
				return ErrResultMustBeNonNil
			}
			result.Set(reflect.New(result.Type().Elem()))
		}
		result = result.Elem()
	}

	kind = result.Kind()
	typ = result.Type()
	switch kind {
	case reflect.String:
		result.SetString(string(row))
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		x, err := strconv.ParseUint(string(row), 10, 64)
		if err != nil {
			return err
		}
		result.SetUint(x)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		x, err := strconv.ParseInt(string(row), 10, 64)
		if err != nil {
			return err
		}
		result.SetInt(x)
		return nil
	case reflect.Float32, reflect.Float64:
		x, err := strconv.ParseFloat(string(row), 64)
		if err != nil {
			return err
		}
		result.SetFloat(x)
		return nil
	case reflect.Bool:
		if len(row) != 1 {
			return ErrUnexpectedType
		}
		var x bool
		switch row[0] {
		case 'f':
			x = false
		case 't':
			x = true
		default:
			return ErrUnexpectedType
		}
		result.SetBool(x)
		return nil
	default:
		return ErrUnexpectedType
	}
}

func (T *resultWriter) WritePacket(packet zap.Packet) error {
	if T.err != nil {
		return ErrFailed
	}
	switch packet.Type() {
	case packets.TypeRowDescription:
		if !T.rd.ReadFromPacket(packet) {
			return errors.New("invalid format")
		}
	case packets.TypeDataRow:
		var dr packets.DataRow
		if !dr.ReadFromPacket(packet) {
			return errors.New("invalid format")
		}
		for i, row := range dr.Columns {
			if err := T.set(i, row); err != nil {
				T.err = err
				return err
			}
		}
		T.row += 1
	case packets.TypeErrorResponse:
		var err packets.ErrorResponse
		if !err.ReadFromPacket(packet) {
			return errors.New("invalid format")
		}
		T.err = errors.New(err.Error.String())
	}
	return nil
}

var _ zap.Writer = (*resultWriter)(nil)

func Query(server zap.ReadWriter, result any, query string, args ...any) error {
	if len(args) == 0 {
		// simple query

		w := resultWriter{
			result: reflect.ValueOf(result),
		}
		ctx := backends.Context{
			Peer: zap.CombinedReadWriter{
				Reader: eofReader{},
				Writer: &w,
			},
		}
		if err := backends.QueryString(&ctx, server, query); err != nil {
			return err
		}
		if w.err != nil {
			return w.err
		}

		return nil
	}

	// must use eqp

	// parse
	parse := packets.Parse{
		Query: query,
	}

	// bind
	params := make([][]byte, 0, len(args))
	for _, arg := range args {
		var value []byte
		switch v := arg.(type) {
		case string:
			value = []byte(v)
		default:
			return ErrUnexpectedType
		}
		params = append(params, value)
	}
	bind := packets.Bind{
		ParameterValues: params,
	}

	// describe
	describe := packets.Describe{
		Which: 'P',
	}

	// execute
	execute := packets.Execute{
		// TODO(garet) hint for max rows?
	}

	// sync
	sync := zap.NewPacket(packets.TypeSync)

	w := resultWriter{
		result: reflect.ValueOf(result),
	}
	r := packetReader{
		packets: []zap.Packet{
			bind.IntoPacket(),
			describe.IntoPacket(),
			execute.IntoPacket(),
			sync,
		},
	}
	ctx := backends.Context{
		Peer: zap.CombinedReadWriter{
			Reader: &r,
			Writer: &w,
		},
	}

	if err := backends.Transaction(&ctx, server, parse.IntoPacket()); err != nil {
		return err
	}
	if w.err != nil {
		return w.err
	}

	return nil
}
