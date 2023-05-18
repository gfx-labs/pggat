package bouncers

import (
	"fmt"
	"log"

	"pggat2/lib/bouncer/bouncers/v1/bctx"
	"pggat2/lib/bouncer/bouncers/v1/berr"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

// serverRead is a wrapper for bctx.Context's ServerRead but it handles async operations
func serverRead(ctx *bctx.Context) (zap.In, berr.Error) {
	for {
		in, err := ctx.ServerRead()
		if err != nil {
			return zap.In{}, err
		}
		switch in.Type() {
		case packets.NoticeResponse,
			packets.ParameterStatus,
			packets.NotificationResponse:
			err = ctx.ClientProxy(in)
			if err != nil {
				return zap.In{}, err
			}
		default:
			return in, nil
		}
	}
}

func readyForQuery(ctx *bctx.Context, in zap.In) berr.Error {
	ctx.EndEQP()
	state, ok := packets.ReadReadyForQuery(in)
	if !ok {
		return berr.ServerBadPacket
	}
	if state == 'I' {
		ctx.EndTransaction()
	}
	return nil
}

func copyIn0(ctx *bctx.Context) berr.Error {
	in, err := ctx.ClientRead()
	if err != nil {
		return err
	}

	switch in.Type() {
	case packets.CopyData:
		return ctx.ServerProxy(in)
	case packets.CopyDone, packets.CopyFail:
		ctx.EndCopyIn()
		return ctx.ServerProxy(in)
	default:
		return berr.ClientProtocolError
	}
}

func copyInRecoverServer(ctx *bctx.Context, err berr.Error) {
	// send copyFail to server, will stop server copy
	out := ctx.ServerWrite()
	out.Type(packets.CopyFail)
	out.String(fmt.Sprintf("client error: %s", err.String()))
	_ = ctx.ServerSend(out)

	ctx.EndCopyIn()
}

func copyInRecoverClient(ctx *bctx.Context, err berr.Error) {
	// send error to client, will stop client copy
	out := ctx.ClientWrite()
	packets.WriteErrorResponse(out, err.PError())
	_ = ctx.ClientSend(out)

	ctx.EndCopyIn()
}

func copyInRecover(ctx *bctx.Context, err berr.Error) {
	copyInRecoverServer(ctx, err)
	copyInRecoverClient(ctx, err)
}

func copyIn(ctx *bctx.Context) {
	ctx.BeginCopyIn()

	for ctx.InCopyIn() {
		err := copyIn0(ctx)
		if err != nil {
			copyInRecover(ctx, err)
			return
		}
	}
}

func copyOut0(ctx *bctx.Context) berr.Error {
	in, err := serverRead(ctx)
	if err != nil {
		return err
	}

	switch in.Type() {
	case packets.CopyData:
		return ctx.ClientProxy(in)
	case packets.CopyDone, packets.ErrorResponse:
		ctx.EndCopyOut()
		return ctx.ClientProxy(in)
	default:
		log.Printf("unexpected packet %c\n", in.Type())
		panic("unexpected packet from server")
		return berr.ServerProtocolError
	}
}

func copyOutRecoverServer0(ctx *bctx.Context) {
	in, err2 := serverRead(ctx)
	if err2 != nil {
		ctx.EndCopyOut()
		return
	}
	switch in.Type() {
	case packets.CopyData:
		// continue
	case packets.CopyDone, packets.ErrorResponse:
		ctx.EndCopyOut()
		return
	default:
		panic("unexpected packet from server")
	}
}

func copyOutRecoverServer(ctx *bctx.Context, _ berr.Error) {
	// read until server is done with its copy
	for ctx.InCopyOut() {
		copyOutRecoverServer0(ctx)
	}
}

func copyOutRecoverClient(ctx *bctx.Context, err berr.Error) {
	// send error to client, will stop client copy
	out := ctx.ClientWrite()
	packets.WriteErrorResponse(out, err.PError())
	_ = ctx.ClientSend(out)

	ctx.EndCopyOut()
}

func copyOutRecover(ctx *bctx.Context, err berr.Error) {
	copyOutRecoverServer(ctx, err)
	copyOutRecoverClient(ctx, err)
}

func copyOut(ctx *bctx.Context) {
	ctx.BeginCopyOut()

	for ctx.InCopyOut() {
		err := copyOut0(ctx)
		if err != nil {
			copyOutRecover(ctx, err)
			return
		}
	}
}

func query0(ctx *bctx.Context) berr.Error {
	in, err := serverRead(ctx)
	if err != nil {
		return err
	}

	switch in.Type() {
	case packets.CommandComplete,
		packets.RowDescription,
		packets.DataRow,
		packets.EmptyQueryResponse,
		packets.ErrorResponse:
		return ctx.ClientProxy(in)
	case packets.CopyInResponse:
		err = ctx.ClientProxy(in)
		if err != nil {
			return err
		}
		copyIn(ctx)
		return nil
	case packets.CopyOutResponse:
		err = ctx.ClientProxy(in)
		if err != nil {
			return err
		}
		copyOut(ctx)
		return nil
	case packets.ReadyForQuery:
		ctx.EndQuery()
		err = ctx.ClientProxy(in)
		if err != nil {
			return err
		}
		return readyForQuery(ctx, in)
	default:
		log.Printf("unexpected packet %c\n", in.Type())
		panic("unexpected packet from server")
		return berr.ServerProtocolError
	}
}

func queryRecoverServer0(ctx *bctx.Context, err berr.Error) {
	in, err2 := serverRead(ctx)
	if err2 != nil {
		ctx.EndQuery()
		return
	}
	switch in.Type() {
	case packets.CommandComplete,
		packets.RowDescription,
		packets.DataRow,
		packets.EmptyQueryResponse,
		packets.ErrorResponse:
		// continue
	case packets.CopyInResponse:
		ctx.BeginCopyIn()
		copyInRecoverServer(ctx, err)
	case packets.CopyOutResponse:
		ctx.BeginCopyOut()
		copyOutRecoverServer(ctx, err)
	case packets.ReadyForQuery:
		ctx.EndQuery()
		readyForQuery(ctx, in)
	default:
		panic("unexpected packet from server")
	}
}

// serverTransactionFail ensures the server is in a failed txn block
func serverTransactionFail(ctx *bctx.Context, err berr.Error) {
	// we need to change this to a failed transaction block, write a simple query that will fail
	out := ctx.ServerWrite()
	out.Type(packets.Query)
	out.String("RAISE;")
	_ = ctx.ServerSend(out)
	ctx.BeginQuery()
	for ctx.InQuery() {
		queryRecoverServer0(ctx, err)
	}
}

func queryRecoverServer(ctx *bctx.Context, err berr.Error) {
	for ctx.InQuery() {
		queryRecoverServer0(ctx, err)
	}
	if ctx.InTransaction() {
		serverTransactionFail(ctx, err)
	}
}

func queryRecoverClient(ctx *bctx.Context, err berr.Error) {
	// send error to client followed by ready for query
	out := ctx.ClientWrite()
	packets.WriteErrorResponse(out, err.PError())
	_ = ctx.ClientSend(out)
	out = ctx.ClientWrite()
	if ctx.InTransaction() {
		packets.WriteReadyForQuery(out, 'E')
	} else {
		packets.WriteReadyForQuery(out, 'I')
	}
	_ = ctx.ClientSend(out)

	ctx.EndQuery()
}

func queryRecover(ctx *bctx.Context, err berr.Error) {
	queryRecoverServer(ctx, err)
	queryRecoverClient(ctx, err)
}

func query(ctx *bctx.Context) {
	ctx.BeginQuery()

	for ctx.InQuery() {
		err := query0(ctx)
		if err != nil {
			queryRecover(ctx, err)
			return
		}
	}

	return
}

func functionCall0(ctx *bctx.Context) berr.Error {
	in, err := serverRead(ctx)
	if err != nil {
		return err
	}

	switch in.Type() {
	case packets.ErrorResponse, packets.FunctionCallResponse:
		return ctx.ClientProxy(in)
	case packets.ReadyForQuery:
		ctx.EndFunctionCall()
		err = ctx.ClientProxy(in)
		if err != nil {
			return err
		}
		return readyForQuery(ctx, in)
	default:
		log.Printf("unexpected packet %c\n", in.Type())
		panic("unexpected packet from server")
		return berr.ServerProtocolError
	}
}

func functionCallRecoverServer0(ctx *bctx.Context) {
	in, err2 := serverRead(ctx)
	if err2 != nil {
		ctx.EndFunctionCall()
		return
	}
	switch in.Type() {
	case packets.ErrorResponse, packets.FunctionCallResponse:
		// continue
	case packets.ReadyForQuery:
		ctx.EndFunctionCall()
		readyForQuery(ctx, in)
	default:
		panic("unexpected packet from server")
	}
}

func functionCallRecoverServer(ctx *bctx.Context, err berr.Error) {
	for ctx.InFunctionCall() {
		functionCallRecoverServer0(ctx)
	}
	if ctx.InTransaction() {
		serverTransactionFail(ctx, err)
	}
}

func functionCallRecoverClient(ctx *bctx.Context, err berr.Error) {
	// send error to client followed by ready for query, will stop client function call
	out := ctx.ClientWrite()
	packets.WriteErrorResponse(out, err.PError())
	_ = ctx.ClientSend(out)
	out = ctx.ClientWrite()
	if ctx.InTransaction() {
		packets.WriteReadyForQuery(out, 'E')
	} else {
		packets.WriteReadyForQuery(out, 'I')
	}
	_ = ctx.ClientSend(out)

	ctx.EndFunctionCall()
}

func functionCallRecover(ctx *bctx.Context, err berr.Error) {
	functionCallRecoverServer(ctx, err)
	functionCallRecoverClient(ctx, err)
}

func functionCall(ctx *bctx.Context) {
	ctx.BeginFunctionCall()

	for ctx.InFunctionCall() {
		err := functionCall0(ctx)
		if err != nil {
			functionCallRecover(ctx, err)
			return
		}
	}
}

func sync0(ctx *bctx.Context) berr.Error {
	in, err := serverRead(ctx)
	if err != nil {
		return err
	}

	switch in.Type() {
	case packets.ParseComplete,
		packets.BindComplete,
		packets.ErrorResponse,
		packets.RowDescription,
		packets.NoData,
		packets.ParameterDescription,

		packets.CommandComplete,
		packets.DataRow,
		packets.EmptyQueryResponse,
		packets.PortalSuspended:
		return ctx.ClientProxy(in)
	case packets.ReadyForQuery:
		ctx.EndSync()
		err = ctx.ClientProxy(in)
		if err != nil {
			return err
		}
		return readyForQuery(ctx, in)
	default:
		log.Printf("unexpected packet %c\n", in.Type())
		panic("unexpected packet from server")
		return berr.ServerProtocolError
	}
}

func syncRecoverServer0(ctx *bctx.Context, _ berr.Error) {
	in, err2 := serverRead(ctx)
	if err2 != nil {
		ctx.EndSync()
		return
	}

	switch in.Type() {
	case packets.ParseComplete,
		packets.BindComplete,
		packets.ErrorResponse,
		packets.RowDescription,
		packets.NoData,
		packets.ParameterDescription,

		packets.CommandComplete,
		packets.DataRow,
		packets.EmptyQueryResponse,
		packets.PortalSuspended:
		// continue
	case packets.ReadyForQuery:
		ctx.EndSync()
		readyForQuery(ctx, in)
	default:
		panic("unexpected packet from server")
	}
}

func syncRecoverServer(ctx *bctx.Context, err berr.Error) {
	for ctx.InSync() {
		syncRecoverServer0(ctx, err)
	}
	if ctx.InTransaction() {
		serverTransactionFail(ctx, err)
	}
}

func syncRecoverClient(ctx *bctx.Context, err berr.Error) {
	// send error to client followed by ready for query
	out := ctx.ClientWrite()
	packets.WriteErrorResponse(out, err.PError())
	_ = ctx.ClientSend(out)
	out = ctx.ClientWrite()
	if ctx.InTransaction() {
		packets.WriteReadyForQuery(out, 'E')
	} else {
		packets.WriteReadyForQuery(out, 'I')
	}
	_ = ctx.ClientSend(out)

	ctx.EndSync()
}

func syncRecover(ctx *bctx.Context, err berr.Error) {
	syncRecoverServer(ctx, err)
	syncRecoverClient(ctx, err)
}

func sync(ctx *bctx.Context) {
	ctx.BeginSync()

	for ctx.InSync() {
		err := sync0(ctx)
		if err != nil {
			syncRecover(ctx, err)
			return
		}
	}
}

func transaction0(ctx *bctx.Context) berr.Error {
	in, err := ctx.ClientRead()
	if err != nil {
		return err
	}

	switch in.Type() {
	case packets.Query:
		err = ctx.ServerProxy(in)
		if err != nil {
			return err
		}
		query(ctx)
		return nil
	case packets.FunctionCall:
		err = ctx.ServerProxy(in)
		if err != nil {
			return err
		}
		functionCall(ctx)
		return nil
	case packets.Sync:
		if !ctx.InEQP() {
			ctx.BeginEQP()
		}
		err = ctx.ServerProxy(in)
		if err != nil {
			return err
		}
		sync(ctx)
		return nil
	case packets.Parse, packets.Bind, packets.Close, packets.Describe, packets.Execute, packets.Flush:
		if !ctx.InEQP() {
			ctx.BeginEQP()
		}
		return ctx.ServerProxy(in)
	default:
		return berr.ClientProtocolError
	}
}

func transactionRecoverServer(ctx *bctx.Context, err berr.Error) {
	if ctx.InEQP() {
		// send sync and ignore until ready for query
		out := ctx.ServerWrite()
		out.Type(packets.Sync)
		err2 := ctx.ServerSend(out)
		if err2 != nil {
			return
		}
		ctx.BeginSync()
		syncRecoverServer(ctx, err)
	}
	if ctx.InTransaction() {
		// send END to break out of transaction and wait for ready for query
		out := ctx.ServerWrite()
		out.Type(packets.Query)
		out.String("END;")
		err2 := ctx.ServerSend(out)
		if err2 != nil {
			return
		}
		ctx.BeginQuery()
		queryRecoverServer(ctx, err)
	}
}

func transactionRecoverClient(ctx *bctx.Context, err berr.Error) {
	out := ctx.ClientWrite()
	packets.WriteErrorResponse(out, err.PError())
	_ = ctx.ClientSend(out)
	out = ctx.ClientWrite()
	packets.WriteReadyForQuery(out, 'I')
	_ = ctx.ClientSend(out)

	ctx.EndEQP()
	ctx.EndTransaction()
}

func transactionRecover(ctx *bctx.Context, err berr.Error) {
	transactionRecoverServer(ctx, err)
	transactionRecoverClient(ctx, err)
}

func transaction(ctx *bctx.Context) {
	ctx.BeginTransaction()

	for ctx.InTransaction() {
		err := transaction0(ctx)
		if err != nil {
			transactionRecover(ctx, err)
			return
		}
	}
}

func Bounce(client, server zap.ReadWriter) {
	ctx := bctx.MakeContext(client, server, 0) // TODO(garet) make this configurable
	defer ctx.Done()
	transaction(&ctx)
	ctx.AssertDone()
}
