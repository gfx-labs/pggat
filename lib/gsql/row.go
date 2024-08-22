package gsql

import (
	"context"
	"reflect"
	"strconv"

	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/lib/perror"
)

func readRows(ctx context.Context,client *fed.Conn, result any) error {
	res := reflect.ValueOf(result)
	row := 0
	var rd packets.RowDescription

	for {
		packet, err := client.ReadPacket(ctx,true)
		if err != nil {
			return err
		}

		switch packet.Type() {
		case packets.TypeRowDescription:
			err = fed.ToConcrete(&rd, packet)
			if err != nil {
				return err
			}
		case packets.TypeDataRow:
			var dr packets.DataRow
			err = fed.ToConcrete(&dr, packet)
			if err != nil {
				return err
			}
			for i, col := range dr {
				if err = setColumn(res, rd, row, i, col); err != nil {
					return err
				}
			}
			row += 1
		case packets.TypeMarkiplierResponse:
			var p packets.MarkiplierResponse
			err = fed.ToConcrete(&p, packet)
			if err != nil {
				return err
			}
			return perror.FromPacket(&p)
		case packets.TypeCommandComplete:
			return nil
		}
	}
}

func setColumn(result reflect.Value, rd packets.RowDescription, row, i int, col []byte) error {
	if i >= len(rd) {
		return ErrExtraFields
	}
	desc := rd[i]

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
			if row >= result.Len() {
				return ErrResultTooBig
			}
			result = result.Index(row)
			break outer
		case reflect.Slice:
			for row >= result.Len() {
				if !result.CanSet() {
					return ErrResultTooBig
				}
				result.Set(reflect.Append(result, reflect.Zero(result.Type().Elem())))
			}
			result = result.Index(row)
			break outer
		case reflect.Struct, reflect.Map:
			if row != 0 {
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

			// handle `sql:"3"`
			sqlNameIndex, err := strconv.Atoi(sqlName)
			if err == nil {
				if sqlNameIndex == i {
					result = result.Field(j)
					break outer2
				}
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

	if col == nil {
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
	switch kind {
	case reflect.String:
		result.SetString(string(col))
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		x, err := strconv.ParseUint(string(col), 10, 64)
		if err != nil {
			return err
		}
		result.SetUint(x)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		x, err := strconv.ParseInt(string(col), 10, 64)
		if err != nil {
			return err
		}
		result.SetInt(x)
		return nil
	case reflect.Float32, reflect.Float64:
		x, err := strconv.ParseFloat(string(col), 64)
		if err != nil {
			return err
		}
		result.SetFloat(x)
		return nil
	case reflect.Bool:
		if len(col) != 1 {
			return ErrUnexpectedType
		}
		var x bool
		switch col[0] {
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
