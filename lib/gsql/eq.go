package gsql

import (
	"reflect"
	"strconv"

	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
)

func ExtendedQuery(client fed.Conn, result any, query string, args ...any) error {
	if len(args) == 0 {
		return Query(client, []any{result}, query)
	}

	// parse
	parse := packets.Parse{
		Query: query,
	}
	if err := client.WritePacket(&parse); err != nil {
		return err
	}

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
	if err := client.WritePacket(&bind); err != nil {
		return err
	}

	// describe
	describe := packets.Describe{
		Which: 'P',
	}
	if err := client.WritePacket(&describe); err != nil {
		return err
	}

	// execute
	execute := packets.Execute{}
	if err := client.WritePacket(&execute); err != nil {
		return err
	}

	// sync
	sync := packets.Sync{}
	if err := client.WritePacket(&sync); err != nil {
		return err
	}

	// result
	if err := readQueryResults(client, result); err != nil {
		return err
	}

	// make sure we receive ready for query
	packet, err := client.ReadPacket(true)
	if err != nil {
		return err
	}

	if packet.Type() != packets.TypeReadyForQuery {
		return ErrExpectedReadyForQuery
	}

	return nil
}
