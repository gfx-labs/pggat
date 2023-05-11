package bouncers

import (
	"errors"
	"log"
	"runtime/debug"

	"pggat2/lib/perror"
	"pggat2/lib/pnet"
	"pggat2/lib/pnet/packet"
	packets "pggat2/lib/pnet/packet/packets/v3.0"
)

type Status int

const (
	Fail Status = iota
	Ok
)

func clientFail(client pnet.ReadWriter, err perror.Error) {
	// DEBUG(garet)
	log.Println("client fail", err)
	debug.PrintStack()

	out := client.Write()
	packets.WriteErrorResponse(out, err)
	_ = client.Send(out.Finish())
}

func serverFail(server pnet.ReadWriter, err error) {
	panic(err)
}

func copyIn0(client, server pnet.ReadWriter) (done bool, status Status) {
	in, err := client.Read()
	if err != nil {
		clientFail(client, perror.Wrap(err))
		return false, Fail
	}

	switch in.Type() {
	case packet.CopyData:
		err = pnet.ProxyPacket(server, in)
		if err != nil {
			serverFail(server, err)
			return false, Fail
		}
		return true, Ok
	case packet.CopyDone, packet.CopyFail:
		err = pnet.ProxyPacket(server, in)
		if err != nil {
			serverFail(server, err)
			return false, Fail
		}
		return true, Ok
	default:
		clientFail(client, pnet.ErrProtocolError)
		return false, Fail
	}
}

func copyIn(client, server pnet.ReadWriter, in packet.In) (status Status) {
	// send in (copyInResponse) to client
	err := pnet.ProxyPacket(client, in)
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

func copyOut0(client, server pnet.ReadWriter) (done bool, status Status) {
	in, err := server.Read()
	if err != nil {
		serverFail(server, err)
		return false, Fail
	}

	switch in.Type() {
	case packet.CopyData:
		err = pnet.ProxyPacket(client, in)
		if err != nil {
			clientFail(client, perror.Wrap(err))
			return false, Fail
		}
		return false, Ok
	case packet.CopyDone, packet.ErrorResponse:
		err = pnet.ProxyPacket(client, in)
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

func copyOut(client, server pnet.ReadWriter, in packet.In) (status Status) {
	// send in (copyOutResponse) to client
	err := pnet.ProxyPacket(client, in)
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

func query0(client, server pnet.ReadWriter) (done bool, status Status) {
	in, err := server.Read()
	if err != nil {
		serverFail(server, err)
		return false, Fail
	}

	switch in.Type() {
	case packet.CommandComplete,
		packet.RowDescription,
		packet.DataRow,
		packet.EmptyQueryResponse,
		packet.ErrorResponse,
		packet.NoticeResponse,
		packet.ParameterStatus:
		err = pnet.ProxyPacket(client, in)
		if err != nil {
			clientFail(client, perror.Wrap(err))
			return false, Fail
		}
		return false, Ok
	case packet.CopyInResponse:
		status = copyIn(client, server, in)
		if status != Ok {
			return false, status
		}
		return false, Ok
	case packet.CopyOutResponse:
		status = copyOut(client, server, in)
		if status != Ok {
			return false, status
		}
		return false, Ok
	case packet.ReadyForQuery:
		err = pnet.ProxyPacket(client, in)
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

func query(client, server pnet.ReadWriter, in packet.In) (status Status) {
	// send in (initial query) to server
	err := pnet.ProxyPacket(server, in)
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

func functionCall0(client, server pnet.ReadWriter) (done bool, status Status) {
	in, err := server.Read()
	if err != nil {
		serverFail(server, err)
		return false, Fail
	}

	switch in.Type() {
	case packet.ErrorResponse, packet.FunctionCallResponse, packet.NoticeResponse:
		err = pnet.ProxyPacket(client, in)
		if err != nil {
			clientFail(client, perror.Wrap(err))
			return false, Fail
		}
		return false, Ok
	case packet.ReadyForQuery:
		err = pnet.ProxyPacket(client, in)
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

func functionCall(client, server pnet.ReadWriter, in packet.In) (status Status) {
	// send in (FunctionCall) to server
	err := pnet.ProxyPacket(server, in)
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

func sync0(client, server pnet.ReadWriter) (done bool, status Status) {
	in, err := server.Read()
	if err != nil {
		serverFail(server, err)
		return false, Fail
	}

	switch in.Type() {
	case packet.ParseComplete,
		packet.BindComplete,
		packet.ErrorResponse,
		packet.RowDescription,
		packet.NoData,
		packet.ParameterDescription,

		packet.CommandComplete,
		packet.DataRow,
		packet.EmptyQueryResponse,
		packet.NoticeResponse,
		packet.ParameterStatus,
		packet.PortalSuspended:
		err = pnet.ProxyPacket(client, in)
		if err != nil {
			clientFail(client, perror.Wrap(err))
			return false, Fail
		}
		return false, Ok
	case packet.ReadyForQuery:
		err = pnet.ProxyPacket(client, in)
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

func sync(client, server pnet.ReadWriter, in packet.In) (status Status) {
	// send initial (sync) to server
	err := pnet.ProxyPacket(server, in)
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

func eqp(client, server pnet.ReadWriter, in packet.In) (status Status) {
	for {
		switch in.Type() {
		case packet.Sync:
			return sync(client, server, in)
		case packet.Parse, packet.Bind, packet.Describe, packet.Execute:
			err := pnet.ProxyPacket(server, in)
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

func Bounce(client, server pnet.ReadWriter) {
	in, err := client.Read()
	if err != nil {
		clientFail(client, perror.Wrap(err))
		return
	}

	switch in.Type() {
	case packet.Query:
		query(client, server, in)
	case packet.FunctionCall:
		functionCall(client, server, in)
	case packet.Sync, packet.Parse, packet.Bind, packet.Describe, packet.Execute:
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
