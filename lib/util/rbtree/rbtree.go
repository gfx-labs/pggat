package rbtree

type order interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64
}

// RBTree is a left-leaning red-black BST
type RBTree[K order, V any] struct {
	root *node[K, V]
	// pool of node to reuse memory
	pool []*node[K, V]
}

func (T *RBTree[K, V]) free(n *node[K, V]) {
	*n = node[K, V]{}
	T.pool = append(T.pool, n)
}

func (T *RBTree[K, V]) alloc() *node[K, V] {
	if len(T.pool) > 0 {
		v := T.pool[len(T.pool)-1]
		T.pool = T.pool[:len(T.pool)-1]
		return v
	}
	return new(node[K, V])
}

func (T *RBTree[K, V]) Get(key K) (V, bool) {
	n := T.root
	for n != nil {
		if key > n.key {
			n = n.right
		} else if key < n.key {
			n = n.left
		} else {
			return n.value, true
		}
	}
	return *new(V), false
}

func (T *RBTree[K, V]) Set(key K, value V) {
	T.root = T.put(T.root, key, value)
	T.root.color = black
}

func (T *RBTree[K, V]) put(n *node[K, V], key K, value V) *node[K, V] {
	if n == nil {
		n = T.alloc()
		n.key = key
		n.value = value
		n.color = red
		return n
	}

	if key > n.key {
		n.right = T.put(n.right, key, value)
	} else if key < n.key {
		n.left = T.put(n.left, key, value)
	} else {
		n.value = value
	}

	return T.balance(n)
}

func (T *RBTree[K, V]) Delete(key K) {
	if _, ok := T.Get(key); !ok {
		return
	}

	if T.root.left.getColor() == black && T.root.right.getColor() == black {
		T.root.color = red
	}

	T.root = T.delete(T.root, key)
	if T.root != nil {
		T.root.color = black
	}
}

func (T *RBTree[K, V]) delete(n *node[K, V], key K) *node[K, V] {
	if key < n.key {
		if n.left.getColor() == black && n.left.left.getColor() == black {
			n = T.moveRedLeft(n)
		}
		n.left = T.delete(n.left, key)
	} else {
		if n.left.getColor() == red {
			n = T.rotateRight(n)
		}
		if key == n.key && n.right == nil {
			T.free(n)
			return nil
		}
		if n.right.getColor() == black && n.right.left.getColor() == black {
			n = T.moveRedRight(n)
		}
		if key == n.key {
			x := T.min(n.right)
			n.key = x.key
			n.value = x.value
			n.right = T.deleteMin(n.right)
		} else {
			n.right = T.delete(n.right, key)
		}
	}
	return T.balance(n)
}

func (T *RBTree[K, V]) deleteMin(n *node[K, V]) *node[K, V] {
	if n.left == nil {
		T.free(n)
		return nil
	}

	if n.left.getColor() == black && n.left.left.getColor() == black {
		n = T.moveRedLeft(n)
	}

	n.left = T.deleteMin(n.left)
	return T.balance(n)
}

func (T *RBTree[K, V]) Min() (K, V, bool) {
	m := T.min(T.root)
	if m == nil {
		return *new(K), *new(V), false
	}
	return m.key, m.value, true
}

func (T *RBTree[K, V]) Max() (K, V, bool) {
	m := T.max(T.root)
	if m == nil {
		return *new(K), *new(V), false
	}
	return m.key, m.value, true
}

func (T *RBTree[K, V]) rotateRight(n *node[K, V]) *node[K, V] {
	if n == nil || n.left.getColor() == black {
		panic("assertion failed")
	}
	x := n.left
	n.left = x.right
	x.right = n
	x.color = n.color
	n.color = red
	return x
}

func (T *RBTree[K, V]) rotateLeft(n *node[K, V]) *node[K, V] {
	if n == nil || n.right.getColor() == black {
		panic("assertion failed")
	}
	x := n.right
	n.right = x.left
	x.left = n
	x.color = n.color
	n.color = red
	return x
}

func (T *RBTree[K, V]) flipColors(n *node[K, V]) {
	n.color = n.color.opposite()
	n.left.color = n.left.color.opposite()
	n.right.color = n.right.color.opposite()
}

func (T *RBTree[K, V]) moveRedLeft(n *node[K, V]) *node[K, V] {
	T.flipColors(n)
	if n.right.left.getColor() == red {
		n.right = T.rotateRight(n.right)
		n = T.rotateLeft(n)
		T.flipColors(n)
	}
	return n
}

func (T *RBTree[K, V]) moveRedRight(n *node[K, V]) *node[K, V] {
	T.flipColors(n)
	if n.left.left.getColor() == red {
		n = T.rotateRight(n)
		T.flipColors(n)
	}
	return n
}

func (T *RBTree[K, V]) balance(n *node[K, V]) *node[K, V] {
	if n.right.getColor() == red && n.left.getColor() == black {
		n = T.rotateLeft(n)
	}
	if n.left.getColor() == red && n.left.left.getColor() == red {
		n = T.rotateRight(n)
	}
	if n.left.getColor() == red && n.right.getColor() == red {
		T.flipColors(n)
	}

	return n
}

func (T *RBTree[K, V]) min(n *node[K, V]) *node[K, V] {
	if n == nil {
		return nil
	}
	if n.left == nil {
		return n
	}
	return T.min(n.left)
}

func (T *RBTree[K, V]) max(n *node[K, V]) *node[K, V] {
	if n == nil {
		return nil
	}
	if n.right == nil {
		return n
	}
	return T.max(n.right)
}

func (T *RBTree[K, V]) next(n *node[K, V], key K) *node[K, V] {
	if n == nil {
		return nil
	}
	if n.key == key {
		return T.min(n.right)
	}
	if n.key < key {
		return T.next(n.right, key)
	}

	// n.key > key
	next := T.next(n.left, key)
	if next == nil {
		return n
	}
	return next
}

func (T *RBTree[K, V]) Next(key K) (K, V, bool) {
	n := T.next(T.root, key)
	if n == nil {
		return *new(K), *new(V), false
	}
	return n.key, n.value, true
}

func (T *RBTree[K, V]) prev(n *node[K, V], key K) *node[K, V] {
	if n == nil {
		return nil
	}
	if n.key == key {
		return T.max(n.left)
	}
	if n.key > key {
		return T.prev(n.left, key)
	}

	// n.key < key
	prev := T.prev(n.right, key)
	if prev == nil {
		return n
	}
	return prev
}

func (T *RBTree[K, V]) Prev(key K) (K, V, bool) {
	n := T.prev(T.root, key)
	if n == nil {
		return *new(K), *new(V), false
	}
	return n.key, n.value, true
}

type color bool

const (
	black color = false
	red   color = true
)

func (T color) opposite() color {
	if T == black {
		return red
	}
	return black
}

type node[K order, V any] struct {
	key         K
	value       V
	left, right *node[K, V]
	color       color
}

func (T *node[K, V]) getColor() color {
	if T == nil {
		return black
	}
	return T.color
}
