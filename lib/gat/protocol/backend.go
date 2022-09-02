package protocol

import "io"

// codegen: modify for debug only

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
	fields FieldsAuthentication
}

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
	fields FieldsBackendKeyData
}

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
	fields FieldsBindComplete
}

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
	fields FieldsCloseComplete
}

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
	fields FieldsCommandComplete
}

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
	fields FieldsCopyBothResponse
}

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
	fields FieldsCopyInResponse
}

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
	fields FieldsCopyOutResponse
}

type FieldsDataRowColumns struct {
	Bytes []int8
}

func (T *FieldsDataRowColumns) Read(payloadLength int, reader io.Reader) (err error) {
	var BytesLength int32
	BytesLength, err = ReadInt32(reader)
	if err != nil {
		return
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
	fields FieldsDataRow
}

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
	fields FieldsEmptyQueryResponse
}

type FieldsErrorResponse struct {
	Code  byte
	Value string
}

func (T *FieldsErrorResponse) Read(payloadLength int, reader io.Reader) (err error) {
	T.Code, err = ReadByte(reader)
	if err != nil {
		return
	}
	T.Value, err = ReadString(reader)
	if err != nil {
		return
	}
	return
}

func (T *FieldsErrorResponse) Write(writer io.Writer) (length int, err error) {
	var temp int
	temp, err = WriteByte(writer, T.Code)
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

type ErrorResponse struct {
	fields FieldsErrorResponse
}

type FieldsFunctionCallResponse struct {
	Result []byte
}

func (T *FieldsFunctionCallResponse) Read(payloadLength int, reader io.Reader) (err error) {
	var ResultLength int32
	ResultLength, err = ReadInt32(reader)
	if err != nil {
		return
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
	fields FieldsFunctionCallResponse
}

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
	fields FieldsNegotiateProtocolVersion
}

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
	fields FieldsNoData
}

type FieldsNoticeResponse struct {
	Type  byte
	Value string
}

func (T *FieldsNoticeResponse) Read(payloadLength int, reader io.Reader) (err error) {
	T.Type, err = ReadByte(reader)
	if err != nil {
		return
	}
	if T.Type != 0 {
		T.Value, err = ReadString(reader)
		if err != nil {
			return
		}
	}
	return
}

func (T *FieldsNoticeResponse) Write(writer io.Writer) (length int, err error) {
	var temp int
	temp, err = WriteByte(writer, T.Type)
	if err != nil {
		return
	}
	length += temp
	if T.Type != 0 {
		temp, err = WriteString(writer, T.Value)
		if err != nil {
			return
		}
		length += temp
	}
	_ = temp
	return
}

type NoticeResponse struct {
	fields FieldsNoticeResponse
}

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
	fields FieldsNotificationResponse
}

type FieldsParameterDescription struct {
	Parameters []int32
}

func (T *FieldsParameterDescription) Read(payloadLength int, reader io.Reader) (err error) {
	var ParametersLength int16
	ParametersLength, err = ReadInt16(reader)
	if err != nil {
		return
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
	fields FieldsParameterDescription
}

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
	fields FieldsParameterStatus
}

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
	fields FieldsParseComplete
}

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
	fields FieldsPortalSuspended
}

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
	fields FieldsReadForQuery
}

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
	fields FieldsRowDescription
}
