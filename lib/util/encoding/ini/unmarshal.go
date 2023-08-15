package ini

import (
	"bytes"
	"errors"
	"reflect"
	"strconv"
)

type Unmarshaller interface {
	UnmarshalINI(bytes []byte) error
}

var (
	unmarshaller = reflect.TypeOf((*Unmarshaller)(nil)).Elem()
)

func get(rv reflect.Value, key []byte, fn func(rv reflect.Value) error) error {
outer:
	for {
		switch rv.Kind() {
		case reflect.Pointer:
			if rv.IsNil() {
				rv.Set(reflect.New(rv.Type().Elem()))
			}
			rv = rv.Elem()
		case reflect.Struct, reflect.Map:
			break outer
		default:
			return nil
		}
	}

	switch rv.Kind() {
	case reflect.Struct:
		keystr := string(key)
		rt := rv.Type()
		numFields := rt.NumField()
		for i := 0; i < numFields; i++ {
			field := rt.Field(i)
			if !field.IsExported() {
				continue
			}
			name, ok := field.Tag.Lookup("ini")
			if !ok {
				name = field.Name
			}
			if name == "*" {
				return get(rv.Field(i), key, fn)
			}
			if name == keystr {
				return fn(rv.Field(i))
			}
		}
		return nil
	case reflect.Map:
		rt := rv.Type()
		rtKey := rt.Key()
		if rv.IsNil() {
			rv.Set(reflect.MakeMap(rt))
		}
		k := reflect.New(rtKey).Elem()
		if err := set(k, key); err != nil {
			return err
		}
		v := reflect.New(rt.Elem()).Elem()
		if err := fn(v); err != nil {
			return err
		}
		rv.SetMapIndex(k, v)
		return nil
	default:
		panic("unreachable")
	}
}

func set(rv reflect.Value, value []byte) error {
outer:
	for {
		switch rv.Kind() {
		case reflect.Pointer:
			if rv.IsNil() {
				rv.Set(reflect.New(rv.Type().Elem()))
			}
			rv = rv.Elem()
		case reflect.Struct,
			reflect.Map,
			reflect.String,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64,
			reflect.Array, reflect.Slice:
			break outer
		default:
			return errors.New("cannot set value of this type")
		}
	}

	rt := rv.Type()
	if rt.Implements(unmarshaller) {
		rvu := rv.Interface().(Unmarshaller)
		return rvu.UnmarshalINI(value)
	}
	if rv.CanAddr() && reflect.PointerTo(rt).Implements(unmarshaller) {
		rvu := rv.Addr().Interface().(Unmarshaller)
		return rvu.UnmarshalINI(value)
	}

	switch rv.Kind() {
	case reflect.Struct, reflect.Map:
		fields := bytes.Fields(value)
		for _, field := range fields {
			k, v, ok := bytes.Cut(field, []byte{'='})
			if !ok {
				return errors.New("expected key=value")
			}
			if err := get(rv, k, func(rvValue reflect.Value) error {
				return set(rvValue, v)
			}); err != nil {
				return err
			}
		}
		return nil
	case reflect.Array:
		items := bytes.Split(value, []byte{','})
		if len(items) != rv.Len() {
			return errors.New("wrong length for array")
		}
		for i, item := range items {
			if err := set(rv.Index(i), bytes.TrimSpace(item)); err != nil {
				return err
			}
		}
		return nil
	case reflect.Slice:
		items := bytes.Split(value, []byte{','})
		slice := reflect.MakeSlice(rt.Elem(), len(items), len(items))
		for i, item := range items {
			if err := set(slice.Index(i), bytes.TrimSpace(item)); err != nil {
				return err
			}
		}
		rv.Set(slice)
		return nil
	case reflect.String:
		rv.SetString(string(value))
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := strconv.ParseInt(string(value), 10, 64)
		if err != nil {
			return err
		}
		rv.SetInt(v)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, err := strconv.ParseUint(string(value), 10, 64)
		if err != nil {
			return err
		}
		rv.SetUint(v)
		return nil
	case reflect.Float32, reflect.Float64:
		v, err := strconv.ParseFloat(string(value), 64)
		if err != nil {
			return err
		}
		rv.SetFloat(v)
		return nil
	default:
		panic("unreachable")
	}
}

func setpath(rv reflect.Value, section, key, value []byte) error {
	if len(section) == 0 {
		return get(rv, key, func(entry reflect.Value) error {
			return set(entry, value)
		})
	}
	return get(rv, section, func(sec reflect.Value) error {
		return get(sec, key, func(entry reflect.Value) error {
			return set(entry, value)
		})
	})
}

func Unmarshal(data []byte, v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return errors.New("expected pointer to non nil")
	}
	rv = rv.Elem()

	var section []byte

	var line []byte
	for {
		line, data, _ = bytes.Cut(data, []byte{'\n'})
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			if len(data) == 0 {
				break
			}
			continue
		}

		// %include
		if bytes.HasPrefix(line, []byte("%include")) {
			return errors.New("%include directive found. Use ReadFile instead")
		}

		// comment
		if bytes.HasPrefix(line, []byte{';'}) || bytes.HasPrefix(line, []byte{'#'}) {
			continue
		}

		// section
		if bytes.HasPrefix(line, []byte{'['}) && bytes.HasSuffix(line, []byte{']'}) {
			section = line[1 : len(line)-1]
			continue
		}

		// kv pair
		key, value, ok := bytes.Cut(line, []byte{'='})
		if !ok {
			return errors.New("expected key = value")
		}
		key = bytes.TrimSpace(key)
		value = bytes.TrimSpace(value)

		if err := setpath(rv, section, key, value); err != nil {
			return err
		}
	}

	return nil
}
