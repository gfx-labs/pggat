package gat

import (
	"net"

	"pggat2/lib/bouncer/frontends/v0"
	"pggat2/lib/fed"
)

type Acceptor struct {
	Listener net.Listener
	Options  frontends.AcceptOptions
}

func (T Acceptor) Accept() (fed.Conn, frontends.AcceptParams, error) {
	netConn, err := T.Listener.Accept()
	if err != nil {
		return nil, frontends.AcceptParams{}, err
	}
	conn := fed.WrapNetConn(netConn)
	params, err := frontends.Accept(conn, T.Options)
	if err != nil {
		_ = conn.Close()
		return nil, frontends.AcceptParams{}, err
	}
	return conn, params, nil
}

func Listen(network, address string, options frontends.AcceptOptions) (Acceptor, error) {
	listener, err := net.Listen(network, address)
	if err != nil {
		return Acceptor{}, err
	}
	return Acceptor{
		Listener: listener,
		Options:  options,
	}, nil
}

func serve(client fed.Conn, acceptParams frontends.AcceptParams, pools Pools) error {
	defer func() {
		_ = client.Close()
	}()

	if acceptParams.CancelKey != [8]byte{} {
		p := pools.LookupKey(acceptParams.CancelKey)
		if p == nil {
			return nil
		}

		return p.Cancel(acceptParams.CancelKey)
	}

	p := pools.Lookup(acceptParams.User, acceptParams.Database)

	if p == nil {
		return nil
	}

	authParams, err := frontends.Authenticate(client, frontends.AuthenticateOptions{
		Credentials: p.GetCredentials(),
	})
	if err != nil {
		return err
	}

	pools.RegisterKey(authParams.BackendKey, acceptParams.User, acceptParams.Database)
	defer pools.UnregisterKey(authParams.BackendKey)

	return p.Serve(client, acceptParams, authParams)
}

func Serve(acceptor Acceptor, pools Pools) error {
	for {
		conn, acceptParams, err := acceptor.Accept()
		if err != nil {
			// log.Println("error accepting", err)
			continue
		}
		go func() {
			_ = serve(conn, acceptParams, pools)
			// log.Println("error serving", err)
		}()
	}
}

func ListenAndServe(network, address string, options frontends.AcceptOptions, pools Pools) error {
	listener, err := Listen(network, address, options)
	if err != nil {
		return err
	}
	return Serve(listener, pools)
}
