package bctx

import (
	"pggat2/lib/bouncer/bouncers/v2/berr"
	"pggat2/lib/zap"
)

type Context struct {
	client, server zap.ReadWriter

	// state (for flow and recovery)
	CopyOut      bool
	CopyIn       bool
	Query        bool
	FunctionCall bool
	Sync         bool
	EQP          bool
	TxState      byte
}

func MakeContext(client, server zap.ReadWriter) Context {
	return Context{
		client:  client,
		server:  server,
		TxState: 'I',
	}
}

func (T *Context) ClientRead() (zap.In, berr.Error) {
	in, err := T.client.Read()
	if err != nil {
		return zap.In{}, berr.MakeClient(err)
	}
	return in, nil
}

func (T *Context) ServerRead() (zap.In, berr.Error) {
	in, err := T.server.Read()
	if err != nil {
		return zap.In{}, berr.MakeServer(err)
	}
	return in, nil
}

func (T *Context) ClientSend(out zap.Out) berr.Error {
	err := T.client.Send(out)
	if err != nil {
		return berr.MakeClient(err)
	}
	return nil
}

func (T *Context) ServerSend(out zap.Out) berr.Error {
	err := T.server.Send(out)
	if err != nil {
		return berr.MakeServer(err)
	}
	return nil
}

func (T *Context) ClientProxy(in zap.In) berr.Error {
	return T.ClientSend(zap.InToOut(in))
}

func (T *Context) ServerProxy(in zap.In) berr.Error {
	return T.ServerSend(zap.InToOut(in))
}

func (T *Context) ClientWrite() zap.Out {
	return T.client.Write()
}

func (T *Context) ServerWrite() zap.Out {
	return T.server.Write()
}
