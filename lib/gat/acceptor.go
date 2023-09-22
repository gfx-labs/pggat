package gat

import (
	"errors"
	"io"
	"net"

	"tuxpa.in/a/zlog/log"

	"pggat/lib/bouncer/frontends/v0"
	"pggat/lib/fed"
	"pggat/lib/util/beforeexit"
)

type Acceptor struct {
	Listener net.Listener
	Options  frontends.AcceptOptions
}

func Listen(network, address string, options frontends.AcceptOptions) (Acceptor, error) {
	listener, err := net.Listen(network, address)
	if err != nil {
		return Acceptor{}, err
	}
	if network == "unix" {
		// unix sockets are not cleaned up if process is stopped but i really wish they were
		beforeexit.Run(func() {
			_ = listener.Close()
		})
	}
	return Acceptor{
		Listener: listener,
		Options:  options,
	}, nil
}

func serve(client fed.Conn, acceptParams frontends.AcceptParams, pools *KeyedPools) error {
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
		log.Printf("pool not found: user=%s database=%s", acceptParams.User, acceptParams.Database)
		return nil
	}

	ctx := frontends.AuthenticateContext{
		Conn: client,
		Options: frontends.AuthenticateOptions{
			Credentials: p.GetCredentials(),
		},
	}
	authParams, err := frontends.Authenticate(&ctx)
	if err != nil {
		return err
	}

	pools.RegisterKey(authParams.BackendKey, acceptParams.User, acceptParams.Database)
	defer pools.UnregisterKey(authParams.BackendKey)

	return p.Serve(client, acceptParams.InitialParameters, authParams.BackendKey)
}

func Serve(acceptor Acceptor, pools *KeyedPools) error {
	for {
		netConn, err := acceptor.Listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			log.Print("error accepting connection: ", err)
			continue
		}
		conn := fed.WrapNetConn(netConn)

		go func() {
			defer func() {
				_ = conn.Close()
			}()

			ctx := frontends.AcceptContext{
				Conn:    conn,
				Options: acceptor.Options,
			}
			acceptParams, acceptErr := frontends.Accept(&ctx)
			if acceptErr != nil {
				log.Print("error accepting client: ", acceptErr)
				return
			}

			err = serve(conn, acceptParams, pools)
			if err != nil && !errors.Is(err, io.EOF) {
				log.Print("error serving client: ", err)
				return
			}
		}()
	}
}

func ListenAndServe(network, address string, options frontends.AcceptOptions, pools *KeyedPools) error {
	listener, err := Listen(network, address, options)
	if err != nil {
		return err
	}
	return Serve(listener, pools)
}
