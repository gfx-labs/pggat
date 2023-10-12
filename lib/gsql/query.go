package gsql

import (
	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
)

func Query(client *fed.Conn, results []any, query string) error {
	var q = packets.Query(query)
	if err := client.WritePacket(&q); err != nil {
		return err
	}

	if err := readQueryResults(client, results...); err != nil {
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

func readQueryResults(client *fed.Conn, results ...any) error {
	for _, result := range results {
		if err := readRows(client, result); err != nil {
			return err
		}
	}
	return nil
}
