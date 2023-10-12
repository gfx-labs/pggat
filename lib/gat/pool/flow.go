package pool

import (
	"gfx.cafe/gfx/pggat/lib/bouncer/backends/v0"
	"gfx.cafe/gfx/pggat/lib/fed/middlewares/eqp"
	"gfx.cafe/gfx/pggat/lib/fed/middlewares/ps"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
	"gfx.cafe/gfx/pggat/lib/util/slices"
)

func pair(options Config, client *pooledClient, server *pooledServer) (clientErr, serverErr error) {
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
		clientErr, serverErr = ps.Sync(options.TrackedParameters, client.GetConn(), client.GetPS(), server.GetConn(), server.GetPS())
	case ParameterStatusSyncInitial:
		clientErr, serverErr = syncInitialParameters(options, client, server)
	}

	if clientErr != nil || serverErr != nil {
		return
	}

	if options.ExtendedQuerySync {
		serverErr = eqp.Sync(client.GetEQP(), server.GetConn(), server.GetEQP())
	}

	return
}

func syncInitialParameters(options Config, client *pooledClient, server *pooledServer) (clientErr, serverErr error) {
	clientParams := client.GetInitialParameters()
	serverParams := server.GetInitialParameters()

	for key, value := range clientParams {
		// skip already set params
		if serverParams[key] == value {
			p := packets.ParameterStatus{
				Key:   key.String(),
				Value: serverParams[key],
			}
			clientErr = client.GetConn().WritePacket(&p)
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
		clientErr = client.GetConn().WritePacket(&p)
		if clientErr != nil {
			return
		}

		if !setServer {
			continue
		}

		serverErr, _ = backends.SetParameter(server.GetConn(), nil, key, value)
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
		clientErr = client.GetConn().WritePacket(&p)
		if clientErr != nil {
			return
		}
	}

	return

}

func transactionComplete(client *pooledClient, server *pooledServer) {
	client.TransactionComplete()
	server.TransactionComplete()
}
