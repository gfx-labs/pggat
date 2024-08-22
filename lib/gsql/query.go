package gsql

import (
	"context"
	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
)

func Query(ctx context.Context, client *fed.Conn, results []any, query string) error {
	var q = packets.Query(query)
	if err := client.WritePacket(ctx, &q); err != nil {
		return err
	}

	if err := readQueryResults(ctx, client, results...); err != nil {
		return err
	}

	// make sure we receive ready for query
	packet, err := client.ReadPacket(ctx, true)
	if err != nil {
		return err
	}

	if packet.Type() != packets.TypeReadyForQuery {
		return ErrExpectedReadyForQuery
	}

	return nil
}

func readQueryResults(ctx context.Context, client *fed.Conn, results ...any) error {
	for _, result := range results {
		if err := readRows(ctx, client, result); err != nil {
			return err
		}
	}
	return nil
}
