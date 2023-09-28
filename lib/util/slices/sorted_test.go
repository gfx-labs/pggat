package slices

import (
	"log"
	"sort"
	"testing"
)

func TestSorted_Insert(t *testing.T) {
	sorter := func(v string) int {
		return len(v)
	}

	expected := []string{
		"test",
		"abc",
		"this is a long string",
		"gjkdfjgksg",
		"retre",
		"abd",
		"def",
		"ttierotiretiiret34t43t34534",
	}

	var x Sorted[string]
	for _, v := range expected {
		x = x.Insert(v, sorter)
	}

	if !sort.SliceIsSorted(x, func(i, j int) bool {
		return sorter(x[i]) < sorter(x[j])
	}) {
		t.Errorf("slice isn't sorted: %#v", x)
	}
}

func TestSorted_Update(t *testing.T) {
	values := map[string]int{
		"abc":               43,
		"def":               32,
		"cool":              594390069,
		"amazing":           -432,
		"i hope this works": 32,
	}

	sorter := func(v string) int {
		return values[v]
	}

	var x Sorted[string]
	for v := range values {
		x = x.Insert(v, sorter)
	}

	if !sort.SliceIsSorted(x, func(i, j int) bool {
		return sorter(x[i]) < sorter(x[j])
	}) {
		t.Errorf("slice isn't sorted: %#v", x)
	}

	log.Printf("%#v", x)

	values["cool"] = -10
	x.Update(Index(x, "cool"), sorter)
	values["amazing"] = 543543
	x.Update(Index(x, "amazing"), sorter)
	x.Update(Index(x, "abc"), sorter)
	values["i hope this works"] = 44
	x.Update(Index(x, "i hope this works"), sorter)
	values["abc"] = 31
	x.Update(Index(x, "abc"), sorter)

	if !sort.SliceIsSorted(x, func(i, j int) bool {
		return sorter(x[i]) < sorter(x[j])
	}) {
		t.Errorf("slice isn't sorted: %#v", x)
	}
}
