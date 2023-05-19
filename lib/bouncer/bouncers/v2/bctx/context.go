package bctx

import (
	"pggat2/lib/bouncer/bouncers/v2/berr"
	"pggat2/lib/zap"
)

type Context struct {
	client, server zap.ReadWriter
	txn            byte
}

func MakeContext(client, server zap.ReadWriter) Context {
	return Context{
		client: client,
		server: server,
		txn:    'I',
	}
}

func (T *Context) SetTransactionState(state byte) {
	T.txn = state
}

func (T *Context) GetTransactionState() byte {
	return T.txn
}

func (T *Context) ClientRead() (zap.In, berr.Error) {
	in, err := T.client.Read()
	if err != nil {
		return zap.In{}, err
	}
	return in, nil
}

func (T *Context) ServerRead() (zap.In, berr.Error) {

}

func (T *Context) ClientSend(out zap.Out) berr.Error {

}

func (T *Context) ServerSend(out zap.Out) berr.Error {

}

func (T *Context) ClientProxy(in zap.In) berr.Error {
	return T.ClientSend(zap.InToOut(in))
}

func (T *Context) ServerProxy(in zap.In) berr.Error {
	return T.ServerSend(zap.InToOut(in))
}
