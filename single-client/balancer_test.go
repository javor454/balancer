package singleclient

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBalancer(t *testing.T) {
	t.Run("should return error when client queue is full", func(t *testing.T) {
		const (
			requestCapacity = 3
			clientCapacity  = 3
		)
		balancer := NewBalancer(requestCapacity, clientCapacity, 5*time.Second)

		// +1 because first client is set to be active, not enqueued
		for i := 0; i < clientCapacity+1; i++ {
			_, err := balancer.RegisterClient(strconv.Itoa(i))
			assert.Nil(t, err, "expected client to be registered")
		}

		client, err := balancer.RegisterClient("4")
		assert.ErrorIs(t, err, ErrQueueFull, "expected error when client queue is full")
		assert.Nil(t, client, "expected client not to be returned on failure")
	})

	t.Run("should return error when request capacity is exceeded", func(t *testing.T) {
		const (
			requestCapacity = 3
			clientCapacity  = 3
		)
		balancer := NewBalancer(requestCapacity, clientCapacity, 5*time.Second)

		client, err := balancer.RegisterClient("1")
		assert.Nil(t, err, "expected client to be registered")

		for i := 0; i < requestCapacity; i++ {
			perm := balancer.RequestPermit(client)
			assert.True(t, perm, "expected permit to add request")
		}

		perm := balancer.RequestPermit(client)
		assert.False(t, perm, "expected no permit when request capacity is exceeded")
	})

	t.Run("should not permit request from inactive client", func(t *testing.T) {
		const (
			requestCapacity = 3
			clientCapacity  = 3
		)

		balancer := NewBalancer(requestCapacity, clientCapacity, 5*time.Second)

		activeClient, err := balancer.RegisterClient("1")
		assert.Nil(t, err, "expected client to be registered")

		perm := balancer.RequestPermit(activeClient)
		assert.True(t, perm, "expected permit when client is active")

		enqueuedClient, err := balancer.RegisterClient("2")
		assert.Nil(t, err, "expected client to be registered")

		perm = balancer.RequestPermit(enqueuedClient)
		assert.False(t, perm, "expected no permit when client is inactive")
	})

	t.Run("should activate next client when current client is deregistered", func(t *testing.T) {
		const (
			requestCapacity = 3
			clientCapacity  = 3
		)

		balancer := NewBalancer(requestCapacity, clientCapacity, 5*time.Second)

		activeClient, err := balancer.RegisterClient("1")
		assert.Nil(t, err, "expected client to be registered")

		enqueuedClient, err := balancer.RegisterClient("2")
		assert.Nil(t, err, "expected client to be registered")

		balancer.DeregisterClient(activeClient)

		perm := balancer.RequestPermit(enqueuedClient)
		assert.True(t, perm, "expected permit when client is active")
	})

	t.Run("should release permit after release interval", func(t *testing.T) {
		const (
			requestCapacity = 1
			clientCapacity  = 3
			releaseInterval = 100 * time.Millisecond
		)

		balancer := NewBalancer(requestCapacity, clientCapacity, releaseInterval)

		client, err := balancer.RegisterClient("1")
		assert.Nil(t, err, "expected client to be registered")

		perm1 := balancer.RequestPermit(client)
		assert.True(t, perm1, "expected first permit to be granted")

		perm2 := balancer.RequestPermit(client)
		assert.False(t, perm2, "expected second request to be rejected")

		time.Sleep(releaseInterval + 10*time.Millisecond)

		perm3 := balancer.RequestPermit(client)
		assert.True(t, perm3, "expected third request to be permitted after release")
	})

	t.Run("should not permit request when there are no active clients", func(t *testing.T) {
		const (
			requestCapacity = 1
			clientCapacity  = 3
			releaseInterval = 100 * time.Millisecond
		)

		balancer := NewBalancer(requestCapacity, clientCapacity, releaseInterval)

		perm := balancer.RequestPermit(nil)
		assert.False(t, perm, "expected no permit when there are no active clients")

		client, err := balancer.RegisterClient("1")
		assert.Nil(t, err, "expected client to be registered")

		balancer.DeregisterClient(client)

		perm = balancer.RequestPermit(client)
		assert.False(t, perm, "expected no permit when there are no active clients")
	})

	t.Run("should not permit request when there are no active clients", func(t *testing.T) {
		const (
			requestCapacity = 1
			clientCapacity  = 3
			releaseInterval = 100 * time.Millisecond
		)

		balancer := NewBalancer(requestCapacity, clientCapacity, releaseInterval)

		balancer.DeregisterClient(nil)
	})
}
