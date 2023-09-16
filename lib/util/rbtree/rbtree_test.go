package rbtree

import (
	"math/rand"
	"testing"
)

func assertSome[K order, V comparable](t *testing.T, tree *RBTree[K, V], key K, value V) {
	v, ok := tree.Get(key)
	if !ok {
		t.Error("expected tree to have key", key)
		return
	}
	if v != value {
		t.Error("expected", value, "but got", v)
		return
	}
}

func assertNone[K order, V comparable](t *testing.T, tree *RBTree[K, V], key K) {
	v, ok := tree.Get(key)
	if ok {
		t.Error("expected no value but got", v)
		return
	}
}

func assertMin[K order, V comparable](t *testing.T, tree *RBTree[K, V], key K, value V) {
	k, v, ok := tree.Min()
	if !ok {
		t.Error("expected tree to have values")
	}
	if k != key || v != value {
		t.Error("expected key, value to be", key, value, "but got", k, v)
	}
}

func assertMax[K order, V comparable](t *testing.T, tree *RBTree[K, V], key K, value V) {
	k, v, ok := tree.Max()
	if !ok {
		t.Error("expected tree to have values")
	}
	if k != key || v != value {
		t.Error("expected key, value to be", key, value, "but got", k, v)
	}
}

func assertNextSome[K order, V comparable](t *testing.T, tree *RBTree[K, V], after K, key K, value V) {
	k, v, ok := tree.Next(after)
	if !ok {
		t.Error("expected tree to have another value")
	}
	if k != key || v != value {
		t.Error("expected key, value to be", key, value, "but got", k, v)
	}
}

func assertNextNone[K order, V comparable](t *testing.T, tree *RBTree[K, V], after K) {
	k, v, ok := tree.Next(after)
	if ok {
		t.Error("expected tree to have no more values but got", k, v)
	}
}

func assertPrevSome[K order, V comparable](t *testing.T, tree *RBTree[K, V], before K, key K, value V) {
	k, v, ok := tree.Prev(before)
	if !ok {
		t.Error("expected tree to have another value")
	}
	if k != key || v != value {
		t.Error("expected key, value to be", key, value, "but got", k, v)
	}
}

func assertPrevNone[K order, V comparable](t *testing.T, tree *RBTree[K, V], before K) {
	k, v, ok := tree.Prev(before)
	if ok {
		t.Error("expected tree to have no more values but got", k, v)
	}
}

func TestRBTree_Insert(t *testing.T) {
	tree := new(RBTree[int, int])
	tree.Set(1, 2)
	tree.Set(3, 4)
	tree.Set(5, 6)
	assertSome(t, tree, 1, 2)
	assertSome(t, tree, 3, 4)
	assertSome(t, tree, 5, 6)
}

func TestRBTree_Delete(t *testing.T) {
	tree := new(RBTree[int, int])
	tree.Set(1, 2)
	tree.Set(3, 4)
	tree.Set(5, 6)
	tree.Delete(3)
	tree.Delete(2)
	assertSome(t, tree, 1, 2)
	assertNone(t, tree, 3)
	assertSome(t, tree, 5, 6)
}

func TestRBTree_Min(t *testing.T) {
	tree := new(RBTree[int, int])
	tree.Set(1, 2)
	tree.Set(3, 4)
	tree.Set(5, 6)
	assertMin(t, tree, 1, 2)
	tree.Delete(3)
	tree.Delete(1)
	assertMin(t, tree, 5, 6)
}

func TestRBTree_Iter(t *testing.T) {
	tree := new(RBTree[int, int])
	tree.Set(1, 2)
	tree.Set(5, 6)
	tree.Set(3, 4)
	assertMin(t, tree, 1, 2)
	assertNextSome(t, tree, 1, 3, 4)
	assertNextSome(t, tree, 3, 5, 6)
	assertNextNone(t, tree, 5)
}

func TestRBTree_IterRev(t *testing.T) {
	tree := new(RBTree[int, int])
	tree.Set(1, 2)
	tree.Set(5, 6)
	tree.Set(3, 4)
	assertMax(t, tree, 5, 6)
	assertPrevSome(t, tree, 5, 3, 4)
	assertPrevSome(t, tree, 3, 1, 2)
	assertPrevNone(t, tree, 1)
}

func TestRBTree_Stress(t *testing.T) {
	const n = 1000000

	tree := new(RBTree[int, int])

	existing := make(map[int]struct{})

	for i := 0; i < n; i++ {
		k1 := rand.Int()
		v1 := rand.Int()
		k2 := rand.Int()
		v2 := rand.Int()
		tree.Set(k1, v1)
		tree.Set(k2, v2)
		tree.Delete(k1)
		delete(existing, k1)
		existing[k2] = struct{}{}
	}

	// make sure all values we expect are there
	for k := range existing {
		if _, ok := tree.Get(k); !ok {
			t.Error("expected value to exist")
		}
	}

	// iterate through to make sure there aren't any extra values
	prev := 0
	for k, v, ok := tree.Min(); ok; k, v, ok = tree.Next(k) {
		if k < prev {
			t.Error("expected tree to be correctly ordered")
		}
		prev = k
		if _, ok = existing[k]; !ok {
			t.Error("unexpected value in tree")
		}
		_ = v
	}
}

func TestRBTree_Min2(t *testing.T) {
	tree := new(RBTree[int, struct{}])

	const n = 100000

	for i := 0; i < n; i++ {
		tree.Set(i, struct{}{})
	}

	for i := 0; i < n; i++ {
		k, _, ok := tree.Min()
		if !ok {
			t.Error("expected tree to have min value")
			return
		}

		if k != i {
			t.Error("out of order")
			return
		}

		tree.Delete(k)
	}

	_, _, ok := tree.Min()
	if ok {
		t.Error("expected no more values")
	}
}
