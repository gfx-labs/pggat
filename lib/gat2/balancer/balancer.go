package balancer

// Balancer is the frontend that listens for clients and accepts them, routing them to the correct pool
type Balancer interface {
	Run() error
}
