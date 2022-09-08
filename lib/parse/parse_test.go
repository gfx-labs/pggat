package parse

import (
	"fmt"
	"github.com/auxten/postgresql-parser/pkg/sql/parser"
	"testing"
)

const testQuery = "SELECT * FROM Customers WHERE (CustomerName LIKE 'L%'\nOR CustomerName LIKE 'R%' /*OR CustomerName LIKE 'S%'\nOR CustomerName LIKE 'T%'*/ OR CustomerName LIKE 'W%')\nAND Country='USA'\nORDER BY CustomerName;\n"

func testParse() error {
	sql, err := Parse(testQuery)
	if err != nil {
		return err
	}
	if len(sql) != 1 {
		return fmt.Errorf("expected 13 commands, got %d", len(sql))
	}
	return nil
}

func TestParse(t *testing.T) {
	err := testParse()
	if err != nil {
		t.Error(t)
	}
}

func BenchmarkParse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		err := testParse()
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkOld(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(testQuery)
		if err != nil {
			b.Error(err)
		}
	}
}
