package ini

import (
	"bytes"
	"errors"
	"reflect"
	"strconv"
	"strings"
)

func get(rv reflect.Value, key string, fn func(rv reflect.Value) error) error {
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
			if name == key {
				return fn(rv.Field(i))
			}
		}
		return nil
	case reflect.Map:
		rt := rv.Type()
		rtKey := rt.Key()
		if rtKey.Kind() != reflect.String {
			return nil
		}
		if rv.IsNil() {
			rv.Set(reflect.MakeMap(rt))
		}
		k := reflect.New(rtKey).Elem()
		k.SetString(key)
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

func set(rv reflect.Value, value string) error {
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
			reflect.Slice:
			break outer
		default:
			return errors.New("cannot set value of this type")
		}
	}

	switch rv.Kind() {
	case reflect.Struct, reflect.Map:
		fields := strings.Fields(value)
		for _, field := range fields {
			k, v, ok := strings.Cut(field, "=")
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
		items := strings.Split(value, ",")
		if len(items) != rv.Len() {
			return errors.New("wrong length for array")
		}
		for i, item := range items {
			if err := set(rv.Index(i), strings.TrimSpace(item)); err != nil {
				return err
			}
		}
		return nil
	case reflect.Slice:
		items := strings.Split(value, ",")
		slice := reflect.MakeSlice(rv.Type().Elem(), len(items), len(items))
		for i, item := range items {
			if err := set(slice.Index(i), strings.TrimSpace(item)); err != nil {
				return err
			}
		}
		rv.Set(slice)
		return nil
	case reflect.String:
		rv.SetString(value)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		rv.SetInt(v)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		rv.SetUint(v)
		return nil
	case reflect.Float32, reflect.Float64:
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		rv.SetFloat(v)
		return nil
	default:
		panic("unreachable")
	}
}

func setpath(rv reflect.Value, section, key, value string) error {
	if section == "" {
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

	var section string

	var line []byte
	for {
		line, data, _ = bytes.Cut(data, []byte{'\n'})
		if len(line) == 0 {
			if len(data) == 0 {
				break
			}
			continue
		}

		line = bytes.TrimSpace(line)

		// comment
		if bytes.HasPrefix(line, []byte{';'}) || bytes.HasPrefix(line, []byte{'#'}) {
			continue
		}

		// section
		if bytes.HasPrefix(line, []byte{'['}) && bytes.HasSuffix(line, []byte{']'}) {
			section = string(line[1 : len(line)-1])
			continue
		}

		// kv pair
		key, value, ok := bytes.Cut(line, []byte{'='})
		if !ok {
			return errors.New("expected key = value")
		}
		key = bytes.TrimSpace(key)
		value = bytes.TrimSpace(value)

		if err := setpath(rv, section, string(key), string(value)); err != nil {
			return err
		}
	}

	return nil
}
