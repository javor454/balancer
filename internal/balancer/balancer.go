package balancer

import "fmt"

type Balancer struct {
	capacity int
}

func NewBalancer(config *Config) (*Balancer, error) {
	switch config.Strategy {
	case SingleClient:
		fmt.Printf("Using strategy: %s\n", config.Strategy)

		return NewSingleClientBalancer(config.Capacity)
	default:
		return nil, fmt.Errorf("invalid strategy %q", config.Strategy)
	}
}

func NewSingleClientBalancer(capacity int) (*Balancer, error) {
	return &Balancer{
		capacity: capacity,
	}, nil
}

func (b *Balancer) Balance() error {
	fmt.Printf("Balancing capacity: %d\n", b.capacity)
	return nil
}
