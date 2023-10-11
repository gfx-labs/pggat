package gsql

import (
	"reflect"
	"strconv"

	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
)

func ExtendedQuery(client *Client, result any, query string, args ...any) error {
	if len(args) == 0 {
		Query(client, []any{result}, query)
		return nil
	}

	var pkts []fed.Packet

	// parse
	parse := packets.Parse{
		Query: query,
	}
	pkts = append(pkts, &parse)

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
		Parameters: params,
	}
	pkts = append(pkts, &bind)

	// describe
	describe := packets.Describe{
		Which: 'P',
	}
	pkts = append(pkts, &describe)

	// execute
	execute := packets.Execute{}
	pkts = append(pkts, &execute)

	// sync
	sync := packets.Sync{}
	pkts = append(pkts, &sync)

	// result
	client.Do(NewQueryWriter(result), pkts...)
	return nil
}
