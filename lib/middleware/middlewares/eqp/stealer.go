package eqp

import (
	"errors"

	"pggat2/lib/pnet"
	"pggat2/lib/pnet/packet"
	packets "pggat2/lib/pnet/packet/packets/v3.0"
)

// Stealer wraps a Consumer and duplicates the underlying Consumer's portals and prepared statements on use.
type Stealer struct {
	creator  *Creator
	consumer *Consumer

	// need a second buf because we cannot use the underlying Consumer's buf (or it would overwrite the outgoing packet)
	buf packet.OutBuf
}

func NewStealer(consumer *Consumer, creator *Creator) *Stealer {
	return &Stealer{
		creator:  creator,
		consumer: consumer,
	}
}

func (T *Stealer) Read() (packet.In, error) {
	for {
		in, err := T.consumer.Read()
		if err != nil {
			return packet.In{}, err
		}
		switch in.Type() {
		case packet.ParseComplete:
			// previous parse was successful
		case packet.BindComplete:
			// previous bind was successful
		default:
			// forward
			return in, nil
		}
	}
}

func (T *Stealer) ReadUntyped() (packet.In, error) {
	return T.consumer.ReadUntyped()
}

func (T *Stealer) Write() packet.Out {
	return T.consumer.Write()
}

func (T *Stealer) WriteByte(b byte) error {
	return T.consumer.WriteByte(b)
}

func (T *Stealer) bindPreparedStatement(target string, preparedStatement PreparedStatement) error {
	T.buf.Reset()
	out := packet.MakeOut(&T.buf)
	packets.WriteParse(out, target, preparedStatement.Query, preparedStatement.ParameterDataTypes)
	return T.consumer.Send(out.Finish())
}

func (T *Stealer) bindPortal(target string, portal Portal) error {
	T.buf.Reset()
	out := packet.MakeOut(&T.buf)
	packets.WriteBind(out, target, portal.Source, portal.ParameterFormatCodes, portal.ParameterValues, portal.ResultFormatCodes)
	return T.consumer.Send(out.Finish())
}

func (T *Stealer) closePreparedStatement(target string) error {
	if _, ok := T.consumer.preparedStatements[target]; !ok {
		// doesn't exist
		return nil
	}
	T.buf.Reset()
	out := packet.MakeOut(&T.buf)
	packets.WriteClose(out, 'S', target)
	return T.consumer.Send(out.Finish())
}

func (T *Stealer) closePortal(target string) error {
	if _, ok := T.consumer.portals[target]; !ok {
		// doesn't exist
		return nil
	}
	T.buf.Reset()
	out := packet.MakeOut(&T.buf)
	packets.WriteClose(out, 'P', target)
	return T.consumer.Send(out.Finish())
}

func (T *Stealer) syncPreparedStatement(target string) error {
	creatorStatement, ok := T.creator.preparedStatements[target]
	if !ok {
		return T.closePreparedStatement(target)
	}
	consumerStatement, prev := T.consumer.preparedStatements[target]
	if creatorStatement.Equals(consumerStatement) {
		return nil
	}
	// clean up prev
	if prev {
		err := T.closePreparedStatement(target)
		if err != nil {
			return err
		}
	}
	// send prepared statement
	return T.bindPreparedStatement(target, creatorStatement)
}

func (T *Stealer) syncPortal(target string) error {
	creatorPortal, ok := T.creator.portals[target]
	if !ok {
		return T.closePortal(target)
	}
	consumerPortal, prev := T.consumer.portals[target]
	if creatorPortal.Equals(consumerPortal) {
		return nil
	}
	// clean up prev
	if prev {
		err := T.closePortal(target)
		if err != nil {
			return err
		}
	}
	// send portal
	return T.bindPortal(target, creatorPortal)
}

func (T *Stealer) Send(typ packet.Type, bytes []byte) error {
	// check if we are using a prepared statement or portal that we need to steal
	buf := packet.MakeInBuf(typ, bytes)
	in := packet.MakeIn(&buf)
	switch typ {
	case packet.Describe:
		which, target, ok := packets.ReadDescribe(in)
		if !ok {
			return errors.New("bad packet format")
		}
		switch which {
		case 'S':
			err := T.syncPreparedStatement(target)
			if err != nil {
				return err
			}
		case 'P':
			err := T.syncPortal(target)
			if err != nil {
				return err
			}
		default:
			return errors.New("unknown describe target")
		}
	case packet.Execute:
		target, _, ok := packets.ReadExecute(in)
		if !ok {
			return errors.New("bad packet format")
		}
		err := T.syncPortal(target)
		if err != nil {
			return err
		}
	}

	return T.consumer.Send(typ, bytes)
}

var _ pnet.ReadWriter = (*Stealer)(nil)
