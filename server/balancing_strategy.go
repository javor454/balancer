package server

import (
	"context"
	"net/http"
	"time"
)

// BalancingStrategy defines the common interface for balancing strategies
type BalancingStrategy interface {
	// NextServer selects the next available backend server
	// Returns the handler and any selection error
	NextServer(ctx context.Context) (http.Handler, error)

	// AcquireCapacity attempts to acquire system capacity
	// Returns error if no capacity available within timeout
	AcquireCapacity(ctx context.Context, timeout time.Duration) error

	// ReleaseCapacity releases acquired capacity back to the system
	ReleaseCapacity()

	// GetMaxCapacity returns the maximum system capacity
	GetMaxCapacity() int

	// GetAvailableCapacity returns currently available capacity
	GetAvailableCapacity() int
}
