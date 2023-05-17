package ps

import (
	"errors"

	"pggat2/lib/middleware"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

type Client struct {
	parameters map[string]string
	buf        zap.Buf

	middleware.Nil
}

func MakeClient() Client {
	return Client{
		parameters: make(map[string]string),
	}
}

func (T *Client) Done() {
	T.buf.Done()
}

func (T *Client) updateParameter(w zap.Writer, name, value string) error {
	if T.parameters[name] == value {
		return nil
	}

	out := T.buf.Write()
	packets.WriteParameterStatus(out, name, value)
	err := w.Send(out)
	if err != nil {
		return err
	}

	T.parameters[name] = value

	return nil
}

func (T *Client) Sync(w zap.Writer, server *Server) error {
	// TODO(garet) i don't like this
	for name := range T.parameters {
		expected := server.parameters[name]
		err := T.updateParameter(w, name, expected)
		if err != nil {
			return err
		}
	}

	for name, expected := range server.parameters {
		err := T.updateParameter(w, name, expected)
		if err != nil {
			return err
		}
	}

	return nil
}

func (T *Client) Send(ctx middleware.Context, out zap.Out) error {
	in := zap.OutToIn(out)
	switch in.Type() {
	case packets.ParameterStatus:
		key, value, ok := packets.ReadParameterStatus(in)
		if !ok {
			return errors.New("bad packet format")
		}
		if T.parameters[key] == value {
			// already set
			ctx.Cancel()
			break
		}
		T.parameters[key] = value
	}
	return nil
}

var _ middleware.Middleware = (*Client)(nil)
