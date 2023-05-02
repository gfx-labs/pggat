package perror

type Extra byte

const (
	Detail           Extra = 'D'
	Hint             Extra = 'H'
	Position         Extra = 'P'
	InternalPosition Extra = 'p'
	InternalQuery    Extra = 'q'
	Where            Extra = 'W'
	SchemaName       Extra = 's'
	TableName        Extra = 't'
	ColumnName       Extra = 'c'
	DataTypeName     Extra = 'd'
	ConstraintName   Extra = 'n'
	File             Extra = 'F'
	Line             Extra = 'L'
	Routine          Extra = 'R'
)

type ExtraField struct {
	Type  Extra
	Value string
}
