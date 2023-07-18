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

func (T *Context) ClientRead(packet *zap.Packet) berr.Error {
	err := T.client.Read(packet)
	if err != nil {
		return berr.MakeClient(err)
	}
	return nil
}

func (T *Context) ServerRead(packet *zap.Packet) berr.Error {
	err := T.server.Read(packet)
	if err != nil {
		return berr.MakeServer(err)
	}
	return nil
}

func (T *Context) ClientWrite(packet *zap.Packet) berr.Error {
	err := T.client.Write(packet)
	if err != nil {
		return berr.MakeClient(err)
	}
	return nil
}

func (T *Context) ServerWrite(packet *zap.Packet) berr.Error {
	err := T.server.Write(packet)
	if err != nil {
		return berr.MakeServer(err)
	}
	return nil
}

func (T *Context) ClientWriteV(packets *zap.Packets) berr.Error {
	err := T.client.WriteV(packets)
	if err != nil {
		return berr.MakeClient(err)
	}
	return nil
}

func (T *Context) ServerWriteV(packets *zap.Packets) berr.Error {
	err := T.server.WriteV(packets)
	if err != nil {
		return berr.MakeServer(err)
	}
	return nil
}
