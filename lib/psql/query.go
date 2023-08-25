package psql

import (
	"crypto/tls"
	"errors"
	"io"
	"reflect"

	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

type resultReader struct {
	result reflect.Value
	rd     packets.RowDescription
	row    int
}

func (T *resultReader) EnableSSLClient(_ *tls.Config) error {
	return errors.New("ssl not supported")
}

func (T *resultReader) EnableSSLServer(_ *tls.Config) error {
	return errors.New("ssl not supported")
}

func (T *resultReader) ReadByte() (byte, error) {
	return 0, io.EOF
}

func (T *resultReader) ReadPacket(_ bool) (zap.Packet, error) {
	return nil, io.EOF
}

func (T *resultReader) WriteByte(_ byte) error {
	return nil
}

func (T *resultReader) set(i int, row []byte) error {
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
	default:
		return ErrUnexpectedType
	}
}

func (T *resultReader) WritePacket(packet zap.Packet) error {
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
				return err
			}
		}
		T.row += 1
	}
	return nil
}

func (T *resultReader) Close() error {
	return nil
}

var _ zap.ReadWriter = (*resultReader)(nil)

func Query(server zap.ReadWriter, query string, result any) error {
	res := resultReader{
		result: reflect.ValueOf(result),
	}
	ctx := backends.Context{
		Peer: &res,
	}
	if err := backends.QueryString(&ctx, server, query); err != nil {
		return err
	}

	return nil
}
