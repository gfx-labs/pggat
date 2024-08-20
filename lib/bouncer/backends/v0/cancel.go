package backends

import (
	"context"
	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
)

func Cancel(ctx context.Context, server *fed.Conn, key fed.BackendKey) error {
	p := packets.Startup{
		Mode: &packets.StartupPayloadControl{
			Mode: &packets.StartupPayloadControlPayloadCancel{
				ProcessID: key.ProcessID,
				SecretKey: key.SecretKey,
			},
		},
	}
	return server.WritePacket(ctx, &p)
}
