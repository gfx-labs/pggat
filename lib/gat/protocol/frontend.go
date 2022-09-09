package protocol

import (
	"bytes"
	"gfx.cafe/util/go/bufpool"
	"io"
)

// codegen: modify for debug only

var _ bytes.Buffer
var _ io.Reader

type FieldsAuthenticationResponse struct {
	Data []byte
}

func (T *FieldsAuthenticationResponse) Read(payloadLength int, reader io.Reader) (err error) {
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

func (T *FieldsAuthenticationResponse) Write(writer io.Writer) (length int, err error) {
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

type AuthenticationResponse struct {
	Fields FieldsAuthenticationResponse
}

// Read reads all but the packet identifier
// WARNING: This packet DOES have an identifier. Call protocol.Read or trim the identifier first!
func (T *AuthenticationResponse) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *AuthenticationResponse) Write(writer io.Writer) (length int, err error) {
	buf := bufpool.Get(0)
	buf.Reset()
	defer bufpool.Put(buf)
	length, err = T.Fields.Write(buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('p'))
	if err != nil {
		length = 1
		return
	}
	_, err = WriteInt32(writer, int32(length)+4)
	if err != nil {
		length += 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
}

var _ Packet = (*AuthenticationResponse)(nil)

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
		T.Value = nil
	} else {
		T.Value = make([]byte, int(ValueLength))
		for i := 0; i < int(ValueLength); i++ {
			T.Value[i], err = ReadByte(reader)
			if err != nil {
				return
			}
		}
	}
	return
}

func (T *FieldsBindParameterValues) Write(writer io.Writer) (length int, err error) {
	var temp int
	if T.Value == nil {
		temp, err = WriteInt32(writer, int32(-1))
	} else {
		temp, err = WriteInt32(writer, int32(len(T.Value)))
	}
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
	Destination          string
	PreparedStatement    string
	ParameterFormatCodes []int16
	ParameterValues      []FieldsBindParameterValues
	ResultFormatCodes    []int16
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
	var ParameterFormatCodesLength int16
	ParameterFormatCodesLength, err = ReadInt16(reader)
	if err != nil {
		return
	}
	if ParameterFormatCodesLength == int16(-1) {
		T.ParameterFormatCodes = nil
	} else {
		T.ParameterFormatCodes = make([]int16, int(ParameterFormatCodesLength))
		for i := 0; i < int(ParameterFormatCodesLength); i++ {
			T.ParameterFormatCodes[i], err = ReadInt16(reader)
			if err != nil {
				return
			}
		}
	}
	var ParameterValuesLength int16
	ParameterValuesLength, err = ReadInt16(reader)
	if err != nil {
		return
	}
	if ParameterValuesLength == int16(-1) {
		T.ParameterValues = nil
	} else {
		T.ParameterValues = make([]FieldsBindParameterValues, int(ParameterValuesLength))
		for i := 0; i < int(ParameterValuesLength); i++ {
			err = T.ParameterValues[i].Read(payloadLength, reader)
			if err != nil {
				return
			}
		}
	}
	var ResultFormatCodesLength int16
	ResultFormatCodesLength, err = ReadInt16(reader)
	if err != nil {
		return
	}
	if ResultFormatCodesLength == int16(-1) {
		T.ResultFormatCodes = nil
	} else {
		T.ResultFormatCodes = make([]int16, int(ResultFormatCodesLength))
		for i := 0; i < int(ResultFormatCodesLength); i++ {
			T.ResultFormatCodes[i], err = ReadInt16(reader)
			if err != nil {
				return
			}
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
	if T.ParameterFormatCodes == nil {
		temp, err = WriteInt16(writer, int16(-1))
	} else {
		temp, err = WriteInt16(writer, int16(len(T.ParameterFormatCodes)))
	}
	if err != nil {
		return
	}
	length += temp
	for _, v := range T.ParameterFormatCodes {
		temp, err = WriteInt16(writer, v)
		if err != nil {
			return
		}
		length += temp
	}
	if T.ParameterValues == nil {
		temp, err = WriteInt16(writer, int16(-1))
	} else {
		temp, err = WriteInt16(writer, int16(len(T.ParameterValues)))
	}
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
	if T.ResultFormatCodes == nil {
		temp, err = WriteInt16(writer, int16(-1))
	} else {
		temp, err = WriteInt16(writer, int16(len(T.ResultFormatCodes)))
	}
	if err != nil {
		return
	}
	length += temp
	for _, v := range T.ResultFormatCodes {
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
	Fields FieldsBind
}

// Read reads all but the packet identifier
// WARNING: This packet DOES have an identifier. Call protocol.Read or trim the identifier first!
func (T *Bind) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *Bind) Write(writer io.Writer) (length int, err error) {
	buf := bufpool.Get(0)
	buf.Reset()
	defer bufpool.Put(buf)
	length, err = T.Fields.Write(buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('B'))
	if err != nil {
		length = 1
		return
	}
	_, err = WriteInt32(writer, int32(length)+4)
	if err != nil {
		length += 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
}

var _ Packet = (*Bind)(nil)

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
	Fields FieldsClose
}

// Read reads all but the packet identifier
// WARNING: This packet DOES have an identifier. Call protocol.Read or trim the identifier first!
func (T *Close) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *Close) Write(writer io.Writer) (length int, err error) {
	buf := bufpool.Get(0)
	buf.Reset()
	defer bufpool.Put(buf)
	length, err = T.Fields.Write(buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('C'))
	if err != nil {
		length = 1
		return
	}
	_, err = WriteInt32(writer, int32(length)+4)
	if err != nil {
		length += 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
}

var _ Packet = (*Close)(nil)

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
	Fields FieldsCopyFail
}

// Read reads all but the packet identifier
// WARNING: This packet DOES have an identifier. Call protocol.Read or trim the identifier first!
func (T *CopyFail) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *CopyFail) Write(writer io.Writer) (length int, err error) {
	buf := bufpool.Get(0)
	buf.Reset()
	defer bufpool.Put(buf)
	length, err = T.Fields.Write(buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('f'))
	if err != nil {
		length = 1
		return
	}
	_, err = WriteInt32(writer, int32(length)+4)
	if err != nil {
		length += 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
}

var _ Packet = (*CopyFail)(nil)

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
	Fields FieldsDescribe
}

// Read reads all but the packet identifier
// WARNING: This packet DOES have an identifier. Call protocol.Read or trim the identifier first!
func (T *Describe) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *Describe) Write(writer io.Writer) (length int, err error) {
	buf := bufpool.Get(0)
	buf.Reset()
	defer bufpool.Put(buf)
	length, err = T.Fields.Write(buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('D'))
	if err != nil {
		length = 1
		return
	}
	_, err = WriteInt32(writer, int32(length)+4)
	if err != nil {
		length += 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
}

var _ Packet = (*Describe)(nil)

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
	Fields FieldsExecute
}

// Read reads all but the packet identifier
// WARNING: This packet DOES have an identifier. Call protocol.Read or trim the identifier first!
func (T *Execute) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *Execute) Write(writer io.Writer) (length int, err error) {
	buf := bufpool.Get(0)
	buf.Reset()
	defer bufpool.Put(buf)
	length, err = T.Fields.Write(buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('E'))
	if err != nil {
		length = 1
		return
	}
	_, err = WriteInt32(writer, int32(length)+4)
	if err != nil {
		length += 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
}

var _ Packet = (*Execute)(nil)

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
	Fields FieldsFlush
}

// Read reads all but the packet identifier
// WARNING: This packet DOES have an identifier. Call protocol.Read or trim the identifier first!
func (T *Flush) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *Flush) Write(writer io.Writer) (length int, err error) {
	buf := bufpool.Get(0)
	buf.Reset()
	defer bufpool.Put(buf)
	length, err = T.Fields.Write(buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('H'))
	if err != nil {
		length = 1
		return
	}
	_, err = WriteInt32(writer, int32(length)+4)
	if err != nil {
		length += 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
}

var _ Packet = (*Flush)(nil)

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
		T.Value = nil
	} else {
		T.Value = make([]byte, int(ValueLength))
		for i := 0; i < int(ValueLength); i++ {
			T.Value[i], err = ReadByte(reader)
			if err != nil {
				return
			}
		}
	}
	return
}

func (T *FieldsFunctionCallArguments) Write(writer io.Writer) (length int, err error) {
	var temp int
	if T.Value == nil {
		temp, err = WriteInt32(writer, int32(-1))
	} else {
		temp, err = WriteInt32(writer, int32(len(T.Value)))
	}
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
		T.ArgumentFormatCodes = nil
	} else {
		T.ArgumentFormatCodes = make([]int16, int(ArgumentFormatCodesLength))
		for i := 0; i < int(ArgumentFormatCodesLength); i++ {
			T.ArgumentFormatCodes[i], err = ReadInt16(reader)
			if err != nil {
				return
			}
		}
	}
	var ArgumentsLength int16
	ArgumentsLength, err = ReadInt16(reader)
	if err != nil {
		return
	}
	if ArgumentsLength == int16(-1) {
		T.Arguments = nil
	} else {
		T.Arguments = make([]FieldsFunctionCallArguments, int(ArgumentsLength))
		for i := 0; i < int(ArgumentsLength); i++ {
			err = T.Arguments[i].Read(payloadLength, reader)
			if err != nil {
				return
			}
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
	if T.ArgumentFormatCodes == nil {
		temp, err = WriteInt16(writer, int16(-1))
	} else {
		temp, err = WriteInt16(writer, int16(len(T.ArgumentFormatCodes)))
	}
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
	if T.Arguments == nil {
		temp, err = WriteInt16(writer, int16(-1))
	} else {
		temp, err = WriteInt16(writer, int16(len(T.Arguments)))
	}
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
	Fields FieldsFunctionCall
}

// Read reads all but the packet identifier
// WARNING: This packet DOES have an identifier. Call protocol.Read or trim the identifier first!
func (T *FunctionCall) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *FunctionCall) Write(writer io.Writer) (length int, err error) {
	buf := bufpool.Get(0)
	buf.Reset()
	defer bufpool.Put(buf)
	length, err = T.Fields.Write(buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('F'))
	if err != nil {
		length = 1
		return
	}
	_, err = WriteInt32(writer, int32(length)+4)
	if err != nil {
		length += 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
}

var _ Packet = (*FunctionCall)(nil)

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
	Fields FieldsGSSENCRequest
}

// Read reads all but the packet identifier
func (T *GSSENCRequest) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *GSSENCRequest) Write(writer io.Writer) (length int, err error) {
	buf := bufpool.Get(0)
	buf.Reset()
	defer bufpool.Put(buf)
	length, err = T.Fields.Write(buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteInt32(writer, int32(length)+4)
	if err != nil {
		length += 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
}

var _ Packet = (*GSSENCRequest)(nil)

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
	var ParameterDataTypesLength int16
	ParameterDataTypesLength, err = ReadInt16(reader)
	if err != nil {
		return
	}
	if ParameterDataTypesLength == int16(-1) {
		T.ParameterDataTypes = nil
	} else {
		T.ParameterDataTypes = make([]int32, int(ParameterDataTypesLength))
		for i := 0; i < int(ParameterDataTypesLength); i++ {
			T.ParameterDataTypes[i], err = ReadInt32(reader)
			if err != nil {
				return
			}
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
	if T.ParameterDataTypes == nil {
		temp, err = WriteInt16(writer, int16(-1))
	} else {
		temp, err = WriteInt16(writer, int16(len(T.ParameterDataTypes)))
	}
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
	Fields FieldsParse
}

// Read reads all but the packet identifier
// WARNING: This packet DOES have an identifier. Call protocol.Read or trim the identifier first!
func (T *Parse) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *Parse) Write(writer io.Writer) (length int, err error) {
	buf := bufpool.Get(0)
	buf.Reset()
	defer bufpool.Put(buf)
	length, err = T.Fields.Write(buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('P'))
	if err != nil {
		length = 1
		return
	}
	_, err = WriteInt32(writer, int32(length)+4)
	if err != nil {
		length += 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
}

var _ Packet = (*Parse)(nil)

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
	Fields FieldsQuery
}

// Read reads all but the packet identifier
// WARNING: This packet DOES have an identifier. Call protocol.Read or trim the identifier first!
func (T *Query) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *Query) Write(writer io.Writer) (length int, err error) {
	buf := bufpool.Get(0)
	buf.Reset()
	defer bufpool.Put(buf)
	length, err = T.Fields.Write(buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('Q'))
	if err != nil {
		length = 1
		return
	}
	_, err = WriteInt32(writer, int32(length)+4)
	if err != nil {
		length += 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
}

var _ Packet = (*Query)(nil)

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
	Fields FieldsSSLRequest
}

// Read reads all but the packet identifier
func (T *SSLRequest) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *SSLRequest) Write(writer io.Writer) (length int, err error) {
	buf := bufpool.Get(0)
	buf.Reset()
	defer bufpool.Put(buf)
	length, err = T.Fields.Write(buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteInt32(writer, int32(length)+4)
	if err != nil {
		length += 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
}

var _ Packet = (*SSLRequest)(nil)

type FieldsStartupMessageParameters struct {
	Name  string
	Value string
}

func (T *FieldsStartupMessageParameters) Read(payloadLength int, reader io.Reader) (err error) {
	T.Name, err = ReadString(reader)
	if err != nil {
		return
	}
	if T.Name != "" {
		T.Value, err = ReadString(reader)
		if err != nil {
			return
		}
	}
	return
}

func (T *FieldsStartupMessageParameters) Write(writer io.Writer) (length int, err error) {
	var temp int
	temp, err = WriteString(writer, T.Name)
	if err != nil {
		return
	}
	length += temp
	if T.Name != "" {
		temp, err = WriteString(writer, T.Value)
		if err != nil {
			return
		}
		length += temp
	}
	_ = temp
	return
}

type FieldsStartupMessage struct {
	ProtocolVersionNumber int32
	ProcessKey            int32
	SecretKey             int32
	Parameters            []FieldsStartupMessageParameters
}

func (T *FieldsStartupMessage) Read(payloadLength int, reader io.Reader) (err error) {
	T.ProtocolVersionNumber, err = ReadInt32(reader)
	if err != nil {
		return
	}
	if T.ProtocolVersionNumber == 80877102 {
		T.ProcessKey, err = ReadInt32(reader)
		if err != nil {
			return
		}
	}
	if T.ProtocolVersionNumber == 80877102 {
		T.SecretKey, err = ReadInt32(reader)
		if err != nil {
			return
		}
	}
	if T.ProtocolVersionNumber == 196608 {
		var P FieldsStartupMessageParameters
		for ok := true; ok; ok = P.Name != "" {
			var newP FieldsStartupMessageParameters
			err = newP.Read(payloadLength, reader)
			if err != nil {
				return
			}
			T.Parameters = append(T.Parameters, newP)
			P = newP
		}
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
	if T.ProtocolVersionNumber == 80877102 {
		temp, err = WriteInt32(writer, T.ProcessKey)
		if err != nil {
			return
		}
		length += temp
	}
	if T.ProtocolVersionNumber == 80877102 {
		temp, err = WriteInt32(writer, T.SecretKey)
		if err != nil {
			return
		}
		length += temp
	}
	if T.ProtocolVersionNumber == 196608 {
		for _, v := range T.Parameters {
			temp, err = v.Write(writer)
			if err != nil {
				return
			}
			length += temp
		}
	}
	_ = temp
	return
}

type StartupMessage struct {
	Fields FieldsStartupMessage
}

// Read reads all but the packet identifier
func (T *StartupMessage) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *StartupMessage) Write(writer io.Writer) (length int, err error) {
	buf := bufpool.Get(0)
	buf.Reset()
	defer bufpool.Put(buf)
	length, err = T.Fields.Write(buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteInt32(writer, int32(length)+4)
	if err != nil {
		length += 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
}

var _ Packet = (*StartupMessage)(nil)

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
	Fields FieldsSync
}

// Read reads all but the packet identifier
// WARNING: This packet DOES have an identifier. Call protocol.Read or trim the identifier first!
func (T *Sync) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *Sync) Write(writer io.Writer) (length int, err error) {
	buf := bufpool.Get(0)
	buf.Reset()
	defer bufpool.Put(buf)
	length, err = T.Fields.Write(buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('S'))
	if err != nil {
		length = 1
		return
	}
	_, err = WriteInt32(writer, int32(length)+4)
	if err != nil {
		length += 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
}

var _ Packet = (*Sync)(nil)

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
	Fields FieldsTerminate
}

// Read reads all but the packet identifier
// WARNING: This packet DOES have an identifier. Call protocol.Read or trim the identifier first!
func (T *Terminate) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *Terminate) Write(writer io.Writer) (length int, err error) {
	buf := bufpool.Get(0)
	buf.Reset()
	defer bufpool.Put(buf)
	length, err = T.Fields.Write(buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('X'))
	if err != nil {
		length = 1
		return
	}
	_, err = WriteInt32(writer, int32(length)+4)
	if err != nil {
		length += 5
		return
	}
	length += 5
	_, err = writer.Write(buf.Bytes())
	return
}

var _ Packet = (*Terminate)(nil)
