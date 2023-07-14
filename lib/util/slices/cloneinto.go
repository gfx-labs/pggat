package slices

func CloneInto[T any](dst, src []T) []T {
	dst = Resize(dst, len(src))
	for i, v := range src {
		dst[i] = v
	}
	return dst
}
