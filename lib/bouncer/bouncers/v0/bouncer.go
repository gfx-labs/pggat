package bouncers

import (
	"errors"
	"log"
	"runtime/debug"

	"pggat2/lib/perror"
	"pggat2/lib/zap"
	"pggat2/lib/zap/packets/v3.0"
)

type Status int

const (
	Fail Status = iota
	Ok
)

func clientFail(client zap.ReadWriter, err perror.Error) {
	// DEBUG(garet)
	log.Println("client fail", err)
	debug.PrintStack()

	out := client.Write()
	packets.WriteErrorResponse(out, err)
	_ = client.Send(out)
}

func serverFail(server zap.ReadWriter, err error) {
	panic(err)
}

func copyIn0(client, server zap.ReadWriter) (done bool, status Status) {
	in, err := client.Read()
	if err != nil {
		clientFail(client, perror.Wrap(err))
		return false, Fail
	}

	switch in.Type() {
	case packets.CopyData:
		err = server.Send(zap.InToOut(in))
		if err != nil {
			serverFail(server, err)
			return false, Fail
		}
		return true, Ok
	case packets.CopyDone, packets.CopyFail:
		err = server.Send(zap.InToOut(in))
		if err != nil {
			serverFail(server, err)
			return false, Fail
		}
		return true, Ok
	default:
		clientFail(client, packets.ErrUnexpectedPacket)
		return false, Fail
	}
}

func copyIn(client, server zap.ReadWriter, in zap.In) (status Status) {
	// send in (copyInResponse) to client
	err := client.Send(zap.InToOut(in))
	if err != nil {
		clientFail(client, perror.Wrap(err))
		return Fail
	}

	// copy in from client
	for {
		var done bool
		done, status = copyIn0(client, server)
		if status != Ok {
			return
		}
		if done {
			break
		}
	}
	return Ok
}

func copyOut0(client, server zap.ReadWriter) (done bool, status Status) {
	in, err := server.Read()
	if err != nil {
		serverFail(server, err)
		return false, Fail
	}

	switch in.Type() {
	case packets.CopyData:
		err = client.Send(zap.InToOut(in))
		if err != nil {
			clientFail(client, perror.Wrap(err))
			return false, Fail
		}
		return false, Ok
	case packets.CopyDone, packets.ErrorResponse:
		err = client.Send(zap.InToOut(in))
		if err != nil {
			clientFail(client, perror.Wrap(err))
			return false, Fail
		}
		return true, Ok
	default:
		serverFail(server, errors.New("protocol error"))
		return false, Fail
	}
}

func copyOut(client, server zap.ReadWriter, in zap.In) (status Status) {
	// send in (copyOutResponse) to client
	err := client.Send(zap.InToOut(in))
	if err != nil {
		clientFail(client, perror.Wrap(err))
		return Fail
	}

	// copy out from server
	for {
		var done bool
		done, status = copyOut0(client, server)
		if status != Ok {
			return
		}
		if done {
			break
		}
	}
	return Ok
}

func query0(client, server zap.ReadWriter) (done bool, status Status) {
	in, err := server.Read()
	if err != nil {
		serverFail(server, err)
		return false, Fail
	}

	switch in.Type() {
	case packets.CommandComplete,
		packets.RowDescription,
		packets.DataRow,
		packets.EmptyQueryResponse,
		packets.ErrorResponse,
		packets.NoticeResponse,
		packets.ParameterStatus:
		err = client.Send(zap.InToOut(in))
		if err != nil {
			clientFail(client, perror.Wrap(err))
			return false, Fail
		}
		return false, Ok
	case packets.CopyInResponse:
		status = copyIn(client, server, in)
		if status != Ok {
			return false, status
		}
		return false, Ok
	case packets.CopyOutResponse:
		status = copyOut(client, server, in)
		if status != Ok {
			return false, status
		}
		return false, Ok
	case packets.ReadyForQuery:
		err = client.Send(zap.InToOut(in))
		if err != nil {
			clientFail(client, perror.Wrap(err))
			return false, Fail
		}
		return true, Ok
	default:
		serverFail(server, errors.New("protocol error"))
		return false, Fail
	}
}

func query(client, server zap.ReadWriter, in zap.In) (status Status) {
	// send in (initial query) to server
	err := server.Send(zap.InToOut(in))
	if err != nil {
		serverFail(server, err)
		return Fail
	}

	for {
		var done bool
		done, status = query0(client, server)
		if status != Ok {
			return
		}
		if done {
			break
		}
	}
	return Ok
}

func functionCall0(client, server zap.ReadWriter) (done bool, status Status) {
	in, err := server.Read()
	if err != nil {
		serverFail(server, err)
		return false, Fail
	}

	switch in.Type() {
	case packets.ErrorResponse, packets.FunctionCallResponse, packets.NoticeResponse:
		err = client.Send(zap.InToOut(in))
		if err != nil {
			clientFail(client, perror.Wrap(err))
			return false, Fail
		}
		return false, Ok
	case packets.ReadyForQuery:
		err = client.Send(zap.InToOut(in))
		if err != nil {
			clientFail(client, perror.Wrap(err))
			return false, Fail
		}
		return true, Ok
	default:
		serverFail(server, errors.New("protocol error"))
		return false, Fail
	}
}

func functionCall(client, server zap.ReadWriter, in zap.In) (status Status) {
	// send in (FunctionCall) to server
	err := server.Send(zap.InToOut(in))
	if err != nil {
		serverFail(server, err)
		return Fail
	}

	for {
		var done bool
		done, status = functionCall0(client, server)
		if status != Ok {
			return
		}
		if done {
			break
		}
	}
	return Ok
}

func sync0(client, server zap.ReadWriter) (done bool, status Status) {
	in, err := server.Read()
	if err != nil {
		serverFail(server, err)
		return false, Fail
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
		packets.NoticeResponse,
		packets.ParameterStatus,
		packets.PortalSuspended:
		err = client.Send(zap.InToOut(in))
		if err != nil {
			clientFail(client, perror.Wrap(err))
			return false, Fail
		}
		return false, Ok
	case packets.ReadyForQuery:
		err = client.Send(zap.InToOut(in))
		if err != nil {
			clientFail(client, perror.Wrap(err))
			return false, Fail
		}
		return true, Ok
	default:
		log.Printf("operation %c", in.Type())
		serverFail(server, errors.New("protocol error"))
		return false, Fail
	}
}

func sync(client, server zap.ReadWriter, in zap.In) (status Status) {
	// send initial (sync) to server
	err := server.Send(zap.InToOut(in))
	if err != nil {
		serverFail(server, err)
		return Fail
	}

	// relay everything until ready for query
	for {
		var done bool
		done, status = sync0(client, server)
		if status != Ok {
			return
		}
		if done {
			break
		}
	}
	return Ok
}

func eqp(client, server zap.ReadWriter, in zap.In) (status Status) {
	for {
		switch in.Type() {
		case packets.Sync:
			return sync(client, server, in)
		case packets.Parse, packets.Bind, packets.Describe, packets.Execute:
			err := server.Send(zap.InToOut(in))
			if err != nil {
				serverFail(server, err)
				return Fail
			}
		default:
			log.Printf("operation %c", in.Type())
			clientFail(client, perror.New(
				perror.ERROR,
				perror.FeatureNotSupported,
				"unsupported operation",
			))
			return Fail
		}
		var err error
		in, err = client.Read()
		if err != nil {
			clientFail(client, perror.Wrap(err))
			return Fail
		}
	}
}

func Bounce(client, server zap.ReadWriter) {
	in, err := client.Read()
	if err != nil {
		clientFail(client, perror.Wrap(err))
		return
	}

	switch in.Type() {
	case packets.Query:
		query(client, server, in)
	case packets.FunctionCall:
		functionCall(client, server, in)
	case packets.Sync, packets.Parse, packets.Bind, packets.Describe, packets.Execute:
		eqp(client, server, in)
	default:
		log.Printf("operation %c", in.Type())
		clientFail(client, perror.New(
			perror.ERROR,
			perror.FeatureNotSupported,
			"unsupported operation",
		))
		return
	}
}
