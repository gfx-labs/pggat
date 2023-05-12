package bctx

import (
	"pggat2/lib/bouncer/bouncers/v1/berr"
	"pggat2/lib/util/decorator"
	"pggat2/lib/zap"
)

type Context struct {
	noCopy decorator.NoCopy

	transaction  bool
	query        bool
	copyIn       bool
	copyOut      bool
	functionCall bool
	eqp          bool

	client, server zap.ReadWriter
}

func MakeContext(client, server zap.ReadWriter) Context {
	return Context{
		client: client,
		server: server,
	}
}

// io helper funcs

func (T *Context) ClientRead() (zap.In, berr.Error) {

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

func (T *Context) BeginEQP() {
	if T.eqp {
		panic("already in EQP")
	}
	T.eqp = true
}

func (T *Context) InEQP() bool {
	return T.eqp
}

func (T *Context) EndEQP() {
	T.eqp = false
}
