package bctx

import (
	"time"

	"pggat2/lib/bouncer/bouncers/v1/berr"
	"pggat2/lib/perror"
	"pggat2/lib/util/decorator"
	"pggat2/lib/zap"
)

type Context struct {
	noCopy decorator.NoCopy

	clientIdleTimeout time.Duration

	transaction  bool
	query        bool
	copyIn       bool
	copyOut      bool
	functionCall bool
	sync         bool

	client, server zap.ReadWriter
}

func MakeContext(client, server zap.ReadWriter, clientIdleTimeout time.Duration) Context {
	return Context{
		clientIdleTimeout: clientIdleTimeout,

		client: client,
		server: server,
	}
}

func (T *Context) Done() {
	// if it fails, it's not my problem - Garet, May 12, 2023
	_ = T.client.SetReadDeadline(time.Time{})
}

// io helper funcs

func (T *Context) ClientRead() (zap.In, berr.Error) {
	if T.clientIdleTimeout > 0 {
		err := T.client.SetReadDeadline(time.Now().Add(T.clientIdleTimeout))
		if err != nil {
			return zap.In{}, berr.Client{
				Error: perror.Wrap(err),
			}
		}
	}
	in, err := T.client.Read()
	if err != nil {
		return zap.In{}, berr.Client{
			Error: perror.Wrap(err),
		}
	}
	return in, nil
}

func (T *Context) ServerRead() (zap.In, berr.Error) {
	in, err := T.server.Read()
	if err != nil {
		return zap.In{}, berr.Server{
			Error: err,
		}
	}
	return in, nil
}

func (T *Context) ClientWrite() zap.Out {
	return T.client.Write()
}

func (T *Context) ServerWrite() zap.Out {
	return T.server.Write()
}

func (T *Context) ClientSend(out zap.Out) berr.Error {
	err := T.client.Send(out)
	if err != nil {
		return berr.Client{
			Error: perror.Wrap(err),
		}
	}
	return nil
}

func (T *Context) ServerSend(out zap.Out) berr.Error {
	err := T.server.Send(out)
	if err != nil {
		return berr.Server{
			Error: err,
		}
	}
	return nil
}

func (T *Context) ClientProxy(in zap.In) berr.Error {
	return T.ClientSend(zap.InToOut(in))
}

func (T *Context) ServerProxy(in zap.In) berr.Error {
	return T.ServerSend(zap.InToOut(in))
}

// state helper funcs

func (T *Context) BeginTransaction() {
	if T.transaction {
		panic("already in transaction")
	}
	T.transaction = true
}

func (T *Context) InTransaction() bool {
	return T.transaction
}

func (T *Context) EndTransaction() {
	T.transaction = false
}

func (T *Context) BeginQuery() {
	if T.query {
		panic("already in query")
	}
	T.query = true
}

func (T *Context) InQuery() bool {
	return T.query
}

func (T *Context) EndQuery() {
	T.query = false
}

func (T *Context) BeginCopyIn() {
	if T.copyIn {
		panic("already in copyIn")
	}
	T.copyIn = true
}

func (T *Context) InCopyIn() bool {
	return T.copyIn
}

func (T *Context) EndCopyIn() {
	T.copyIn = false
}

func (T *Context) BeginCopyOut() {
	if T.copyOut {
		panic("already in copyOut")
	}
	T.copyOut = true
}

func (T *Context) InCopyOut() bool {
	return T.copyOut
}

func (T *Context) EndCopyOut() {
	T.copyOut = false
}

func (T *Context) BeginFunctionCall() {
	if T.functionCall {
		panic("already in functionCall")
	}
	T.functionCall = true
}

func (T *Context) InFunctionCall() bool {
	return T.functionCall
}

func (T *Context) EndFunctionCall() {
	T.functionCall = false
}

func (T *Context) BeginSync() {
	if T.sync {
		panic("already in sync")
	}
	T.sync = true
}

func (T *Context) InSync() bool {
	return T.sync
}

func (T *Context) EndSync() {
	T.sync = false
}
