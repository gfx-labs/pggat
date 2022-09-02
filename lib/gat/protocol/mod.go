package protocol

import "io"

type Packet interface {
	Read(reader io.Reader) error
	Write(writer io.Writer) (int, error)
}

// Read switches on the identifier and returns the matching packet
// DO NOT call this function if the packet in queue does not have an identifier
func Read(reader io.Reader) (packet Packet, err error) {
	var identifier byte
	identifier, err = ReadByte(reader)
	if err != nil {
		return
	}

	switch identifier {
	case byte('R'):
		packet = new(Authentication)
	case byte('K'):
		packet = new(BackendKeyData)
	case byte('B'):
		packet = new(Bind)
	case byte('2'):
		packet = new(BindComplete)
	case byte('F'):
		packet = new(CancelRequest)
	case byte('C'):
		packet = new(Close)
	case byte('3'):
		packet = new(CloseComplete)
	case byte('C'):
		packet = new(CommandComplete)
	case byte('W'):
		packet = new(CopyBothResponse)
	case byte('d'):
		packet = new(CopyData)
	case byte('c'):
		packet = new(CopyDone)
	case byte('f'):
		packet = new(CopyFail)
	case byte('G'):
		packet = new(CopyInResponse)
	case byte('H'):
		packet = new(CopyOutResponse)
	case byte('D'):
		packet = new(DataRow)
	case byte('D'):
		packet = new(Describe)
	case byte('I'):
		packet = new(EmptyQueryResponse)
	case byte('E'):
		packet = new(ErrorResponse)
	case byte('E'):
		packet = new(Execute)
	case byte('H'):
		packet = new(Flush)
	case byte('F'):
		packet = new(FunctionCall)
	case byte('V'):
		packet = new(FunctionCallResponse)
	case byte('p'):
		packet = new(GSSResponse)
	case byte('v'):
		packet = new(NegotiateProtocolVersion)
	case byte('n'):
		packet = new(NoData)
	case byte('N'):
		packet = new(NoticeResponse)
	case byte('A'):
		packet = new(NotificationResponse)
	case byte('t'):
		packet = new(ParameterDescription)
	case byte('S'):
		packet = new(ParameterStatus)
	case byte('P'):
		packet = new(Parse)
	case byte('1'):
		packet = new(ParseComplete)
	case byte('p'):
		packet = new(PasswordMessage)
	case byte('s'):
		packet = new(PortalSuspended)
	case byte('Q'):
		packet = new(Query)
	case byte('Z'):
		packet = new(ReadForQuery)
	case byte('T'):
		packet = new(RowDescription)
	case byte('p'):
		packet = new(SASLInitialResponse)
	case byte('p'):
		packet = new(SASLResponse)
	case byte('S'):
		packet = new(Sync)
	case byte('X'):
		packet = new(Terminate)
	}

	err = packet.Read(reader)
	if err != nil {
		return
	}

	return
}
