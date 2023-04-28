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

func TestRBTree_Stress(t *testing.T) {
	const n = 1000000

	tree := new(RBTree[int, int])

	for i := 0; i < n; i++ {
		k := rand.Int()
		v := rand.Int()
		tree.Set(k, v)
		tree.Set(rand.Int(), rand.Int())
		tree.Delete(k)
	}
}
