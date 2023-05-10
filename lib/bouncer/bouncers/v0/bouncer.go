package bouncers

import (
	"errors"

	"pggat2/lib/perror"
	"pggat2/lib/pnet"
	"pggat2/lib/pnet/packet"
)

type Status int

const (
	Fail Status = iota
	Ok
)

func clientFail(client pnet.ReadWriter, err perror.Error) {
	panic(err)
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
	default:
		clientFail(client, perror.New(
			perror.ERROR,
			perror.FeatureNotSupported,
			"unsupported operation",
		))
		return
	}
}
