package psql

import (
	"errors"
	"reflect"
	"strconv"

	"pggat2/lib/fed"
	packets "pggat2/lib/fed/packets/v3.0"
)

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

func (T *resultWriter) WritePacket(packet fed.Packet) error {
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

var _ fed.Writer = (*resultWriter)(nil)
