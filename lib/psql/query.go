package psql

import (
	"reflect"
	"strconv"

	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

func Query(server zap.ReadWriter, result any, query string, args ...any) error {
	res := reflect.ValueOf(result)

	if len(args) == 0 {
		// simple query

		w := resultWriter{
			result: res,
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
	execute := packets.Execute{}

	// sync
	sync := zap.NewPacket(packets.TypeSync)

	w := resultWriter{
		result: res,
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
