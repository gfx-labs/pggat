package pool

import (
	packets "pggat/lib/fed/packets/v3.0"
	"pggat/lib/gat/metrics"
	"pggat/lib/middleware/middlewares/ps"
	"pggat/lib/util/slices"
)

func Pair(options Options, client *Client, server *Server) (clientErr, serverErr error) {
	client.SetState(metrics.ConnStateActive, server.GetID())
	server.SetState(metrics.ConnStateActive, client.GetID())

	switch options.ParameterStatusSync {
	case ParameterStatusSyncDynamic:
		clientErr, serverErr = ps.Sync(options.TrackedParameters, client.GetConn(), client.GetPS(), server.GetConn(), server.GetPS())
	case ParameterStatusSyncInitial:
		clientErr, serverErr = SyncInitialParameters(options, client, server)
	}

	if options.ExtendedQuerySync {
		server.GetEQP().SetClient(client.GetEQP())
	}

	return
}

func SyncInitialParameters(options Options, client *Client, server *Server) (clientErr, serverErr error) {
	clientParams := client.GetInitialParameters()
	serverParams := server.GetInitialParameters()

	for key, value := range clientParams {
		setServer := slices.Contains(options.TrackedParameters, key)

		// skip already set params
		if serverParams[key] == value {
			setServer = false
		} else if !setServer {
			value = serverParams[key]
		}

		p := packets.ParameterStatus{
			Key:   key.String(),
			Value: serverParams[key],
		}
		clientErr = client.GetConn().WritePacket(p.IntoPacket())
		if clientErr != nil {
			return
		}

		if !setServer {
			continue
		}

		serverErr = backends.SetParameter(new(backends.Context), server.GetConn(), key, value)
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
		clientErr = client.GetConn().WritePacket(p.IntoPacket())
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
