package iter

import (
	"strconv"
	"testing"
)

func expect[T comparable](t *testing.T, iter Iter[T], value T) {
	v, ok := iter()
	if !ok {
		t.Error("expected iter to have a value, but reached end")
		return
	}
	if v != value {
		t.Error("expected next value to be", value, "( got:", v, ")")
	}
}

func end[T any](t *testing.T, iter Iter[T]) {
	v, ok := iter()
	if ok {
		t.Error("expected iter to end but got value", v)
	}
}

func TestSlice(t *testing.T) {
	slice := []int{1, 2, 3}
	iter := Slice(slice)

	expect(t, iter, 1)
	expect(t, iter, 2)
	expect(t, iter, 3)
	end(t, iter)
}

func TestMap(t *testing.T) {
	slice := []int{1, 2, 3}
	iter := Slice(slice)
	iter2 := Map(iter, strconv.Itoa)

	expect(t, iter2, "1")
	expect(t, iter2, "2")
	expect(t, iter2, "3")
	end(t, iter2)
}

func TestFilter(t *testing.T) {
	slice := []int{1, 2, 3}
	iter := Slice(slice)
	iter2 := Filter(iter, func(i int) bool {
		return i%2 == 0
	})

	expect(t, iter2, 2)
	end(t, iter2)
}

func TestChain(t *testing.T) {
	slice := []int{1, 2, 3}
	slice2 := []int{4, 5}
	slice3 := []int{6, 7}
	iter := Slice(slice)
	iter2 := Slice(slice2)
	iter3 := Slice(slice3)
	iter4 := Chain(iter, iter2, iter3)

	expect(t, iter4, 1)
	expect(t, iter4, 2)
	expect(t, iter4, 3)
	expect(t, iter4, 4)
	expect(t, iter4, 5)
	expect(t, iter4, 6)
	expect(t, iter4, 7)
	end(t, iter4)
}

func TestForEach(t *testing.T) {
	slice := []int{1, 2, 3}
	iter := Slice(slice)

	i := 0
	ForEach(iter, func(v int) {
		if slice[i] != v {
			t.Error("expected index", i, "to be", slice[i], "but got", v)
		}
		i++
	})
	end(t, iter)
}
