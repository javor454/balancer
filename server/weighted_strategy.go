package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/javor454/balancer/auth"
)

// WeightedStrategy implements weighted backend selection with per-client capacity quotas
type WeightedStrategy struct {
	servers                []*server
	currentServerIndex     int
	maxCapacity            int
	acquireCapacityTimeout time.Duration
	authHandler            *auth.AuthHandler

	// Per-client capacity management
	clientQuotas  map[string]chan struct{}
	spilloverPool chan struct{}
	quotaMutex    sync.RWMutex

	// Tracking active weights and quotas
	activeWeights  map[string]int
	quotaAllocated map[string]int
}

// NewWeightedStrategy creates a new weighted balancing strategy
func NewWeightedStrategy(ctx context.Context, servers []*server, maxCapacity int, acquireCapacityTimeout time.Duration, authHandler *auth.AuthHandler) *WeightedStrategy {
	ws := &WeightedStrategy{
		servers:                servers,
		currentServerIndex:     0,
		maxCapacity:            maxCapacity,
		acquireCapacityTimeout: acquireCapacityTimeout,
		authHandler:            authHandler,
		clientQuotas:           make(map[string]chan struct{}),
		activeWeights:          make(map[string]int),
		quotaAllocated:         make(map[string]int),
	}

	ws.spilloverPool = make(chan struct{}, maxCapacity)

	// Initial quota calculation
	ws.recalculateQuotas()

	return ws
}

// NextServer iterate through servers maximum of 2 times to find available server in round-robin fashion
func (s *WeightedStrategy) NextServer(ctx context.Context) (http.Handler, error) {
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

// AcquireCapacity attempts to acquire capacity for a client based on their weight and quota
func (s *WeightedStrategy) AcquireCapacity(ctx context.Context, timeout time.Duration) error {
	// Extract client name from context - should always be present due to auth middleware
	clientName := ExtractClientFromContext(ctx)
	if clientName == "" {
		log.Printf("No client name found in context - this should not happen after auth middleware")
		return errors.New("no client name found in context")
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	s.quotaMutex.RLock()
	clientQuota, exists := s.clientQuotas[clientName]
	s.quotaMutex.RUnlock()

	if !exists {
		// Client not in quota system yet, try spillover as fallback
		log.Printf("Client %s not in quota system yet, using spillover", clientName)
		return s.acquireFromSpillover(ctx, timeout)
	}

	// Try client-specific quota first
	select {
	case clientQuota <- struct{}{}:
		log.Printf("Acquired capacity from client quota for %s", clientName)
		return nil
	case <-timeoutCtx.Done():
		// Client quota full, try spillover pool
		log.Printf("Client %s quota full, trying spillover", clientName)
		return s.acquireFromSpillover(ctx, timeout)
	}
}

// acquireFromSpillover attempts to acquire capacity from the spillover pool
func (s *WeightedStrategy) acquireFromSpillover(ctx context.Context, timeout time.Duration) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case s.spilloverPool <- struct{}{}:
		log.Printf("Acquired capacity from spillover pool")
		return nil
	case <-timeoutCtx.Done():
		return ErrNoCapacity
	}
}

// ReleaseCapacity releases acquired capacity back to the appropriate pool
// NOTE: Current implementation doesn't track which pool was originally used,
// so it releases to whichever pool is available. This is a simplification that
// works for basic functionality but doesn't provide perfect fairness.
// TODO: Implement proper capacity tracking (see Option 1 in activeContext.md)
func (s *WeightedStrategy) ReleaseCapacity() {
	// For simplicity, release to spillover pool first
	select {
	case <-s.spilloverPool:
		log.Printf("Released capacity to spillover pool")
	default:
		// If spillover is empty, try to release from client quotas
		if s.releaseFromClientQuotas() {
			// Successfully released from client quota
		} else {
			log.Printf("Warning: Could not release capacity - no pools available")
		}
	}
}

// releaseFromClientQuotas attempts to release capacity from client quotas
// Returns true if capacity was successfully released, false otherwise
func (s *WeightedStrategy) releaseFromClientQuotas() bool {
	s.quotaMutex.RLock()
	defer s.quotaMutex.RUnlock()

	for clientName, quota := range s.clientQuotas {
		select {
		case <-quota:
			log.Printf("Released capacity from client quota for %s", clientName)
			return true
		default:
			continue
		}
	}
	return false
}

// GetMaxCapacity returns the maximum system capacity
func (s *WeightedStrategy) GetMaxCapacity() int {
	return s.maxCapacity
}

// GetAvailableCapacity returns currently available capacity across all pools
func (s *WeightedStrategy) GetAvailableCapacity() int {
	s.quotaMutex.RLock()
	clientCapacityUsed := 0
	for _, quota := range s.clientQuotas {
		clientCapacityUsed += len(quota)
	}
	s.quotaMutex.RUnlock()

	totalUsed := len(s.spilloverPool) + clientCapacityUsed
	return s.maxCapacity - totalUsed
}

// recalculateQuotas recalculates client quotas based on current weights
// This method is concurrency-safe and preserves current usage to prevent capacity overflow
func (s *WeightedStrategy) recalculateQuotas() {
	clients := s.authHandler.ListRegisteredClients()

	s.quotaMutex.Lock()
	defer s.quotaMutex.Unlock()

	// Calculate total active weights
	totalWeight := 0
	newActiveWeights := make(map[string]int)

	for clientName, client := range clients {
		totalWeight += client.Weight
		newActiveWeights[clientName] = client.Weight
	}

	if totalWeight == 0 {
		// No active clients, reset everything
		s.activeWeights = make(map[string]int)
		s.quotaAllocated = make(map[string]int)
		s.clientQuotas = make(map[string]chan struct{})
		s.spilloverPool = make(chan struct{}, s.maxCapacity)
		log.Printf("No active clients, reset all quotas")
		return
	}

	// Calculate current usage to avoid capacity overflow
	currentUsage := make(map[string]int)
	totalCurrentUsage := 0

	for clientName, quota := range s.clientQuotas {
		usage := len(quota)
		currentUsage[clientName] = usage
		totalCurrentUsage += usage
		log.Printf("Client %s has %d tokens in its quota", clientName, usage)
	}
	spilloverUsage := len(s.spilloverPool)
	log.Printf("Spillover usage: %d", spilloverUsage)
	totalCurrentUsage += spilloverUsage

	// Calculate new quotas
	newQuotaAllocated := make(map[string]int)
	totalAllocated := 0

	for clientName, weight := range newActiveWeights {
		quota := max((weight*s.maxCapacity)/totalWeight, 1)

		// Ensure we don't reduce quota below current usage to prevent overflow
		currentUsageForClient := currentUsage[clientName]
		if quota < currentUsageForClient {
			quota = currentUsageForClient
			log.Printf("Client %s quota adjusted from calculated %d to current usage %d to prevent overflow",
				clientName, (weight*s.maxCapacity)/totalWeight, currentUsageForClient)
		}

		newQuotaAllocated[clientName] = quota
		totalAllocated += quota
	}

	// Adjust for rounding errors - ensure we don't exceed maxCapacity
	if totalAllocated > s.maxCapacity {
		// Scale down quotas proportionally while preserving minimum current usage
		scaleFactor := float64(s.maxCapacity) / float64(totalAllocated)
		totalAllocated = 0

		for clientName, quota := range newQuotaAllocated {
			scaledQuota := int(float64(quota) * scaleFactor)
			minQuota := currentUsage[clientName]
			if scaledQuota < minQuota {
				scaledQuota = minQuota
			}
			newQuotaAllocated[clientName] = scaledQuota
			totalAllocated += scaledQuota
		}
	}

	// Create new client quota channels preserving current tokens
	newClientQuotas := make(map[string]chan struct{})
	for clientName, newQuota := range newQuotaAllocated {
		newChannel := make(chan struct{}, newQuota)

		// Preserve current usage by pre-filling the channel
		currentUsageForClient := currentUsage[clientName]
		for range currentUsageForClient {
			select {
			case newChannel <- struct{}{}:
				// Successfully preserved current usage
			default:
				// This shouldn't happen since we sized the channel appropriately
				log.Printf("Warning: Could not preserve usage for client %s", clientName)
			}
		}

		newClientQuotas[clientName] = newChannel
	}

	// Update spillover pool size and preserve current usage
	spilloverSize := max(s.maxCapacity-totalAllocated, spilloverUsage)

	newSpilloverPool := make(chan struct{}, spilloverSize)
	// Preserve current spillover usage
	for range spilloverUsage {
		select {
		case newSpilloverPool <- struct{}{}:
			// Successfully preserved spillover usage
		default:
			log.Printf("Warning: Could not preserve spillover usage")
		}
	}

	// Update internal state
	s.activeWeights = newActiveWeights
	s.quotaAllocated = newQuotaAllocated
	s.clientQuotas = newClientQuotas
	s.spilloverPool = newSpilloverPool

	log.Printf("Safely recalculated quotas: %v, spillover size: %d (current usage preserved)",
		newQuotaAllocated, spilloverSize)
}

// RecalculateQuotas is a public method to trigger quota recalculation
// This should be called when clients register or deregister
func (s *WeightedStrategy) RecalculateQuotas() {
	s.recalculateQuotas()
}
