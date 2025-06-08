package server

import (
	"context"
	"testing"
	"time"

	"github.com/javor454/balancer/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWeightedStrategy_QuotaCalculation(t *testing.T) {
	ctx := context.Background()
	authHandler := auth.NewAuthHandler(ctx)

	// Register test clients with different weights
	authHandler.RegisterClient("client1", 2)
	authHandler.RegisterClient("client2", 3)
	authHandler.RegisterClient("client3", 1)

	servers := []*server{}
	maxCapacity := 6
	timeout := 1 * time.Second

	strategy := NewWeightedStrategy(ctx, servers, maxCapacity, timeout, authHandler)

	// Trigger quota recalculation after registering clients
	strategy.RecalculateQuotas()

	// Verify quota allocation
	strategy.quotaMutex.RLock()
	quotas := make(map[string]int)
	for client, quota := range strategy.quotaAllocated {
		quotas[client] = quota
	}
	strategy.quotaMutex.RUnlock()

	// Total weights: 2 + 3 + 1 = 6
	// Expected quotas: client1=2, client2=3, client3=1
	assert.Equal(t, 2, quotas["client1"], "client1 should get quota of 2")
	assert.Equal(t, 3, quotas["client2"], "client2 should get quota of 3")
	assert.Equal(t, 1, quotas["client3"], "client3 should get quota of 1")

	// Verify total allocated capacity doesn't exceed max
	totalAllocated := 0
	for _, quota := range quotas {
		totalAllocated += quota
	}
	assert.LessOrEqual(t, totalAllocated, maxCapacity, "Total allocated should not exceed max capacity")
}

func TestWeightedStrategy_QuotaCalculationWithRounding(t *testing.T) {
	ctx := context.Background()
	authHandler := auth.NewAuthHandler(ctx)

	// Register clients where weights don't divide evenly
	authHandler.RegisterClient("client1", 1)
	authHandler.RegisterClient("client2", 1)
	authHandler.RegisterClient("client3", 1)

	servers := []*server{}
	maxCapacity := 5 // 5 capacity for 3 clients = 1.67 each
	timeout := 1 * time.Second

	strategy := NewWeightedStrategy(ctx, servers, maxCapacity, timeout, authHandler)
	strategy.RecalculateQuotas()

	strategy.quotaMutex.RLock()
	totalAllocated := 0
	for _, quota := range strategy.quotaAllocated {
		totalAllocated += quota
		assert.GreaterOrEqual(t, quota, 1, "Each client should get at least 1 quota")
	}
	strategy.quotaMutex.RUnlock()

	assert.LessOrEqual(t, totalAllocated, maxCapacity, "Total allocated should not exceed max capacity")
}

func TestWeightedStrategy_NoActiveClients(t *testing.T) {
	ctx := context.Background()
	authHandler := auth.NewAuthHandler(ctx)

	servers := []*server{}
	maxCapacity := 5
	timeout := 1 * time.Second

	strategy := NewWeightedStrategy(ctx, servers, maxCapacity, timeout, authHandler)
	strategy.RecalculateQuotas()

	// Verify spillover pool has full capacity when no clients
	assert.Equal(t, maxCapacity, cap(strategy.spilloverPool), "Spillover pool should have full capacity")
	assert.Equal(t, 0, len(strategy.clientQuotas), "No client quotas should exist")
}

func TestWeightedStrategy_CapacityAcquisition(t *testing.T) {
	ctx := context.Background()
	authHandler := auth.NewAuthHandler(ctx)

	// Register test clients to ensure spillover pool has capacity
	authHandler.RegisterClient("client1", 1)
	authHandler.RegisterClient("client2", 1)

	servers := []*server{}
	maxCapacity := 5 // 5 capacity, 2 clients with weight 1 each = 2 allocated, 3 spillover
	timeout := 100 * time.Millisecond

	strategy := NewWeightedStrategy(ctx, servers, maxCapacity, timeout, authHandler)
	strategy.RecalculateQuotas()

	// Test capacity acquisition with unknown client (should fail - no client context)
	unknownCtx := context.Background()
	err := strategy.AcquireCapacity(unknownCtx, timeout)
	assert.Error(t, err, "Should fail for request without client context")
	assert.Contains(t, err.Error(), "no client name found in context", "Should return client context error")

	// Test capacity acquisition with known client
	clientCtx := WithClientContext(context.Background(), "client1")
	err = strategy.AcquireCapacity(clientCtx, timeout)
	assert.NoError(t, err, "Should acquire from client quota")

	// Test capacity acquisition with client not yet in quota system (should use spillover)
	newClientCtx := WithClientContext(context.Background(), "client3")
	err = strategy.AcquireCapacity(newClientCtx, timeout)
	assert.NoError(t, err, "Should acquire from spillover for client not in quota system")

	// Release capacity
	strategy.ReleaseCapacity()
	strategy.ReleaseCapacity()
}

func TestWeightedStrategy_GetAvailableCapacity(t *testing.T) {
	ctx := context.Background()
	authHandler := auth.NewAuthHandler(ctx)

	// Register a test client to allow capacity acquisition
	authHandler.RegisterClient("testclient", 1)

	servers := []*server{}
	maxCapacity := 5
	timeout := 100 * time.Millisecond

	strategy := NewWeightedStrategy(ctx, servers, maxCapacity, timeout, authHandler)

	// Initially should have full capacity available
	available := strategy.GetAvailableCapacity()
	assert.Equal(t, maxCapacity, available, "Should start with full capacity available")

	// Acquire some capacity with proper client context
	clientCtx := WithClientContext(context.Background(), "testclient")
	err := strategy.AcquireCapacity(clientCtx, timeout)
	require.NoError(t, err)

	// Available capacity should decrease
	available = strategy.GetAvailableCapacity()
	assert.Equal(t, maxCapacity-1, available, "Available capacity should decrease after acquisition")

	// Release capacity
	strategy.ReleaseCapacity()

	// Available capacity should increase
	available = strategy.GetAvailableCapacity()
	assert.Equal(t, maxCapacity, available, "Available capacity should return to full after release")
}

func TestWeightedStrategy_ConcurrencySafeQuotaRecalculation(t *testing.T) {
	ctx := context.Background()
	authHandler := auth.NewAuthHandler(ctx)

	// Register initial client
	authHandler.RegisterClient("client1", 2)

	servers := []*server{}
	maxCapacity := 5
	timeout := 100 * time.Millisecond

	strategy := NewWeightedStrategy(ctx, servers, maxCapacity, timeout, authHandler)
	strategy.RecalculateQuotas()

	// Client1 should get full capacity (quota=5)
	strategy.quotaMutex.RLock()
	client1Quota := strategy.quotaAllocated["client1"]
	strategy.quotaMutex.RUnlock()
	assert.Equal(t, 5, client1Quota, "Client1 should get full capacity initially")

	// Acquire 3 slots for client1
	client1Ctx := WithClientContext(context.Background(), "client1")
	err := strategy.AcquireCapacity(client1Ctx, timeout)
	require.NoError(t, err)
	err = strategy.AcquireCapacity(client1Ctx, timeout)
	require.NoError(t, err)
	err = strategy.AcquireCapacity(client1Ctx, timeout)
	require.NoError(t, err)

	// Verify 3 slots are in use
	strategy.quotaMutex.RLock()
	currentUsage := len(strategy.clientQuotas["client1"])
	strategy.quotaMutex.RUnlock()
	assert.Equal(t, 3, currentUsage, "Client1 should be using 3 slots")

	// Now register a new client with higher weight - this would normally reduce client1's quota
	authHandler.RegisterClient("client2", 4) // Higher weight than client1 (2)

	// Trigger quota recalculation
	strategy.RecalculateQuotas()

	// After recalculation, quotas should be adjusted but current usage preserved
	strategy.quotaMutex.RLock()
	newClient1Quota := strategy.quotaAllocated["client1"]
	newClient2Quota := strategy.quotaAllocated["client2"]
	currentUsageAfter := len(strategy.clientQuotas["client1"])
	strategy.quotaMutex.RUnlock()

	// Client1's quota should not go below its current usage (3)
	assert.GreaterOrEqual(t, newClient1Quota, 3, "Client1 quota should not go below current usage")
	assert.Equal(t, 3, currentUsageAfter, "Client1 should still be using 3 slots")

	// Total allocated should not exceed max capacity
	totalAllocated := newClient1Quota + newClient2Quota
	assert.LessOrEqual(t, totalAllocated, maxCapacity, "Total allocated should not exceed max capacity")

	// The system should still be functional - client2 should be able to acquire capacity
	client2Ctx := WithClientContext(context.Background(), "client2")
	err = strategy.AcquireCapacity(client2Ctx, timeout)
	assert.NoError(t, err, "Client2 should be able to acquire capacity")

	// Clean up
	strategy.ReleaseCapacity() // client2's slot
	strategy.ReleaseCapacity() // client1's slot
	strategy.ReleaseCapacity() // client1's slot
	strategy.ReleaseCapacity() // client1's slot
}

func TestWeightedStrategy_BasicCapacityPreservation(t *testing.T) {
	ctx := context.Background()
	authHandler := auth.NewAuthHandler(ctx)

	// Register initial client with low weight
	authHandler.RegisterClient("lowPriorityClient", 1)

	servers := []*server{}
	maxCapacity := 3
	timeout := 100 * time.Millisecond

	strategy := NewWeightedStrategy(ctx, servers, maxCapacity, timeout, authHandler)
	strategy.RecalculateQuotas()

	// LowPriorityClient gets full capacity initially (quota=3)
	strategy.quotaMutex.RLock()
	initialQuota := strategy.quotaAllocated["lowPriorityClient"]
	strategy.quotaMutex.RUnlock()
	assert.Equal(t, 3, initialQuota, "Low priority client should get full capacity initially")

	// LowPriorityClient acquires all 3 slots
	clientCtx := WithClientContext(context.Background(), "lowPriorityClient")
	for i := 0; i < 3; i++ {
		err := strategy.AcquireCapacity(clientCtx, timeout)
		require.NoError(t, err, "Should acquire capacity slot %d", i+1)
	}

	// Register high priority client
	authHandler.RegisterClient("highPriorityClient", 4) // Much higher weight

	// Trigger quota recalculation - this will preserve lowPriorityClient's usage
	strategy.RecalculateQuotas()

	strategy.quotaMutex.RLock()
	lowQuotaAfterRecalc := strategy.quotaAllocated["lowPriorityClient"]
	highQuotaAfterRecalc := strategy.quotaAllocated["highPriorityClient"]
	lowUsageAfterRecalc := len(strategy.clientQuotas["lowPriorityClient"])
	strategy.quotaMutex.RUnlock()

	// Quotas should be adjusted but low priority client's usage preserved
	assert.Equal(t, 3, lowQuotaAfterRecalc, "Low priority quota preserved due to current usage")
	assert.Equal(t, 1, highQuotaAfterRecalc, "High priority gets minimum quota of 1")
	assert.Equal(t, 3, lowUsageAfterRecalc, "Low priority still using all 3 slots")

	// Release capacity (without automatic rebalancing)
	strategy.ReleaseCapacity()

	// Verify that quota structure remains stable (no automatic rebalancing)
	strategy.quotaMutex.RLock()
	lowQuotaAfterRelease := strategy.quotaAllocated["lowPriorityClient"]
	highQuotaAfterRelease := strategy.quotaAllocated["highPriorityClient"]
	strategy.quotaMutex.RUnlock()

	// Quotas should remain the same after release (no automatic rebalancing)
	assert.Equal(t, lowQuotaAfterRecalc, lowQuotaAfterRelease, "Low priority quota should remain stable")
	assert.Equal(t, highQuotaAfterRecalc, highQuotaAfterRelease, "High priority quota should remain stable")

	// High priority client should be able to acquire capacity from their quota or spillover
	highClientCtx := WithClientContext(context.Background(), "highPriorityClient")
	err := strategy.AcquireCapacity(highClientCtx, timeout)
	assert.NoError(t, err, "High priority client should be able to acquire capacity")

	// Clean up
	strategy.ReleaseCapacity() // high priority slot
	strategy.ReleaseCapacity() // remaining slots
	strategy.ReleaseCapacity()
}
