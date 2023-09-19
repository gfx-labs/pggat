package pool

import (
	"pggat/lib/bouncer/backends/v0"
	packets "pggat/lib/fed/packets/v3.0"
	"pggat/lib/gat/metrics"
	"pggat/lib/middleware/middlewares/eqp"
	"pggat/lib/middleware/middlewares/ps"
	"pggat/lib/util/slices"
)

func Pair(options Options, client *Client, server *Server) (clientErr, serverErr error) {
	defer func() {
		client.SetState(metrics.ConnStateActive, server.GetID())
		server.SetState(metrics.ConnStateActive, client.GetID())
	}()

	if options.ParameterStatusSync != ParameterStatusSyncNone || options.ExtendedQuerySync {
		client.SetState(metrics.ConnStatePairing, server.GetID())
		server.SetState(metrics.ConnStatePairing, client.GetID())
	}

	switch options.ParameterStatusSync {
	case ParameterStatusSyncDynamic:
		clientErr, serverErr = ps.Sync(options.TrackedParameters, client.GetReadWriter(), client.GetPS(), server.GetReadWriter(), server.GetPS())
	case ParameterStatusSyncInitial:
		clientErr, serverErr = SyncInitialParameters(options, client, server)
	}

	if clientErr != nil || serverErr != nil {
		return
	}

	if options.ExtendedQuerySync {
		serverErr = eqp.Sync(client.GetEQP(), server.GetReadWriter(), server.GetEQP())
	}

	return
}

func SyncInitialParameters(options Options, client *Client, server *Server) (clientErr, serverErr error) {
	clientParams := client.GetInitialParameters()
	serverParams := server.GetInitialParameters()

	for key, value := range clientParams {
		// skip already set params
		if serverParams[key] == value {
			p := packets.ParameterStatus{
				Key:   key.String(),
				Value: serverParams[key],
			}
			clientErr = client.GetConn().WritePacket(p.IntoPacket(nil))
			if clientErr != nil {
				return
			}
			continue
		}

		setServer := slices.Contains(options.TrackedParameters, key)

		if !setServer {
			value = serverParams[key]
		}

		p := packets.ParameterStatus{
			Key:   key.String(),
			Value: value,
		}
		clientErr = client.GetConn().WritePacket(p.IntoPacket(nil))
		if clientErr != nil {
			return
		}

		if !setServer {
			continue
		}

		ctx := backends.Context{
			Server: server.GetReadWriter(),
		}
		serverErr = backends.SetParameter(&ctx, key, value)
		if serverErr != nil {
			return
		}
	}

	for key, value := range serverParams {
		if _, ok := clientParams[key]; ok {
			continue
		}

		// Don't need to run reset on server because it will reset it to the initial value

		// send to client
		p := packets.ParameterStatus{
			Key:   key.String(),
			Value: value,
		}
		clientErr = client.GetConn().WritePacket(p.IntoPacket(nil))
		if clientErr != nil {
			return
		}
	}

	return

}

func TransactionComplete(client *Client, server *Server) {
	client.TransactionComplete()
	server.TransactionComplete()
}
