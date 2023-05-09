package router

type Router interface {
	NewHandler(write bool) Handler
	NewSource() Source
}
