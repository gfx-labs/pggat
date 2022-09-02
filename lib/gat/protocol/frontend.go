package protocol

import "io"

// codegen: modify for debug only

type FieldsBindParameterValues struct {
	Value []byte
}

func (T *FieldsBindParameterValues) Read(payloadLength int, reader io.Reader) (err error) {
	var ValueLength int32
	ValueLength, err = ReadInt32(reader)
	if err != nil {
		return
	}
	T.Value = make([]byte, int(ValueLength))
	for i := 0; i < int(ValueLength); i++ {
		T.Value[i], err = ReadByte(reader)
		if err != nil {
			return
		}
	}
	return
}

type FieldsBind struct {
	Destination             string
	PreparedStatement       string
	FormatCodes             []int16
	ParameterValues         []FieldsBindParameterValues
	ResultColumnFormatCodes []int16
}

func (T *FieldsBind) Read(payloadLength int, reader io.Reader) (err error) {
	T.Destination, err = ReadString(reader)
	if err != nil {
		return
	}
	T.PreparedStatement, err = ReadString(reader)
	if err != nil {
		return
	}
	var FormatCodesLength int16
	FormatCodesLength, err = ReadInt16(reader)
	if err != nil {
		return
	}
	T.FormatCodes = make([]int16, int(FormatCodesLength))
	for i := 0; i < int(FormatCodesLength); i++ {
		T.FormatCodes[i], err = ReadInt16(reader)
		if err != nil {
			return
		}
	}
	var ParameterValuesLength int16
	ParameterValuesLength, err = ReadInt16(reader)
	if err != nil {
		return
	}
	T.ParameterValues = make([]FieldsBindParameterValues, int(ParameterValuesLength))
	for i := 0; i < int(ParameterValuesLength); i++ {
		err = T.ParameterValues[i].Read(payloadLength, reader)
		if err != nil {
			return
		}
	}
	var ResultColumnFormatCodesLength int16
	ResultColumnFormatCodesLength, err = ReadInt16(reader)
	if err != nil {
		return
	}
	T.ResultColumnFormatCodes = make([]int16, int(ResultColumnFormatCodesLength))
	for i := 0; i < int(ResultColumnFormatCodesLength); i++ {
		T.ResultColumnFormatCodes[i], err = ReadInt16(reader)
		if err != nil {
			return
		}
	}
	return
}

type Bind struct {
	fields FieldsBind
}

type FieldsCancelRequest struct {
	RequestCode int32
	ProcessID   int32
	SecretKey   int32
}

func (T *FieldsCancelRequest) Read(payloadLength int, reader io.Reader) (err error) {
	T.RequestCode, err = ReadInt32(reader)
	if err != nil {
		return
	}
	T.ProcessID, err = ReadInt32(reader)
	if err != nil {
		return
	}
	T.SecretKey, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return
}

type CancelRequest struct {
	fields FieldsCancelRequest
}

type FieldsClose struct {
	Which byte
	Name  string
}

func (T *FieldsClose) Read(payloadLength int, reader io.Reader) (err error) {
	T.Which, err = ReadByte(reader)
	if err != nil {
		return
	}
	T.Name, err = ReadString(reader)
	if err != nil {
		return
	}
	return
}

type Close struct {
	fields FieldsClose
}

type FieldsCopyFail struct {
	Cause string
}

func (T *FieldsCopyFail) Read(payloadLength int, reader io.Reader) (err error) {
	T.Cause, err = ReadString(reader)
	if err != nil {
		return
	}
	return
}

type CopyFail struct {
	fields FieldsCopyFail
}

type FieldsDescribe struct {
	Which byte
	Name  string
}

func (T *FieldsDescribe) Read(payloadLength int, reader io.Reader) (err error) {
	T.Which, err = ReadByte(reader)
	if err != nil {
		return
	}
	T.Name, err = ReadString(reader)
	if err != nil {
		return
	}
	return
}

type Describe struct {
	fields FieldsDescribe
}

type FieldsExecute struct {
	Name    string
	MaxRows int32
}

func (T *FieldsExecute) Read(payloadLength int, reader io.Reader) (err error) {
	T.Name, err = ReadString(reader)
	if err != nil {
		return
	}
	T.MaxRows, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return
}

type Execute struct {
	fields FieldsExecute
}

type FieldsFlush struct {
}

func (T *FieldsFlush) Read(payloadLength int, reader io.Reader) (err error) {
	return
}

type Flush struct {
	fields FieldsFlush
}

type FieldsFunctionCallArguments struct {
	Value []byte
}

func (T *FieldsFunctionCallArguments) Read(payloadLength int, reader io.Reader) (err error) {
	var ValueLength int32
	ValueLength, err = ReadInt32(reader)
	if err != nil {
		return
	}
	T.Value = make([]byte, int(ValueLength))
	for i := 0; i < int(ValueLength); i++ {
		T.Value[i], err = ReadByte(reader)
		if err != nil {
			return
		}
	}
	return
}

type FieldsFunctionCall struct {
	Function            int32
	ArgumentFormatCodes []int16
	Arguments           []FieldsFunctionCallArguments
	ResultFormatCode    int16
}

func (T *FieldsFunctionCall) Read(payloadLength int, reader io.Reader) (err error) {
	T.Function, err = ReadInt32(reader)
	if err != nil {
		return
	}
	var ArgumentFormatCodesLength int16
	ArgumentFormatCodesLength, err = ReadInt16(reader)
	if err != nil {
		return
	}
	T.ArgumentFormatCodes = make([]int16, int(ArgumentFormatCodesLength))
	for i := 0; i < int(ArgumentFormatCodesLength); i++ {
		T.ArgumentFormatCodes[i], err = ReadInt16(reader)
		if err != nil {
			return
		}
	}
	var ArgumentsLength int16
	ArgumentsLength, err = ReadInt16(reader)
	if err != nil {
		return
	}
	T.Arguments = make([]FieldsFunctionCallArguments, int(ArgumentsLength))
	for i := 0; i < int(ArgumentsLength); i++ {
		err = T.Arguments[i].Read(payloadLength, reader)
		if err != nil {
			return
		}
	}
	T.ResultFormatCode, err = ReadInt16(reader)
	if err != nil {
		return
	}
	return
}

type FunctionCall struct {
	fields FieldsFunctionCall
}

type FieldsGSSENCRequest struct {
	EncryptionRequestCode int32
}

func (T *FieldsGSSENCRequest) Read(payloadLength int, reader io.Reader) (err error) {
	T.EncryptionRequestCode, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return
}

type GSSENCRequest struct {
	fields FieldsGSSENCRequest
}

type FieldsGSSResponse struct {
	Data []byte
}

func (T *FieldsGSSResponse) Read(payloadLength int, reader io.Reader) (err error) {
	DataLength := payloadLength
	T.Data = make([]byte, int(DataLength))
	for i := 0; i < int(DataLength); i++ {
		T.Data[i], err = ReadByte(reader)
		if err != nil {
			return
		}
	}
	return
}

type GSSResponse struct {
	fields FieldsGSSResponse
}

type FieldsParse struct {
	PreparedStatement  string
	Query              string
	ParameterDataTypes []int32
}

func (T *FieldsParse) Read(payloadLength int, reader io.Reader) (err error) {
	T.PreparedStatement, err = ReadString(reader)
	if err != nil {
		return
	}
	T.Query, err = ReadString(reader)
	if err != nil {
		return
	}
	var ParameterDataTypesLength int32
	ParameterDataTypesLength, err = ReadInt32(reader)
	if err != nil {
		return
	}
	T.ParameterDataTypes = make([]int32, int(ParameterDataTypesLength))
	for i := 0; i < int(ParameterDataTypesLength); i++ {
		T.ParameterDataTypes[i], err = ReadInt32(reader)
		if err != nil {
			return
		}
	}
	return
}

type Parse struct {
	fields FieldsParse
}

type FieldsPasswordMessage struct {
	Password string
}

func (T *FieldsPasswordMessage) Read(payloadLength int, reader io.Reader) (err error) {
	T.Password, err = ReadString(reader)
	if err != nil {
		return
	}
	return
}

type PasswordMessage struct {
	fields FieldsPasswordMessage
}

type FieldsQuery struct {
	Query string
}

func (T *FieldsQuery) Read(payloadLength int, reader io.Reader) (err error) {
	T.Query, err = ReadString(reader)
	if err != nil {
		return
	}
	return
}

type Query struct {
	fields FieldsQuery
}

type FieldsSASLInitialResponse struct {
	Mechanism       string
	InitialResponse []byte
}

func (T *FieldsSASLInitialResponse) Read(payloadLength int, reader io.Reader) (err error) {
	T.Mechanism, err = ReadString(reader)
	if err != nil {
		return
	}
	var InitialResponseLength int32
	InitialResponseLength, err = ReadInt32(reader)
	if err != nil {
		return
	}
	T.InitialResponse = make([]byte, int(InitialResponseLength))
	for i := 0; i < int(InitialResponseLength); i++ {
		T.InitialResponse[i], err = ReadByte(reader)
		if err != nil {
			return
		}
	}
	return
}

type SASLInitialResponse struct {
	fields FieldsSASLInitialResponse
}

type FieldsSASLResponse struct {
	Data []byte
}

func (T *FieldsSASLResponse) Read(payloadLength int, reader io.Reader) (err error) {
	DataLength := payloadLength
	T.Data = make([]byte, int(DataLength))
	for i := 0; i < int(DataLength); i++ {
		T.Data[i], err = ReadByte(reader)
		if err != nil {
			return
		}
	}
	return
}

type SASLResponse struct {
	fields FieldsSASLResponse
}

type FieldsSSLRequest struct {
	SSLRequestCode int32
}

func (T *FieldsSSLRequest) Read(payloadLength int, reader io.Reader) (err error) {
	T.SSLRequestCode, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return
}

type SSLRequest struct {
	fields FieldsSSLRequest
}

type FieldsStartupMessage struct {
	ProtocolVersionNumber int32
	ParameterName         string
	ParameterValue        string
}

func (T *FieldsStartupMessage) Read(payloadLength int, reader io.Reader) (err error) {
	T.ProtocolVersionNumber, err = ReadInt32(reader)
	if err != nil {
		return
	}
	T.ParameterName, err = ReadString(reader)
	if err != nil {
		return
	}
	T.ParameterValue, err = ReadString(reader)
	if err != nil {
		return
	}
	return
}

type StartupMessage struct {
	fields FieldsStartupMessage
}

type FieldsSync struct {
}

func (T *FieldsSync) Read(payloadLength int, reader io.Reader) (err error) {
	return
}

type Sync struct {
	fields FieldsSync
}

type FieldsTerminate struct {
}

func (T *FieldsTerminate) Read(payloadLength int, reader io.Reader) (err error) {
	return
}

type Terminate struct {
	fields FieldsTerminate
}
