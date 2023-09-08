package gsql

import (
	"reflect"
	"strconv"

	"pggat/lib/fed"
	packets "pggat/lib/fed/packets/v3.0"
)

func (T *Client) ExtendedQuery(result any, query string, args ...any) error {
	if len(args) == 0 {
		T.Query(query, result)
		return nil
	}

	T.mu.Lock()
	defer T.mu.Unlock()

	// parse
	parse := packets.Parse{
		Query: query,
	}
	T.queuePackets(parse.IntoPacket())

	// bind
	params := make([][]byte, 0, len(args))
outer:
	for _, arg := range args {
		var value []byte
		argr := reflect.ValueOf(arg)
		for argr.Kind() == reflect.Pointer {
			if argr.IsNil() {
				params = append(params, nil)
				continue outer
			}
			argr = argr.Elem()
		}
		switch argr.Kind() {
		case reflect.String:
			value = []byte(argr.String())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			value = []byte(strconv.FormatUint(argr.Uint(), 10))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			value = []byte(strconv.FormatInt(argr.Int(), 10))
		case reflect.Float32, reflect.Float64:
			value = []byte(strconv.FormatFloat(argr.Float(), 'f', -1, 64))
		case reflect.Bool:
			if argr.Bool() {
				value = []byte{'t'}
			} else {
				value = []byte{'f'}
			}
		case reflect.Invalid:
			value = nil
		default:
			return ErrUnexpectedType
		}
		params = append(params, value)
	}
	bind := packets.Bind{
		ParameterValues: params,
	}
	T.queuePackets(bind.IntoPacket())

	// describe
	describe := packets.Describe{
		Which: 'P',
	}
	T.queuePackets(describe.IntoPacket())

	// execute
	execute := packets.Execute{}
	T.queuePackets(execute.IntoPacket())

	// sync
	sync := fed.NewPacket(packets.TypeSync)
	T.queuePackets(sync)

	// result
	T.queueResults(NewQueryWriter(result))
	return nil
}
