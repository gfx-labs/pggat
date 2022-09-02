package protocol

import (
	"bytes"
	"io"
)

// codegen: modify for debug only

var _ bytes.Buffer
var _ io.Reader

type FieldsBindParameterValues struct {
	Value []byte
}

func (T *FieldsBindParameterValues) Read(payloadLength int, reader io.Reader) (err error) {
	var ValueLength int32
	ValueLength, err = ReadInt32(reader)
	if err != nil {
		return
	}
	if ValueLength == int32(-1) {
		ValueLength = 0
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

func (T *FieldsBindParameterValues) Write(writer io.Writer) (length int, err error) {
	var temp int
	temp, err = WriteInt32(writer, int32(len(T.Value)))
	if err != nil {
		return
	}
	length += temp
	for _, v := range T.Value {
		temp, err = WriteByte(writer, v)
		if err != nil {
			return
		}
		length += temp
	}
	_ = temp
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
	if FormatCodesLength == int16(-1) {
		FormatCodesLength = 0
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
	if ParameterValuesLength == int16(-1) {
		ParameterValuesLength = 0
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
	if ResultColumnFormatCodesLength == int16(-1) {
		ResultColumnFormatCodesLength = 0
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

func (T *FieldsBind) Write(writer io.Writer) (length int, err error) {
	var temp int
	temp, err = WriteString(writer, T.Destination)
	if err != nil {
		return
	}
	length += temp
	temp, err = WriteString(writer, T.PreparedStatement)
	if err != nil {
		return
	}
	length += temp
	temp, err = WriteInt16(writer, int16(len(T.FormatCodes)))
	if err != nil {
		return
	}
	length += temp
	for _, v := range T.FormatCodes {
		temp, err = WriteInt16(writer, v)
		if err != nil {
			return
		}
		length += temp
	}
	temp, err = WriteInt16(writer, int16(len(T.ParameterValues)))
	if err != nil {
		return
	}
	length += temp
	for _, v := range T.ParameterValues {
		temp, err = v.Write(writer)
		if err != nil {
			return
		}
		length += temp
	}
	temp, err = WriteInt16(writer, int16(len(T.ResultColumnFormatCodes)))
	if err != nil {
		return
	}
	length += temp
	for _, v := range T.ResultColumnFormatCodes {
		temp, err = WriteInt16(writer, v)
		if err != nil {
			return
		}
		length += temp
	}
	_ = temp
	return
}

type Bind struct {
	fields FieldsBind
}

// Read reads all but the packet identifier. Be sure to read that beforehand (if it exists)
func (T *Bind) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.fields.Read(int(length-4), reader)
}

func (T *Bind) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('B'))
	if err != nil {
		length = 1
		return
	}
	_, err = WriteInt32(writer, int32(length))
	if err != nil {
		length = 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
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

func (T *FieldsCancelRequest) Write(writer io.Writer) (length int, err error) {
	var temp int
	temp, err = WriteInt32(writer, T.RequestCode)
	if err != nil {
		return
	}
	length += temp
	temp, err = WriteInt32(writer, T.ProcessID)
	if err != nil {
		return
	}
	length += temp
	temp, err = WriteInt32(writer, T.SecretKey)
	if err != nil {
		return
	}
	length += temp
	_ = temp
	return
}

type CancelRequest struct {
	fields FieldsCancelRequest
}

// Read reads all but the packet identifier. Be sure to read that beforehand (if it exists)
func (T *CancelRequest) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.fields.Read(int(length-4), reader)
}

func (T *CancelRequest) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('F'))
	if err != nil {
		length = 1
		return
	}
	_, err = WriteInt32(writer, int32(length))
	if err != nil {
		length = 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
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

func (T *FieldsClose) Write(writer io.Writer) (length int, err error) {
	var temp int
	temp, err = WriteByte(writer, T.Which)
	if err != nil {
		return
	}
	length += temp
	temp, err = WriteString(writer, T.Name)
	if err != nil {
		return
	}
	length += temp
	_ = temp
	return
}

type Close struct {
	fields FieldsClose
}

// Read reads all but the packet identifier. Be sure to read that beforehand (if it exists)
func (T *Close) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.fields.Read(int(length-4), reader)
}

func (T *Close) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('C'))
	if err != nil {
		length = 1
		return
	}
	_, err = WriteInt32(writer, int32(length))
	if err != nil {
		length = 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
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

func (T *FieldsCopyFail) Write(writer io.Writer) (length int, err error) {
	var temp int
	temp, err = WriteString(writer, T.Cause)
	if err != nil {
		return
	}
	length += temp
	_ = temp
	return
}

type CopyFail struct {
	fields FieldsCopyFail
}

// Read reads all but the packet identifier. Be sure to read that beforehand (if it exists)
func (T *CopyFail) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.fields.Read(int(length-4), reader)
}

func (T *CopyFail) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('f'))
	if err != nil {
		length = 1
		return
	}
	_, err = WriteInt32(writer, int32(length))
	if err != nil {
		length = 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
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

func (T *FieldsDescribe) Write(writer io.Writer) (length int, err error) {
	var temp int
	temp, err = WriteByte(writer, T.Which)
	if err != nil {
		return
	}
	length += temp
	temp, err = WriteString(writer, T.Name)
	if err != nil {
		return
	}
	length += temp
	_ = temp
	return
}

type Describe struct {
	fields FieldsDescribe
}

// Read reads all but the packet identifier. Be sure to read that beforehand (if it exists)
func (T *Describe) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.fields.Read(int(length-4), reader)
}

func (T *Describe) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('D'))
	if err != nil {
		length = 1
		return
	}
	_, err = WriteInt32(writer, int32(length))
	if err != nil {
		length = 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
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

func (T *FieldsExecute) Write(writer io.Writer) (length int, err error) {
	var temp int
	temp, err = WriteString(writer, T.Name)
	if err != nil {
		return
	}
	length += temp
	temp, err = WriteInt32(writer, T.MaxRows)
	if err != nil {
		return
	}
	length += temp
	_ = temp
	return
}

type Execute struct {
	fields FieldsExecute
}

// Read reads all but the packet identifier. Be sure to read that beforehand (if it exists)
func (T *Execute) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.fields.Read(int(length-4), reader)
}

func (T *Execute) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('E'))
	if err != nil {
		length = 1
		return
	}
	_, err = WriteInt32(writer, int32(length))
	if err != nil {
		length = 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
}

type FieldsFlush struct {
}

func (T *FieldsFlush) Read(payloadLength int, reader io.Reader) (err error) {
	return
}

func (T *FieldsFlush) Write(writer io.Writer) (length int, err error) {
	var temp int
	_ = temp
	return
}

type Flush struct {
	fields FieldsFlush
}

// Read reads all but the packet identifier. Be sure to read that beforehand (if it exists)
func (T *Flush) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.fields.Read(int(length-4), reader)
}

func (T *Flush) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('H'))
	if err != nil {
		length = 1
		return
	}
	_, err = WriteInt32(writer, int32(length))
	if err != nil {
		length = 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
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
	if ValueLength == int32(-1) {
		ValueLength = 0
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

func (T *FieldsFunctionCallArguments) Write(writer io.Writer) (length int, err error) {
	var temp int
	temp, err = WriteInt32(writer, int32(len(T.Value)))
	if err != nil {
		return
	}
	length += temp
	for _, v := range T.Value {
		temp, err = WriteByte(writer, v)
		if err != nil {
			return
		}
		length += temp
	}
	_ = temp
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
	if ArgumentFormatCodesLength == int16(-1) {
		ArgumentFormatCodesLength = 0
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
	if ArgumentsLength == int16(-1) {
		ArgumentsLength = 0
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

func (T *FieldsFunctionCall) Write(writer io.Writer) (length int, err error) {
	var temp int
	temp, err = WriteInt32(writer, T.Function)
	if err != nil {
		return
	}
	length += temp
	temp, err = WriteInt16(writer, int16(len(T.ArgumentFormatCodes)))
	if err != nil {
		return
	}
	length += temp
	for _, v := range T.ArgumentFormatCodes {
		temp, err = WriteInt16(writer, v)
		if err != nil {
			return
		}
		length += temp
	}
	temp, err = WriteInt16(writer, int16(len(T.Arguments)))
	if err != nil {
		return
	}
	length += temp
	for _, v := range T.Arguments {
		temp, err = v.Write(writer)
		if err != nil {
			return
		}
		length += temp
	}
	temp, err = WriteInt16(writer, T.ResultFormatCode)
	if err != nil {
		return
	}
	length += temp
	_ = temp
	return
}

type FunctionCall struct {
	fields FieldsFunctionCall
}

// Read reads all but the packet identifier. Be sure to read that beforehand (if it exists)
func (T *FunctionCall) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.fields.Read(int(length-4), reader)
}

func (T *FunctionCall) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('F'))
	if err != nil {
		length = 1
		return
	}
	_, err = WriteInt32(writer, int32(length))
	if err != nil {
		length = 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
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

func (T *FieldsGSSENCRequest) Write(writer io.Writer) (length int, err error) {
	var temp int
	temp, err = WriteInt32(writer, T.EncryptionRequestCode)
	if err != nil {
		return
	}
	length += temp
	_ = temp
	return
}

type GSSENCRequest struct {
	fields FieldsGSSENCRequest
}

// Read reads all but the packet identifier. Be sure to read that beforehand (if it exists)
func (T *GSSENCRequest) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.fields.Read(int(length-4), reader)
}

func (T *GSSENCRequest) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteInt32(writer, int32(length))
	if err != nil {
		length = 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
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

func (T *FieldsGSSResponse) Write(writer io.Writer) (length int, err error) {
	var temp int
	for _, v := range T.Data {
		temp, err = WriteByte(writer, v)
		if err != nil {
			return
		}
		length += temp
	}
	_ = temp
	return
}

type GSSResponse struct {
	fields FieldsGSSResponse
}

// Read reads all but the packet identifier. Be sure to read that beforehand (if it exists)
func (T *GSSResponse) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.fields.Read(int(length-4), reader)
}

func (T *GSSResponse) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('p'))
	if err != nil {
		length = 1
		return
	}
	_, err = WriteInt32(writer, int32(length))
	if err != nil {
		length = 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
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
	if ParameterDataTypesLength == int32(-1) {
		ParameterDataTypesLength = 0
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

func (T *FieldsParse) Write(writer io.Writer) (length int, err error) {
	var temp int
	temp, err = WriteString(writer, T.PreparedStatement)
	if err != nil {
		return
	}
	length += temp
	temp, err = WriteString(writer, T.Query)
	if err != nil {
		return
	}
	length += temp
	temp, err = WriteInt32(writer, int32(len(T.ParameterDataTypes)))
	if err != nil {
		return
	}
	length += temp
	for _, v := range T.ParameterDataTypes {
		temp, err = WriteInt32(writer, v)
		if err != nil {
			return
		}
		length += temp
	}
	_ = temp
	return
}

type Parse struct {
	fields FieldsParse
}

// Read reads all but the packet identifier. Be sure to read that beforehand (if it exists)
func (T *Parse) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.fields.Read(int(length-4), reader)
}

func (T *Parse) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('P'))
	if err != nil {
		length = 1
		return
	}
	_, err = WriteInt32(writer, int32(length))
	if err != nil {
		length = 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
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

func (T *FieldsPasswordMessage) Write(writer io.Writer) (length int, err error) {
	var temp int
	temp, err = WriteString(writer, T.Password)
	if err != nil {
		return
	}
	length += temp
	_ = temp
	return
}

type PasswordMessage struct {
	fields FieldsPasswordMessage
}

// Read reads all but the packet identifier. Be sure to read that beforehand (if it exists)
func (T *PasswordMessage) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.fields.Read(int(length-4), reader)
}

func (T *PasswordMessage) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('p'))
	if err != nil {
		length = 1
		return
	}
	_, err = WriteInt32(writer, int32(length))
	if err != nil {
		length = 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
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

func (T *FieldsQuery) Write(writer io.Writer) (length int, err error) {
	var temp int
	temp, err = WriteString(writer, T.Query)
	if err != nil {
		return
	}
	length += temp
	_ = temp
	return
}

type Query struct {
	fields FieldsQuery
}

// Read reads all but the packet identifier. Be sure to read that beforehand (if it exists)
func (T *Query) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.fields.Read(int(length-4), reader)
}

func (T *Query) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('Q'))
	if err != nil {
		length = 1
		return
	}
	_, err = WriteInt32(writer, int32(length))
	if err != nil {
		length = 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
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
	if InitialResponseLength == int32(-1) {
		InitialResponseLength = 0
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

func (T *FieldsSASLInitialResponse) Write(writer io.Writer) (length int, err error) {
	var temp int
	temp, err = WriteString(writer, T.Mechanism)
	if err != nil {
		return
	}
	length += temp
	temp, err = WriteInt32(writer, int32(len(T.InitialResponse)))
	if err != nil {
		return
	}
	length += temp
	for _, v := range T.InitialResponse {
		temp, err = WriteByte(writer, v)
		if err != nil {
			return
		}
		length += temp
	}
	_ = temp
	return
}

type SASLInitialResponse struct {
	fields FieldsSASLInitialResponse
}

// Read reads all but the packet identifier. Be sure to read that beforehand (if it exists)
func (T *SASLInitialResponse) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.fields.Read(int(length-4), reader)
}

func (T *SASLInitialResponse) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('p'))
	if err != nil {
		length = 1
		return
	}
	_, err = WriteInt32(writer, int32(length))
	if err != nil {
		length = 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
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

func (T *FieldsSASLResponse) Write(writer io.Writer) (length int, err error) {
	var temp int
	for _, v := range T.Data {
		temp, err = WriteByte(writer, v)
		if err != nil {
			return
		}
		length += temp
	}
	_ = temp
	return
}

type SASLResponse struct {
	fields FieldsSASLResponse
}

// Read reads all but the packet identifier. Be sure to read that beforehand (if it exists)
func (T *SASLResponse) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.fields.Read(int(length-4), reader)
}

func (T *SASLResponse) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('p'))
	if err != nil {
		length = 1
		return
	}
	_, err = WriteInt32(writer, int32(length))
	if err != nil {
		length = 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
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

func (T *FieldsSSLRequest) Write(writer io.Writer) (length int, err error) {
	var temp int
	temp, err = WriteInt32(writer, T.SSLRequestCode)
	if err != nil {
		return
	}
	length += temp
	_ = temp
	return
}

type SSLRequest struct {
	fields FieldsSSLRequest
}

// Read reads all but the packet identifier. Be sure to read that beforehand (if it exists)
func (T *SSLRequest) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.fields.Read(int(length-4), reader)
}

func (T *SSLRequest) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteInt32(writer, int32(length))
	if err != nil {
		length = 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
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

func (T *FieldsStartupMessage) Write(writer io.Writer) (length int, err error) {
	var temp int
	temp, err = WriteInt32(writer, T.ProtocolVersionNumber)
	if err != nil {
		return
	}
	length += temp
	temp, err = WriteString(writer, T.ParameterName)
	if err != nil {
		return
	}
	length += temp
	temp, err = WriteString(writer, T.ParameterValue)
	if err != nil {
		return
	}
	length += temp
	_ = temp
	return
}

type StartupMessage struct {
	fields FieldsStartupMessage
}

// Read reads all but the packet identifier. Be sure to read that beforehand (if it exists)
func (T *StartupMessage) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.fields.Read(int(length-4), reader)
}

func (T *StartupMessage) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteInt32(writer, int32(length))
	if err != nil {
		length = 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
}

type FieldsSync struct {
}

func (T *FieldsSync) Read(payloadLength int, reader io.Reader) (err error) {
	return
}

func (T *FieldsSync) Write(writer io.Writer) (length int, err error) {
	var temp int
	_ = temp
	return
}

type Sync struct {
	fields FieldsSync
}

// Read reads all but the packet identifier. Be sure to read that beforehand (if it exists)
func (T *Sync) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.fields.Read(int(length-4), reader)
}

func (T *Sync) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('S'))
	if err != nil {
		length = 1
		return
	}
	_, err = WriteInt32(writer, int32(length))
	if err != nil {
		length = 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
}

type FieldsTerminate struct {
}

func (T *FieldsTerminate) Read(payloadLength int, reader io.Reader) (err error) {
	return
}

func (T *FieldsTerminate) Write(writer io.Writer) (length int, err error) {
	var temp int
	_ = temp
	return
}

type Terminate struct {
	fields FieldsTerminate
}

// Read reads all but the packet identifier. Be sure to read that beforehand (if it exists)
func (T *Terminate) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.fields.Read(int(length-4), reader)
}

func (T *Terminate) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('X'))
	if err != nil {
		length = 1
		return
	}
	_, err = WriteInt32(writer, int32(length))
	if err != nil {
		length = 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
}
