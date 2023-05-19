package v2

import (
	"pggat2/lib/bouncer/bouncers/v2/bctx"
	"pggat2/lib/bouncer/bouncers/v2/rclient"
	"pggat2/lib/bouncer/bouncers/v2/rserver"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

func copyIn(ctx *bctx.Context) {

}

func copyOut(ctx *bctx.Context) {

}

func query(ctx *bctx.Context) {

}

func functionCall(ctx *bctx.Context) {

}

func sync(ctx *bctx.Context) {

}

func eqp(ctx *bctx.Context) {
	for {
		in, err := ctx.ClientRead()
		if err != nil {
			rserver.EQP(ctx, err)
			rclient.EQP(ctx, err)
			return
		}

		switch in.Type() {
		case packets.Sync:
			if err = ctx.ServerProxy(in); err != nil {
				rserver.EQP(ctx, err)
				rclient.Sync(ctx, err)
				rclient.EQP(ctx, err)
				return
			}
			sync(ctx)
			return
		case packets.Parse, packets.Bind, packets.Close, packets.Describe, packets.Execute, packets.Flush:
			if err = ctx.ServerProxy(in); err != nil {
				rserver.EQP(ctx, err)
				rclient.EQP(ctx, err)
				return
			}
		}
	}
}

func transaction0(ctx *bctx.Context) {
	in, err := ctx.ClientRead()
	if err != nil {
		// TODO(garet)
		// PROBLEM: should this actually break out of the transaction? probably not
		// because if the error is just prepared statement doesn't exist we should just enter a failed txn block
		rserver.Transaction(ctx, err)
		rclient.Transaction(ctx, err)
		return
	}

	switch in.Type() {
	case packets.Query:
		if err = ctx.ServerProxy(in); err != nil {
			rserver.Transaction(ctx, err)
			rclient.Query(ctx, err)
			rclient.Transaction(ctx, err)
			return
		}
		query(ctx)
	case packets.FunctionCall:
		if err = ctx.ServerProxy(in); err != nil {
			rserver.Transaction(ctx, err)
			rclient.FunctionCall(ctx, err)
			rclient.Transaction(ctx, err)
			return
		}
		functionCall(ctx)
	case packets.Sync:
		// TODO(garet) can this be turned into a phony call, not actually directed to server?
		if err = ctx.ServerProxy(in); err != nil {
			rserver.Transaction(ctx, err)
			rclient.Sync(ctx, err)
			rclient.Transaction(ctx, err)
			return
		}
		sync(ctx)
	case packets.Parse, packets.Bind, packets.Close, packets.Describe, packets.Execute, packets.Flush:
		if err = ctx.ServerProxy(in); err != nil {
			rserver.Transaction(ctx, err)
			rclient.EQP(ctx, err)
			rclient.Transaction(ctx, err)
			return
		}
		eqp(ctx)
	default:
		panic("unknown packet")
	}
}

func transaction(ctx *bctx.Context) {
	ctx.SetTransactionState('T')

	for ctx.GetTransactionState() != 'I' {
		transaction0(ctx)
	}
}

func Bounce(client, server zap.ReadWriter) {
	ctx := bctx.MakeContext(client, server)
	transaction(&ctx)
}
