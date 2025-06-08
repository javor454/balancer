package server

import (
	"context"
	"log"
	"net/http"
	"time"
)

// RoundRobinStrategy implements round-robin backend selection with global capacity control
type RoundRobinStrategy struct {
	servers                []*server
	currentServerIndex     int
	maxCapacity            int
	capacity               chan struct{}
	acquireCapacityTimeout time.Duration
}

// NewRoundRobinStrategy creates a new round-robin balancing strategy
func NewRoundRobinStrategy(servers []*server, maxCapacity int, acquireCapacityTimeout time.Duration) *RoundRobinStrategy {
	return &RoundRobinStrategy{
		servers:                servers,
		currentServerIndex:     0,
		maxCapacity:            maxCapacity,
		capacity:               make(chan struct{}, maxCapacity),
		acquireCapacityTimeout: acquireCapacityTimeout,
	}
}

// NextServer iterate through servers maximum of 2 times to find available server in round-robin fashion
func (s *RoundRobinStrategy) NextServer(ctx context.Context) (http.Handler, error) {
	if err := s.AcquireCapacity(ctx, s.acquireCapacityTimeout); err != nil {
		return nil, err
	}

	log.Printf("Looking for a healthy server...")
	sumBackends := len(s.servers)

	if sumBackends == 0 {
		s.ReleaseCapacity()
		return nil, ErrNoServers
	}

	for range sumBackends * 2 {
		server := s.servers[s.currentServerIndex]
		s.currentServerIndex = (s.currentServerIndex + 1) % sumBackends

		if server.IsAlive() {
			log.Printf("Using server %s", server.url.String())
			return server.reverseProxy, nil
		}
	}

	s.ReleaseCapacity()
	return nil, ErrNoHealthyServers
}

// AcquireCapacity attempts to acquire a slot in the capacity channel
func (s *RoundRobinStrategy) AcquireCapacity(ctx context.Context, timeout time.Duration) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case s.capacity <- struct{}{}: // Try to acquire a token
		return nil
	case <-timeoutCtx.Done():
		return ErrNoCapacity // Timeout without acquiring a token
	}
}

// ReleaseCapacity releases a slot in the capacity channel
func (s *RoundRobinStrategy) ReleaseCapacity() {
	select {
	case <-s.capacity:
	default: // prevents panics if ReleaseCapacity is called more times than AcquireCapacity
	}
}

// GetMaxCapacity returns the maximum server capacity
func (s *RoundRobinStrategy) GetMaxCapacity() int {
	return s.maxCapacity
}

// GetAvailableCapacity returns the available server capacity
func (s *RoundRobinStrategy) GetAvailableCapacity() int {
	return s.maxCapacity - len(s.capacity)
}
