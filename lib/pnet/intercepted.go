package pnet

import (
	"pggat2/lib/middleware"
	"pggat2/lib/pnet/packet"
)

type Intercepted struct {
	ReadWriter
	Middlewares []middleware.Middleware
}

func (T Intercepted) interceptRead(in packet.In) (forward bool, err error) {
	for _, mw := range T.Middlewares {
		forward, err = mw.Read(in)
		if err != nil || !forward {
			return
		}
	}
	return true, nil
}

func (T Intercepted) Read() (packet.In, error) {
	for {
		in, err := T.ReadWriter.Read()
		if err != nil {
			return packet.In{}, err
		}
		var forward bool
		forward, err = T.interceptRead(in)
		if err != nil {
			return packet.In{}, err
		}
		if forward {
			return in, nil
		}
	}
}

func (T Intercepted) ReadUntyped() (packet.In, error) {
	for {
		in, err := T.ReadWriter.ReadUntyped()
		if err != nil {
			return packet.In{}, err
		}
		var forward bool
		forward, err = T.interceptRead(in)
		if err != nil {
			return packet.In{}, err
		}
		if forward {
			return in, nil
		}
	}
}

func (T Intercepted) Send(typ packet.Type, payload []byte) error {
	inBuf := packet.MakeInBuf(typ, payload)
	in := packet.MakeIn(&inBuf)
	for _, mw := range T.Middlewares {
		forward, err := mw.Write(in)
		if err != nil {
			return err
		}
		if !forward {
			return nil
		}
	}
	return T.ReadWriter.Send(typ, payload)
}

var _ ReadWriter = Intercepted{}
