package gatutil

import (
	"fmt"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
)

type TableHeaderColumn struct {
	Name string
	Type Type
}

func (T *TableHeaderColumn) RowDescription() protocol.FieldsRowDescriptionFields {
	return protocol.FieldsRowDescriptionFields{
		Name:         T.Name,
		DataType:     T.Type.OID(),
		DataTypeSize: T.Type.Len(),
		FormatCode:   0,
	}
}

type TableHeader struct {
	Columns []TableHeaderColumn
}

func (T *TableHeader) RowDescription() (pkt protocol.RowDescription) {
	for _, col := range T.Columns {
		pkt.Fields.Fields = append(pkt.Fields.Fields, col.RowDescription())
	}
	return
}

type TableRow struct {
	Columns []any
}

func (T *TableRow) DataRow() (pkt protocol.DataRow) {
	for _, col := range T.Columns {
		pkt.Fields.Columns = append(pkt.Fields.Columns, protocol.FieldsDataRowColumns{
			Bytes: []byte(fmt.Sprintf("%v\x00", col)),
		})
	}
	return
}

type Table struct {
	Header TableHeader
	Rows   []TableRow
}

func (T *Table) Send(client gat.Client) error {
	rowDescription := T.Header.RowDescription()
	err := client.Send(&rowDescription)
	if err != nil {
		return err
	}
	for _, row := range T.Rows {
		dataRow := row.DataRow()
		err = client.Send(&dataRow)
		if err != nil {
			return err
		}
	}
	return nil
}
