package rclient

import "pggat2/lib/bouncer/bouncers/v2/bctx"

func Recover(ctx *bctx.Context) error {
	// TODO(garet) I actually don't know if I have to obey the client's expectations when it comes to Txs
	// We are just going to break out without letting the client tell us to. This might lead to app crashes,
	// but if a database is crashing or behaving badly, you can't really expect any better behavior
	if inCopyIn {
		// TODO(garet) wait until client sends CopyDone or CopyFail
	}
	if inCopyOut {
		// TODO(garet) send ErrorResponse("server failed")
	}
	if inQuery {
		// TODO(garet) send ErrorResponse("server failed")
		// TODO(garet) send ReadyForQuery('I')
	}
	if inFunctionCall {
		// TODO(garet) send ErrorResponse("server failed")
		// TODO(garet) send ReadyForQuery('I')
	}
	if inSync {
		// TODO(garet) send ErrorResponse("server failed")
		// TODO(garet) send ReadyForQuery('I')
	}
	if inEQP {
		// TODO(garet) discard until Sync and then send error and ReadyForQuery('I')
	}
	if ctx.TxState != 'I' {
		// TODO(garet) wait for next packet and then handle it, sending error and ReadyForQuery('I')
	}
	return nil
}
