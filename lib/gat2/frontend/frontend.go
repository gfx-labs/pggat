package frontend

// Frontend handles
type Frontend interface {
	// Run the frontend, awaiting new conns and
	Run() error
}
