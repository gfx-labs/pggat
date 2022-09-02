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

type BackendKeyData struct {
	fields FieldsBackendKeyData
}

type FieldsBindComplete struct {
}

func (T *FieldsBindComplete) Read(payloadLength int, reader io.Reader) (err error) {
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

type DataRow struct {
	fields FieldsDataRow
}

type FieldsEmptyQueryResponse struct {
}

func (T *FieldsEmptyQueryResponse) Read(payloadLength int, reader io.Reader) (err error) {
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

type NegotiateProtocolVersion struct {
	fields FieldsNegotiateProtocolVersion
}

type FieldsNoData struct {
}

func (T *FieldsNoData) Read(payloadLength int, reader io.Reader) (err error) {
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

type ParameterStatus struct {
	fields FieldsParameterStatus
}

type FieldsParseComplete struct {
}

func (T *FieldsParseComplete) Read(payloadLength int, reader io.Reader) (err error) {
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

type PortalSuspended struct {
	fields FieldsPortalSuspended
}

type FieldsReadForQuery struct {
}

func (T *FieldsReadForQuery) Read(payloadLength int, reader io.Reader) (err error) {
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

type RowDescription struct {
	fields FieldsRowDescription
}
