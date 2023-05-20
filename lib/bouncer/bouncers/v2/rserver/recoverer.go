package rserver

import (
	"pggat2/lib/bouncer/bouncers/v2/bctx"
)

func Recover(ctx *bctx.Context) error {
	if inCopyOut {
		// TODO(garet) wait for CopyDone or ErrorResponse
	}
	if inCopyIn {
		// TODO(garet) send CopyFail
	}
	if inQuery {
		// TODO(garet) wait for ready for query, waiting for copyOut if it happens, failing copyIn if it happens
	}
	if inFunctionCall {
		// TODO(garet) wait for ready for query
	}
	if inSync {
		// TODO(garet) wait for ready for query
	}
	if inEQP {
		// TODO(garet) send sync and wait for ready for query
	}
	if ctx.TxState != 'I' {
		// TODO(garet) send Query('ABORT;') and wait for ReadyForQuery
	}
	return nil
}
