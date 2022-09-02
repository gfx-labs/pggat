package protocol

import (
	"bytes"
	"io"
)

// codegen: modify for debug only

var _ bytes.Buffer
var _ io.Reader

type FieldsAuthentication struct {
	Data []byte
}

func (T *FieldsAuthentication) Read(payloadLength int, reader io.Reader) (err error) {
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

func (T *FieldsAuthentication) Write(writer io.Writer) (length int, err error) {
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

type Authentication struct {
	Fields FieldsAuthentication
}

// Read reads all but the packet identifier
// WARNING: This packet DOES have an identifier. Call protocol.Read or trim the identifier first!
func (T *Authentication) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *Authentication) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.Fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('R'))
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

var _ Packet = (*Authentication)(nil)

type FieldsBackendKeyData struct {
	ProcessID int32
	SecretKey int32
}

func (T *FieldsBackendKeyData) Read(payloadLength int, reader io.Reader) (err error) {
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

func (T *FieldsBackendKeyData) Write(writer io.Writer) (length int, err error) {
	var temp int
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

type BackendKeyData struct {
	Fields FieldsBackendKeyData
}

// Read reads all but the packet identifier
// WARNING: This packet DOES have an identifier. Call protocol.Read or trim the identifier first!
func (T *BackendKeyData) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *BackendKeyData) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.Fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('K'))
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

var _ Packet = (*BackendKeyData)(nil)

type FieldsBindComplete struct {
}

func (T *FieldsBindComplete) Read(payloadLength int, reader io.Reader) (err error) {
	return
}

func (T *FieldsBindComplete) Write(writer io.Writer) (length int, err error) {
	var temp int
	_ = temp
	return
}

type BindComplete struct {
	Fields FieldsBindComplete
}

// Read reads all but the packet identifier
// WARNING: This packet DOES have an identifier. Call protocol.Read or trim the identifier first!
func (T *BindComplete) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *BindComplete) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.Fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('2'))
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

var _ Packet = (*BindComplete)(nil)

type FieldsCloseComplete struct {
}

func (T *FieldsCloseComplete) Read(payloadLength int, reader io.Reader) (err error) {
	return
}

func (T *FieldsCloseComplete) Write(writer io.Writer) (length int, err error) {
	var temp int
	_ = temp
	return
}

type CloseComplete struct {
	Fields FieldsCloseComplete
}

// Read reads all but the packet identifier
// WARNING: This packet DOES have an identifier. Call protocol.Read or trim the identifier first!
func (T *CloseComplete) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *CloseComplete) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.Fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('3'))
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

var _ Packet = (*CloseComplete)(nil)

type FieldsCommandComplete struct {
	Data string
}

func (T *FieldsCommandComplete) Read(payloadLength int, reader io.Reader) (err error) {
	T.Data, err = ReadString(reader)
	if err != nil {
		return
	}
	return
}

func (T *FieldsCommandComplete) Write(writer io.Writer) (length int, err error) {
	var temp int
	temp, err = WriteString(writer, T.Data)
	if err != nil {
		return
	}
	length += temp
	_ = temp
	return
}

type CommandComplete struct {
	Fields FieldsCommandComplete
}

// Read reads all but the packet identifier
// WARNING: This packet DOES have an identifier. Call protocol.Read or trim the identifier first!
func (T *CommandComplete) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *CommandComplete) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.Fields.Write(&buf)
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

var _ Packet = (*CommandComplete)(nil)

type FieldsCopyBothResponse struct {
	Format        int8
	ColumnFormats []int16
}

func (T *FieldsCopyBothResponse) Read(payloadLength int, reader io.Reader) (err error) {
	T.Format, err = ReadInt8(reader)
	if err != nil {
		return
	}
	var ColumnFormatsLength int16
	ColumnFormatsLength, err = ReadInt16(reader)
	if err != nil {
		return
	}
	if ColumnFormatsLength == int16(-1) {
		ColumnFormatsLength = 0
	}
	T.ColumnFormats = make([]int16, int(ColumnFormatsLength))
	for i := 0; i < int(ColumnFormatsLength); i++ {
		T.ColumnFormats[i], err = ReadInt16(reader)
		if err != nil {
			return
		}
	}
	return
}

func (T *FieldsCopyBothResponse) Write(writer io.Writer) (length int, err error) {
	var temp int
	temp, err = WriteInt8(writer, T.Format)
	if err != nil {
		return
	}
	length += temp
	temp, err = WriteInt16(writer, int16(len(T.ColumnFormats)))
	if err != nil {
		return
	}
	length += temp
	for _, v := range T.ColumnFormats {
		temp, err = WriteInt16(writer, v)
		if err != nil {
			return
		}
		length += temp
	}
	_ = temp
	return
}

type CopyBothResponse struct {
	Fields FieldsCopyBothResponse
}

// Read reads all but the packet identifier
// WARNING: This packet DOES have an identifier. Call protocol.Read or trim the identifier first!
func (T *CopyBothResponse) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *CopyBothResponse) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.Fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('W'))
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

var _ Packet = (*CopyBothResponse)(nil)

type FieldsCopyInResponse struct {
	Format        int8
	ColumnFormats []int16
}

func (T *FieldsCopyInResponse) Read(payloadLength int, reader io.Reader) (err error) {
	T.Format, err = ReadInt8(reader)
	if err != nil {
		return
	}
	var ColumnFormatsLength int16
	ColumnFormatsLength, err = ReadInt16(reader)
	if err != nil {
		return
	}
	if ColumnFormatsLength == int16(-1) {
		ColumnFormatsLength = 0
	}
	T.ColumnFormats = make([]int16, int(ColumnFormatsLength))
	for i := 0; i < int(ColumnFormatsLength); i++ {
		T.ColumnFormats[i], err = ReadInt16(reader)
		if err != nil {
			return
		}
	}
	return
}

func (T *FieldsCopyInResponse) Write(writer io.Writer) (length int, err error) {
	var temp int
	temp, err = WriteInt8(writer, T.Format)
	if err != nil {
		return
	}
	length += temp
	temp, err = WriteInt16(writer, int16(len(T.ColumnFormats)))
	if err != nil {
		return
	}
	length += temp
	for _, v := range T.ColumnFormats {
		temp, err = WriteInt16(writer, v)
		if err != nil {
			return
		}
		length += temp
	}
	_ = temp
	return
}

type CopyInResponse struct {
	Fields FieldsCopyInResponse
}

// Read reads all but the packet identifier
// WARNING: This packet DOES have an identifier. Call protocol.Read or trim the identifier first!
func (T *CopyInResponse) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *CopyInResponse) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.Fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('G'))
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

var _ Packet = (*CopyInResponse)(nil)

type FieldsCopyOutResponse struct {
	Format        int8
	ColumnFormats []int16
}

func (T *FieldsCopyOutResponse) Read(payloadLength int, reader io.Reader) (err error) {
	T.Format, err = ReadInt8(reader)
	if err != nil {
		return
	}
	var ColumnFormatsLength int16
	ColumnFormatsLength, err = ReadInt16(reader)
	if err != nil {
		return
	}
	if ColumnFormatsLength == int16(-1) {
		ColumnFormatsLength = 0
	}
	T.ColumnFormats = make([]int16, int(ColumnFormatsLength))
	for i := 0; i < int(ColumnFormatsLength); i++ {
		T.ColumnFormats[i], err = ReadInt16(reader)
		if err != nil {
			return
		}
	}
	return
}

func (T *FieldsCopyOutResponse) Write(writer io.Writer) (length int, err error) {
	var temp int
	temp, err = WriteInt8(writer, T.Format)
	if err != nil {
		return
	}
	length += temp
	temp, err = WriteInt16(writer, int16(len(T.ColumnFormats)))
	if err != nil {
		return
	}
	length += temp
	for _, v := range T.ColumnFormats {
		temp, err = WriteInt16(writer, v)
		if err != nil {
			return
		}
		length += temp
	}
	_ = temp
	return
}

type CopyOutResponse struct {
	Fields FieldsCopyOutResponse
}

// Read reads all but the packet identifier
// WARNING: This packet DOES have an identifier. Call protocol.Read or trim the identifier first!
func (T *CopyOutResponse) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *CopyOutResponse) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.Fields.Write(&buf)
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

var _ Packet = (*CopyOutResponse)(nil)

type FieldsDataRowColumns struct {
	Bytes []int8
}

func (T *FieldsDataRowColumns) Read(payloadLength int, reader io.Reader) (err error) {
	var BytesLength int32
	BytesLength, err = ReadInt32(reader)
	if err != nil {
		return
	}
	if BytesLength == int32(-1) {
		BytesLength = 0
	}
	T.Bytes = make([]int8, int(BytesLength))
	for i := 0; i < int(BytesLength); i++ {
		T.Bytes[i], err = ReadInt8(reader)
		if err != nil {
			return
		}
	}
	return
}

func (T *FieldsDataRowColumns) Write(writer io.Writer) (length int, err error) {
	var temp int
	temp, err = WriteInt32(writer, int32(len(T.Bytes)))
	if err != nil {
		return
	}
	length += temp
	for _, v := range T.Bytes {
		temp, err = WriteInt8(writer, v)
		if err != nil {
			return
		}
		length += temp
	}
	_ = temp
	return
}

type FieldsDataRow struct {
	Columns []FieldsDataRowColumns
}

func (T *FieldsDataRow) Read(payloadLength int, reader io.Reader) (err error) {
	var ColumnsLength int16
	ColumnsLength, err = ReadInt16(reader)
	if err != nil {
		return
	}
	if ColumnsLength == int16(-1) {
		ColumnsLength = 0
	}
	T.Columns = make([]FieldsDataRowColumns, int(ColumnsLength))
	for i := 0; i < int(ColumnsLength); i++ {
		err = T.Columns[i].Read(payloadLength, reader)
		if err != nil {
			return
		}
	}
	return
}

func (T *FieldsDataRow) Write(writer io.Writer) (length int, err error) {
	var temp int
	temp, err = WriteInt16(writer, int16(len(T.Columns)))
	if err != nil {
		return
	}
	length += temp
	for _, v := range T.Columns {
		temp, err = v.Write(writer)
		if err != nil {
			return
		}
		length += temp
	}
	_ = temp
	return
}

type DataRow struct {
	Fields FieldsDataRow
}

// Read reads all but the packet identifier
// WARNING: This packet DOES have an identifier. Call protocol.Read or trim the identifier first!
func (T *DataRow) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *DataRow) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.Fields.Write(&buf)
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

var _ Packet = (*DataRow)(nil)

type FieldsEmptyQueryResponse struct {
}

func (T *FieldsEmptyQueryResponse) Read(payloadLength int, reader io.Reader) (err error) {
	return
}

func (T *FieldsEmptyQueryResponse) Write(writer io.Writer) (length int, err error) {
	var temp int
	_ = temp
	return
}

type EmptyQueryResponse struct {
	Fields FieldsEmptyQueryResponse
}

// Read reads all but the packet identifier
// WARNING: This packet DOES have an identifier. Call protocol.Read or trim the identifier first!
func (T *EmptyQueryResponse) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *EmptyQueryResponse) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.Fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('I'))
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

var _ Packet = (*EmptyQueryResponse)(nil)

type FieldsErrorResponseResponses struct {
	Code  byte
	Value string
}

func (T *FieldsErrorResponseResponses) Read(payloadLength int, reader io.Reader) (err error) {
	T.Code, err = ReadByte(reader)
	if err != nil {
		return
	}
	if T.Code != 0 {
		T.Value, err = ReadString(reader)
		if err != nil {
			return
		}
	}
	return
}

func (T *FieldsErrorResponseResponses) Write(writer io.Writer) (length int, err error) {
	var temp int
	temp, err = WriteByte(writer, T.Code)
	if err != nil {
		return
	}
	length += temp
	if T.Code != 0 {
		temp, err = WriteString(writer, T.Value)
		if err != nil {
			return
		}
		length += temp
	}
	_ = temp
	return
}

type FieldsErrorResponse struct {
	Responses []FieldsErrorResponseResponses
}

func (T *FieldsErrorResponse) Read(payloadLength int, reader io.Reader) (err error) {
	var P FieldsErrorResponseResponses
	for ok := true; ok; ok = P.Code != 0 {
		err = P.Read(payloadLength, reader)
		if err != nil {
			return
		}
		T.Responses = append(T.Responses, P)
		var newp FieldsErrorResponseResponses
		P = newp
	}
	return
}

func (T *FieldsErrorResponse) Write(writer io.Writer) (length int, err error) {
	var temp int
	for _, v := range T.Responses {
		temp, err = v.Write(writer)
		if err != nil {
			return
		}
		length += temp
	}
	_ = temp
	return
}

type ErrorResponse struct {
	Fields FieldsErrorResponse
}

// Read reads all but the packet identifier
// WARNING: This packet DOES have an identifier. Call protocol.Read or trim the identifier first!
func (T *ErrorResponse) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *ErrorResponse) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.Fields.Write(&buf)
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

var _ Packet = (*ErrorResponse)(nil)

type FieldsFunctionCallResponse struct {
	Result []byte
}

func (T *FieldsFunctionCallResponse) Read(payloadLength int, reader io.Reader) (err error) {
	var ResultLength int32
	ResultLength, err = ReadInt32(reader)
	if err != nil {
		return
	}
	if ResultLength == int32(-1) {
		ResultLength = 0
	}
	T.Result = make([]byte, int(ResultLength))
	for i := 0; i < int(ResultLength); i++ {
		T.Result[i], err = ReadByte(reader)
		if err != nil {
			return
		}
	}
	return
}

func (T *FieldsFunctionCallResponse) Write(writer io.Writer) (length int, err error) {
	var temp int
	temp, err = WriteInt32(writer, int32(len(T.Result)))
	if err != nil {
		return
	}
	length += temp
	for _, v := range T.Result {
		temp, err = WriteByte(writer, v)
		if err != nil {
			return
		}
		length += temp
	}
	_ = temp
	return
}

type FunctionCallResponse struct {
	Fields FieldsFunctionCallResponse
}

// Read reads all but the packet identifier
// WARNING: This packet DOES have an identifier. Call protocol.Read or trim the identifier first!
func (T *FunctionCallResponse) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *FunctionCallResponse) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.Fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('V'))
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

var _ Packet = (*FunctionCallResponse)(nil)

type FieldsNegotiateProtocolVersion struct {
	MinorVersion  int32
	NotRecognized []string
}

func (T *FieldsNegotiateProtocolVersion) Read(payloadLength int, reader io.Reader) (err error) {
	T.MinorVersion, err = ReadInt32(reader)
	if err != nil {
		return
	}
	var NotRecognizedLength int32
	NotRecognizedLength, err = ReadInt32(reader)
	if err != nil {
		return
	}
	if NotRecognizedLength == int32(-1) {
		NotRecognizedLength = 0
	}
	T.NotRecognized = make([]string, int(NotRecognizedLength))
	for i := 0; i < int(NotRecognizedLength); i++ {
		T.NotRecognized[i], err = ReadString(reader)
		if err != nil {
			return
		}
	}
	return
}

func (T *FieldsNegotiateProtocolVersion) Write(writer io.Writer) (length int, err error) {
	var temp int
	temp, err = WriteInt32(writer, T.MinorVersion)
	if err != nil {
		return
	}
	length += temp
	temp, err = WriteInt32(writer, int32(len(T.NotRecognized)))
	if err != nil {
		return
	}
	length += temp
	for _, v := range T.NotRecognized {
		temp, err = WriteString(writer, v)
		if err != nil {
			return
		}
		length += temp
	}
	_ = temp
	return
}

type NegotiateProtocolVersion struct {
	Fields FieldsNegotiateProtocolVersion
}

// Read reads all but the packet identifier
// WARNING: This packet DOES have an identifier. Call protocol.Read or trim the identifier first!
func (T *NegotiateProtocolVersion) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *NegotiateProtocolVersion) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.Fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('v'))
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

var _ Packet = (*NegotiateProtocolVersion)(nil)

type FieldsNoData struct {
}

func (T *FieldsNoData) Read(payloadLength int, reader io.Reader) (err error) {
	return
}

func (T *FieldsNoData) Write(writer io.Writer) (length int, err error) {
	var temp int
	_ = temp
	return
}

type NoData struct {
	Fields FieldsNoData
}

// Read reads all but the packet identifier
// WARNING: This packet DOES have an identifier. Call protocol.Read or trim the identifier first!
func (T *NoData) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *NoData) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.Fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('n'))
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

var _ Packet = (*NoData)(nil)

type FieldsNoticeResponseResponses struct {
	Code  byte
	Value string
}

func (T *FieldsNoticeResponseResponses) Read(payloadLength int, reader io.Reader) (err error) {
	T.Code, err = ReadByte(reader)
	if err != nil {
		return
	}
	if T.Code != 0 {
		T.Value, err = ReadString(reader)
		if err != nil {
			return
		}
	}
	return
}

func (T *FieldsNoticeResponseResponses) Write(writer io.Writer) (length int, err error) {
	var temp int
	temp, err = WriteByte(writer, T.Code)
	if err != nil {
		return
	}
	length += temp
	if T.Code != 0 {
		temp, err = WriteString(writer, T.Value)
		if err != nil {
			return
		}
		length += temp
	}
	_ = temp
	return
}

type FieldsNoticeResponse struct {
	Responses []FieldsNoticeResponseResponses
}

func (T *FieldsNoticeResponse) Read(payloadLength int, reader io.Reader) (err error) {
	var P FieldsNoticeResponseResponses
	for ok := true; ok; ok = P.Code != 0 {
		err = P.Read(payloadLength, reader)
		if err != nil {
			return
		}
		T.Responses = append(T.Responses, P)
		var newp FieldsNoticeResponseResponses
		P = newp
	}
	return
}

func (T *FieldsNoticeResponse) Write(writer io.Writer) (length int, err error) {
	var temp int
	for _, v := range T.Responses {
		temp, err = v.Write(writer)
		if err != nil {
			return
		}
		length += temp
	}
	_ = temp
	return
}

type NoticeResponse struct {
	Fields FieldsNoticeResponse
}

// Read reads all but the packet identifier
// WARNING: This packet DOES have an identifier. Call protocol.Read or trim the identifier first!
func (T *NoticeResponse) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *NoticeResponse) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.Fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('N'))
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

var _ Packet = (*NoticeResponse)(nil)

type FieldsNotificationResponse struct {
	ProcessID int32
	Channel   string
	Payload   string
}

func (T *FieldsNotificationResponse) Read(payloadLength int, reader io.Reader) (err error) {
	T.ProcessID, err = ReadInt32(reader)
	if err != nil {
		return
	}
	T.Channel, err = ReadString(reader)
	if err != nil {
		return
	}
	T.Payload, err = ReadString(reader)
	if err != nil {
		return
	}
	return
}

func (T *FieldsNotificationResponse) Write(writer io.Writer) (length int, err error) {
	var temp int
	temp, err = WriteInt32(writer, T.ProcessID)
	if err != nil {
		return
	}
	length += temp
	temp, err = WriteString(writer, T.Channel)
	if err != nil {
		return
	}
	length += temp
	temp, err = WriteString(writer, T.Payload)
	if err != nil {
		return
	}
	length += temp
	_ = temp
	return
}

type NotificationResponse struct {
	Fields FieldsNotificationResponse
}

// Read reads all but the packet identifier
// WARNING: This packet DOES have an identifier. Call protocol.Read or trim the identifier first!
func (T *NotificationResponse) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *NotificationResponse) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.Fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('A'))
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

var _ Packet = (*NotificationResponse)(nil)

type FieldsParameterDescription struct {
	Parameters []int32
}

func (T *FieldsParameterDescription) Read(payloadLength int, reader io.Reader) (err error) {
	var ParametersLength int16
	ParametersLength, err = ReadInt16(reader)
	if err != nil {
		return
	}
	if ParametersLength == int16(-1) {
		ParametersLength = 0
	}
	T.Parameters = make([]int32, int(ParametersLength))
	for i := 0; i < int(ParametersLength); i++ {
		T.Parameters[i], err = ReadInt32(reader)
		if err != nil {
			return
		}
	}
	return
}

func (T *FieldsParameterDescription) Write(writer io.Writer) (length int, err error) {
	var temp int
	temp, err = WriteInt16(writer, int16(len(T.Parameters)))
	if err != nil {
		return
	}
	length += temp
	for _, v := range T.Parameters {
		temp, err = WriteInt32(writer, v)
		if err != nil {
			return
		}
		length += temp
	}
	_ = temp
	return
}

type ParameterDescription struct {
	Fields FieldsParameterDescription
}

// Read reads all but the packet identifier
// WARNING: This packet DOES have an identifier. Call protocol.Read or trim the identifier first!
func (T *ParameterDescription) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *ParameterDescription) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.Fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('t'))
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

var _ Packet = (*ParameterDescription)(nil)

type FieldsParameterStatus struct {
	Parameter string
	Value     string
}

func (T *FieldsParameterStatus) Read(payloadLength int, reader io.Reader) (err error) {
	T.Parameter, err = ReadString(reader)
	if err != nil {
		return
	}
	T.Value, err = ReadString(reader)
	if err != nil {
		return
	}
	return
}

func (T *FieldsParameterStatus) Write(writer io.Writer) (length int, err error) {
	var temp int
	temp, err = WriteString(writer, T.Parameter)
	if err != nil {
		return
	}
	length += temp
	temp, err = WriteString(writer, T.Value)
	if err != nil {
		return
	}
	length += temp
	_ = temp
	return
}

type ParameterStatus struct {
	Fields FieldsParameterStatus
}

// Read reads all but the packet identifier
// WARNING: This packet DOES have an identifier. Call protocol.Read or trim the identifier first!
func (T *ParameterStatus) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *ParameterStatus) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.Fields.Write(&buf)
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

var _ Packet = (*ParameterStatus)(nil)

type FieldsParseComplete struct {
}

func (T *FieldsParseComplete) Read(payloadLength int, reader io.Reader) (err error) {
	return
}

func (T *FieldsParseComplete) Write(writer io.Writer) (length int, err error) {
	var temp int
	_ = temp
	return
}

type ParseComplete struct {
	Fields FieldsParseComplete
}

// Read reads all but the packet identifier
// WARNING: This packet DOES have an identifier. Call protocol.Read or trim the identifier first!
func (T *ParseComplete) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *ParseComplete) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.Fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('1'))
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

var _ Packet = (*ParseComplete)(nil)

type FieldsPortalSuspended struct {
}

func (T *FieldsPortalSuspended) Read(payloadLength int, reader io.Reader) (err error) {
	return
}

func (T *FieldsPortalSuspended) Write(writer io.Writer) (length int, err error) {
	var temp int
	_ = temp
	return
}

type PortalSuspended struct {
	Fields FieldsPortalSuspended
}

// Read reads all but the packet identifier
// WARNING: This packet DOES have an identifier. Call protocol.Read or trim the identifier first!
func (T *PortalSuspended) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *PortalSuspended) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.Fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('s'))
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

var _ Packet = (*PortalSuspended)(nil)

type FieldsReadForQuery struct {
}

func (T *FieldsReadForQuery) Read(payloadLength int, reader io.Reader) (err error) {
	return
}

func (T *FieldsReadForQuery) Write(writer io.Writer) (length int, err error) {
	var temp int
	_ = temp
	return
}

type ReadForQuery struct {
	Fields FieldsReadForQuery
}

// Read reads all but the packet identifier
// WARNING: This packet DOES have an identifier. Call protocol.Read or trim the identifier first!
func (T *ReadForQuery) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *ReadForQuery) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.Fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('Z'))
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

var _ Packet = (*ReadForQuery)(nil)

type FieldsRowDescriptionFields struct {
	Name            string
	TableId         int32
	AttributeNumber int16
	DataType        int32
	DataTypeSize    int16
	TypeModifier    int32
	FormatCode      int16
}

func (T *FieldsRowDescriptionFields) Read(payloadLength int, reader io.Reader) (err error) {
	T.Name, err = ReadString(reader)
	if err != nil {
		return
	}
	T.TableId, err = ReadInt32(reader)
	if err != nil {
		return
	}
	T.AttributeNumber, err = ReadInt16(reader)
	if err != nil {
		return
	}
	T.DataType, err = ReadInt32(reader)
	if err != nil {
		return
	}
	T.DataTypeSize, err = ReadInt16(reader)
	if err != nil {
		return
	}
	T.TypeModifier, err = ReadInt32(reader)
	if err != nil {
		return
	}
	T.FormatCode, err = ReadInt16(reader)
	if err != nil {
		return
	}
	return
}

func (T *FieldsRowDescriptionFields) Write(writer io.Writer) (length int, err error) {
	var temp int
	temp, err = WriteString(writer, T.Name)
	if err != nil {
		return
	}
	length += temp
	temp, err = WriteInt32(writer, T.TableId)
	if err != nil {
		return
	}
	length += temp
	temp, err = WriteInt16(writer, T.AttributeNumber)
	if err != nil {
		return
	}
	length += temp
	temp, err = WriteInt32(writer, T.DataType)
	if err != nil {
		return
	}
	length += temp
	temp, err = WriteInt16(writer, T.DataTypeSize)
	if err != nil {
		return
	}
	length += temp
	temp, err = WriteInt32(writer, T.TypeModifier)
	if err != nil {
		return
	}
	length += temp
	temp, err = WriteInt16(writer, T.FormatCode)
	if err != nil {
		return
	}
	length += temp
	_ = temp
	return
}

type FieldsRowDescription struct {
	Fields []FieldsRowDescriptionFields
}

func (T *FieldsRowDescription) Read(payloadLength int, reader io.Reader) (err error) {
	var FieldsLength int16
	FieldsLength, err = ReadInt16(reader)
	if err != nil {
		return
	}
	if FieldsLength == int16(-1) {
		FieldsLength = 0
	}
	T.Fields = make([]FieldsRowDescriptionFields, int(FieldsLength))
	for i := 0; i < int(FieldsLength); i++ {
		err = T.Fields[i].Read(payloadLength, reader)
		if err != nil {
			return
		}
	}
	return
}

func (T *FieldsRowDescription) Write(writer io.Writer) (length int, err error) {
	var temp int
	temp, err = WriteInt16(writer, int16(len(T.Fields)))
	if err != nil {
		return
	}
	length += temp
	for _, v := range T.Fields {
		temp, err = v.Write(writer)
		if err != nil {
			return
		}
		length += temp
	}
	_ = temp
	return
}

type RowDescription struct {
	Fields FieldsRowDescription
}

// Read reads all but the packet identifier
// WARNING: This packet DOES have an identifier. Call protocol.Read or trim the identifier first!
func (T *RowDescription) Read(reader io.Reader) (err error) {
	var length int32
	length, err = ReadInt32(reader)
	if err != nil {
		return
	}
	return T.Fields.Read(int(length-4), reader)
}

func (T *RowDescription) Write(writer io.Writer) (length int, err error) {
	// TODO replace with pool
	var buf bytes.Buffer
	length, err = T.Fields.Write(&buf)
	if err != nil {
		length = 0
		return
	}
	_, err = WriteByte(writer, byte('T'))
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

var _ Packet = (*RowDescription)(nil)
