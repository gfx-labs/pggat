package gat

import (
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
	"strconv"
)

// error codes format. See https://www.postgresql.org/docs/current/protocol-error-fields.html

type Severity string

const (
	Error  Severity = "ERROR"
	Fatal           = "FATAL"
	Panic           = "PANIC"
	Warn            = "WARNING"
	Notice          = "NOTICE"
	Debug           = "DEBUG"
	Info            = "INFO"
	Log             = "LOG"
)

type Code string

const (
	SuccessfulCompletion                                    Code = "00000"
	Warning                                                      = "01000"
	DynamicResultSetsReturned                                    = "0100C"
	ImplicitZeroBitPadding                                       = "01008"
	NullValueEliminatedInSetFunction                             = "01003"
	PrivilegeNotGranted                                          = "01007"
	PrivilegeNotRevoked                                          = "01006"
	WarnStringDataRightTruncation                                = "01004"
	DeprecatedFeature                                            = "01P01"
	NoData                                                       = "02000"
	NoAdditionalDynamicResultSetsReturned                        = "02001"
	SqlStatementNotYetComplete                                   = "03000"
	ConnectionException                                          = "08000"
	ConnectionDoesNotExist                                       = "08003"
	ConnectionFailure                                            = "08006"
	SqlclientUnableToEstablishSqlconnection                      = "08001"
	SqlserverRejectedEstablishmentOfSqlconnection                = "08004"
	TransactionResolutionUnknown                                 = "08007"
	ProtocolViolation                                            = "08P01"
	TriggeredActionException                                     = "09000"
	FeatureNotSupported                                          = "0A000"
	InvalidTransactionInitiation                                 = "0B000"
	LocatorException                                             = "0F000"
	InvalidLocatorSpecification                                  = "0F001"
	InvalidGrantor                                               = "0L000"
	InvalidGrantOperation                                        = "0LP01"
	InvalidRoleSpecification                                     = "0P000"
	DiagnosticsException                                         = "0Z000"
	StackedDiagnosticsAccessedWithoutActiveHandler               = "0Z002"
	CaseNotFound                                                 = "20000"
	CardinalityViolation                                         = "21000"
	DataException                                                = "22000"
	ArraySubscriptError                                          = "2202E"
	CharacterNotInRepertoire                                     = "22021"
	DatetimeFieldOverflow                                        = "22008"
	DivisionByZero                                               = "22012"
	ErrorInAssignment                                            = "22005"
	EscapeCharacterConflict                                      = "2200B"
	IndicatorOverflow                                            = "22022"
	IntervalFieldOverflow                                        = "22015"
	InvalidArgumentForLogarithm                                  = "2201E"
	InvalidArgumentForNtileFunction                              = "22014"
	InvalidArgumentForNthValueFunction                           = "22016"
	InvalidArgumentForPowerFunction                              = "2201F"
	InvalidArgumentForWidthBucketFunction                        = "2201G"
	InvalidCharacterValueForCast                                 = "22018"
	InvalidDatetimeFormat                                        = "22007"
	InvalidEscapeCharacter                                       = "22019"
	InvalidEscapeOctet                                           = "2200D"
	InvalidEscapeSequence                                        = "22025"
	NonstandardUseOfEscapeCharacter                              = "22P06"
	InvalidIndicatorParameterValue                               = "22010"
	InvalidParameterValue                                        = "22023"
	InvalidPrecedingOrFollowingSize                              = "22013"
	InvalidRegularExpression                                     = "2201B"
	InvalidRowCountInLimitClause                                 = "2201W"
	InvalidRowCountInResultOffsetClause                          = "2201X"
	InvalidTablesampleArgument                                   = "2202H"
	InvalidTablesampleRepeat                                     = "2202G"
	InvalidTimeZoneDisplacementValue                             = "22009"
	InvalidUseOfEscapeCharacter                                  = "2200C"
	MostSpecificTypeMismatch                                     = "2200G"
	DataExceptionNullValueNotAllowed                             = "22004"
	NullValueNoIndicatorParameter                                = "22002"
	NumericValueOutOfRange                                       = "22003"
	SequenceGeneratorLimitExceeded                               = "2200H"
	StringDataLengthMismatch                                     = "22026"
	DataExceptionStringDataRightTruncation                       = "22001"
	SubstringError                                               = "22011"
	TrimError                                                    = "22027"
	UnterminatedCString                                          = "22024"
	ZeroLengthCharacterString                                    = "2200F"
	FloatingPointException                                       = "22P01"
	InvalidTextRepresentation                                    = "22P02"
	InvalidBinaryRepresentation                                  = "22P03"
	BadCopyFileFormat                                            = "22P04"
	UntranslatableCharacter                                      = "22P05"
	NotAnXmlDocument                                             = "2200L"
	InvalidXmlDocument                                           = "2200M"
	InvalidXmlContent                                            = "2200N"
	InvalidXmlComment                                            = "2200S"
	InvalidXmlProcessingInstruction                              = "2200T"
	DuplicateJsonObjectKeyValue                                  = "22030"
	InvalidArgumentForSqlJsonDatetimeFunction                    = "22031"
	InvalidJsonText                                              = "22032"
	InvalidSqlJsonSubscript                                      = "22033"
	MoreThanOneSqlJsonItem                                       = "22034"
	NoSqlJsonItem                                                = "22035"
	NonNumericSqlJsonItem                                        = "22036"
	NonUniqueKeysInAJsonObject                                   = "22037"
	SingletonSqlJsonItemRequired                                 = "22038"
	SqlJsonArrayNotFound                                         = "22039"
	SqlJsonMemberNotFound                                        = "2203A"
	SqlJsonNumberNotFound                                        = "2203B"
	SqlJsonObjectNotFound                                        = "2203C"
	TooManyJsonArrayElements                                     = "2203D"
	TooManyJsonObjectMembers                                     = "2203E"
	SqlJsonScalarRequired                                        = "2203F"
	IntegrityConstraintViolation                                 = "23000"
	RestrictViolation                                            = "23001"
	NotNullViolation                                             = "23502"
	ForeignKeyViolation                                          = "23503"
	UniqueViolation                                              = "23505"
	CheckViolation                                               = "23514"
	ExclusionViolation                                           = "23P01"
	InvalidCursorState                                           = "24000"
	InvalidTransactionState                                      = "25000"
	ActiveSqlTransaction                                         = "25001"
	BranchTransactionAlreadyActive                               = "25002"
	HeldCursorRequiresSameIsolationLevel                         = "25008"
	InappropriateAccessModeForBranchTransaction                  = "25003"
	InappropriateIsolationLevelForBranchTransaction              = "25004"
	NoActiveSqlTransactionForBranchTransaction                   = "25005"
	ReadOnlySqlTransaction                                       = "25006"
	SchemaAndDataStatementMixingNotSupported                     = "25007"
	NoActiveSqlTransaction                                       = "25P01"
	InFailedSqlTransaction                                       = "25P02"
	IdleInTransactionSessionTimeout                              = "25P03"
	InvalidSqlStatementName                                      = "26000"
	TriggeredDataChangeViolation                                 = "27000"
	InvalidAuthorizationSpecification                            = "28000"
	InvalidPassword                                              = "28P01"
	DependentPrivilegeDescriptorsStillExist                      = "2B000"
	DependentObjectsStillExist                                   = "2BP01"
	InvalidTransactionTermination                                = "2D000"
	SqlRoutineException                                          = "2F000"
	FunctionExecutedNoReturnStatement                            = "2F005"
	SQLRoutineExceptionModifyingSqlDataNotPermitted              = "2F002"
	SQLRoutineExceptionProhibitedSqlStatementAttempted           = "2F003"
	SQLRoutineExceptionReadingSqlDataNotPermitted                = "2F004"
	InvalidCursorName                                            = "34000"
	ExternalRoutineException                                     = "38000"
	ContainingSqlNotPermitted                                    = "38001"
	ExternalRoutineExceptionModifyingSqlDataNotPermitted         = "38002"
	ExternalRoutineExceptionProhibitedSqlStatementAttempted      = "38003"
	ExternalRoutineExceptionReadingSqlDataNotPermitted           = "38004"
	ExternalRoutineInvocationException                           = "39000"
	InvalidSqlstateReturned                                      = "39001"
	ERIENullValueNotAllowed                                      = "39004"
	TriggerProtocolViolated                                      = "39P01"
	SrfProtocolViolated                                          = "39P02"
	EventTriggerProtocolViolated                                 = "39P03"
	SavepointException                                           = "3B000"
	InvalidSavepointSpecification                                = "3B001"
	InvalidCatalogName                                           = "3D000"
	InvalidSchemaName                                            = "3F000"
	TransactionRollback                                          = "40000"
	TransactionIntegrityConstraintViolation                      = "40002"
	SerializationFailure                                         = "40001"
	StatementCompletionUnknown                                   = "40003"
	DeadlockDetected                                             = "40P01"
	SyntaxErrorOrAccessRuleViolation                             = "42000"
	SyntaxError                                                  = "42601"
	InsufficientPrivilege                                        = "42501"
	CannotCoerce                                                 = "42846"
	GroupingError                                                = "42803"
	WindowingError                                               = "42P20"
	InvalidRecursion                                             = "42P19"
	InvalidForeignKey                                            = "42830"
	InvalidName                                                  = "42602"
	NameTooLong                                                  = "42622"
	ReservedName                                                 = "42939"
	DatatypeMismatch                                             = "42804"
	IndeterminateDatatype                                        = "42P18"
	CollationMismatch                                            = "42P21"
	IndeterminateCollation                                       = "42P22"
	WrongObjectType                                              = "42809"
	GeneratedAlways                                              = "428C9"
	UndefinedColumn                                              = "42703"
	UndefinedFunction                                            = "42883"
	UndefinedTable                                               = "42P01"
	UndefinedParameter                                           = "42P02"
	UndefinedObject                                              = "42704"
	DuplicateColumn                                              = "42701"
	DuplicateCursor                                              = "42P03"
	DuplicateDatabase                                            = "42P04"
	DuplicateFunction                                            = "42723"
	DuplicatePreparedStatement                                   = "42P05"
	DuplicateSchema                                              = "42P06"
	DuplicateTable                                               = "42P07"
	DuplicateAlias                                               = "42712"
	DuplicateObject                                              = "42710"
	AmbiguousColumn                                              = "42702"
	AmbiguousFunction                                            = "42725"
	AmbiguousParameter                                           = "42P08"
	AmbiguousAlias                                               = "42P09"
	InvalidColumnReference                                       = "42P10"
	InvalidColumnDefinition                                      = "42611"
	InvalidCursorDefinition                                      = "42P11"
	InvalidDatabaseDefinition                                    = "42P12"
	InvalidFunctionDefinition                                    = "42P13"
	InvalidPreparedStatementDefinition                           = "42P14"
	InvalidSchemaDefinition                                      = "42P15"
	InvalidTableDefinition                                       = "42P16"
	InvalidObjectDefinition                                      = "42P17"
	WithCheckOptionViolation                                     = "44000"
	InsufficientResources                                        = "53000"
	DiskFull                                                     = "53100"
	OutOfMemory                                                  = "53200"
	TooManyConnections                                           = "53300"
	ConfigurationLimitExceeded                                   = "53400"
	ProgramLimitExceeded                                         = "54000"
	StatementTooComplex                                          = "54001"
	TooManyColumns                                               = "54011"
	TooManyArguments                                             = "54023"
	ObjectNotInPrerequisiteState                                 = "55000"
	ObjectInUse                                                  = "55006"
	CantChangeRuntimeParam                                       = "55P02"
	LockNotAvailable                                             = "55P03"
	UnsafeNewEnumValueUsage                                      = "55P04"
	OperatorIntervention                                         = "57000"
	QueryCanceled                                                = "57014"
	AdminShutdown                                                = "57P01"
	CrashShutdown                                                = "57P02"
	CannotConnectNow                                             = "57P03"
	DatabaseDropped                                              = "57P04"
	IdleSessionTimeout                                           = "57P05"
	SystemError                                                  = "58000"
	IoError                                                      = "58030"
	UndefinedFile                                                = "58P01"
	DuplicateFile                                                = "58P02"
	SnapshotTooOld                                               = "72000"
	ConfigFileError                                              = "F0000"
	LockFileExists                                               = "F0001"
	FdwError                                                     = "HV000"
	FdwColumnNameNotFound                                        = "HV005"
	FdwDynamicParameterValueNeeded                               = "HV002"
	FdwFunctionSequenceError                                     = "HV010"
	FdwInconsistentDescriptorInformation                         = "HV021"
	FdwInvalidAttributeValue                                     = "HV024"
	FdwInvalidColumnName                                         = "HV007"
	FdwInvalidColumnNumber                                       = "HV008"
	FdwInvalidDataType                                           = "HV004"
	FdwInvalidDataTypeDescriptors                                = "HV006"
	FdwInvalidDescriptorFieldIdentifier                          = "HV091"
	FdwInvalidHandle                                             = "HV00B"
	FdwInvalidOptionIndex                                        = "HV00C"
	FdwInvalidOptionName                                         = "HV00D"
	FdwInvalidStringLengthOrBufferLength                         = "HV090"
	FdwInvalidStringFormat                                       = "HV00A"
	FdwInvalidUseOfNullPointer                                   = "HV009"
	FdwTooManyHandles                                            = "HV014"
	FdwOutOfMemory                                               = "HV001"
	FdwNoSchemas                                                 = "HV00P"
	FdwOptionNameNotFound                                        = "HV00J"
	FdwReplyHandle                                               = "HV00K"
	FdwSchemaNotFound                                            = "HV00Q"
	FdwTableNotFound                                             = "HV00R"
	FdwUnableToCreateExecution                                   = "HV00L"
	FdwUnableToCreateReply                                       = "HV00M"
	FdwUnableToEstablishConnection                               = "HV00N"
	PlpgsqlError                                                 = "P0000"
	RaiseException                                               = "P0001"
	NoDataFound                                                  = "P0002"
	TooManyRows                                                  = "P0003"
	AssertFailure                                                = "P0004"
	InternalError                                                = "XX000"
	DataCorrupted                                                = "XX001"
	IndexCorrupted                                               = "XX002"
)

type PostgresError struct {
	Severity         Severity
	Code             Code
	Message          string
	Detail           string
	Hint             string
	Position         int
	InternalPosition int
	InternalQuery    string
	Where            string
	Schema           string
	Table            string
	Column           string
	DataType         string
	Constraint       string
	File             string
	Line             int
	Routine          string
}

func (E *PostgresError) Packet() *protocol.ErrorResponse {
	var fields []protocol.FieldsErrorResponseResponses
	fields = append(fields, protocol.FieldsErrorResponseResponses{
		Code:  byte('S'),
		Value: string(E.Severity),
	}, protocol.FieldsErrorResponseResponses{
		Code:  byte('C'),
		Value: string(E.Code),
	}, protocol.FieldsErrorResponseResponses{
		Code:  byte('M'),
		Value: E.Message,
	})
	if E.Detail != "" {
		fields = append(fields, protocol.FieldsErrorResponseResponses{
			Code:  byte('D'),
			Value: E.Detail,
		})
	}
	if E.Hint != "" {
		fields = append(fields, protocol.FieldsErrorResponseResponses{
			Code:  byte('H'),
			Value: E.Hint,
		})
	}
	if E.Position != 0 {
		fields = append(fields, protocol.FieldsErrorResponseResponses{
			Code:  byte('P'),
			Value: strconv.Itoa(E.Position),
		})
	}
	if E.InternalPosition != 0 {
		fields = append(fields, protocol.FieldsErrorResponseResponses{
			Code:  byte('p'),
			Value: strconv.Itoa(E.InternalPosition),
		})
	}
	if E.InternalQuery != "" {
		fields = append(fields, protocol.FieldsErrorResponseResponses{
			Code:  byte('q'),
			Value: E.InternalQuery,
		})
	}
	if E.Where != "" {
		fields = append(fields, protocol.FieldsErrorResponseResponses{
			Code:  byte('W'),
			Value: E.Where,
		})
	}
	if E.Schema != "" {
		fields = append(fields, protocol.FieldsErrorResponseResponses{
			Code:  byte('s'),
			Value: E.Schema,
		})
	}
	if E.Table != "" {
		fields = append(fields, protocol.FieldsErrorResponseResponses{
			Code:  byte('t'),
			Value: E.Table,
		})
	}
	if E.Column != "" {
		fields = append(fields, protocol.FieldsErrorResponseResponses{
			Code:  byte('c'),
			Value: E.Column,
		})
	}
	if E.DataType != "" {
		fields = append(fields, protocol.FieldsErrorResponseResponses{
			Code:  byte('d'),
			Value: E.DataType,
		})
	}
	if E.Constraint != "" {
		fields = append(fields, protocol.FieldsErrorResponseResponses{
			Code:  byte('n'),
			Value: E.Constraint,
		})
	}
	if E.File != "" {
		fields = append(fields, protocol.FieldsErrorResponseResponses{
			Code:  byte('F'),
			Value: E.File,
		})
	}
	if E.Line != 0 {
		fields = append(fields, protocol.FieldsErrorResponseResponses{
			Code:  byte('L'),
			Value: strconv.Itoa(E.Line),
		})
	}
	if E.Routine != "" {
		fields = append(fields, protocol.FieldsErrorResponseResponses{
			Code:  byte('R'),
			Value: E.Routine,
		})
	}
	fields = append(fields, protocol.FieldsErrorResponseResponses{})
	pkt := new(protocol.ErrorResponse)
	pkt.Fields.Responses = fields
	return pkt
}

func (E *PostgresError) Error() string {
	return E.Message
}
