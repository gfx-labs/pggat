package gat

import (
	"pggat2/lib/auth"
	"pggat2/lib/bouncer/frontends/v0"
	"pggat2/lib/zap"
)

type Gat struct {
	TestPool *Pool
}

func (T *Gat) Serve(client zap.Conn, acceptParams frontends.AcceptParams) error {
	defer func() {
		_ = client.Close()
	}()

	if acceptParams.CancelKey != [8]byte{} {
		// TODO(garet) execute cancel
		return nil
	}

	pool, err := T.GetPool(acceptParams.User, acceptParams.Database)
	if err != nil {
		return err
	}

	var credentials auth.Credentials
	if pool != nil {
		credentials = pool.GetCredentials()
	}

	authParams, err := frontends.Authenticate(client, frontends.AuthenticateOptions{
		Credentials: credentials,
	})
	if err != nil {
		return err
	}

	if pool == nil {
		return nil
	}

	return pool.Serve(client, acceptParams, authParams)
}

func (T *Gat) GetPool(user, database string) (*Pool, error) {
	return T.TestPool, nil
	return nil, nil // TODO(garet)
}
