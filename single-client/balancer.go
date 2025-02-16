package singleclient

import (
	"errors"
	"sync"
	"time"
)

type Client struct {
	ID string
}

type Balancer struct {
	requestCapacity int
	clientQueue     chan *Client
	activeClient    *Client
	activeRequests  int
	mu              sync.Mutex
	releaseInterval time.Duration
}

var ErrQueueFull = errors.New("queue is full")

// NewBalancer creates a new balancer with given capacity
func NewBalancer(requestCapacity, clientCapacity int, releaseInterval time.Duration) *Balancer {
	return &Balancer{
		requestCapacity: requestCapacity,
		clientQueue:     make(chan *Client, clientCapacity), // Buffer size for waiting clients
		releaseInterval: releaseInterval,
	}
}

// RegisterClient adds a new client to the balancer
func (b *Balancer) RegisterClient(id string) (*Client, error) {
	client := &Client{
		ID: id,
	}

	// If no active client, make this one active
	if b.activeClient == nil {
		b.activeClient = client
		return client, nil
	}

	// If queue is full, return error
	select {
	case b.clientQueue <- client:
		return client, nil
	default:
		return nil, ErrQueueFull
	}
}

// RequestPermit attempts to get permission to make a request
func (b *Balancer) RequestPermit(c *Client) bool {
	// Only active client can make requests
	if c == nil || b.activeClient != c {
		return false
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if b.activeRequests < b.requestCapacity {
		b.activeRequests++

		go b.releasePermit()

		return true
	}

	return false
}

// releasePermit releases a permit after request completion
func (b *Balancer) releasePermit() {
	time.Sleep(b.releaseInterval)

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.activeRequests > 0 {
		b.activeRequests--
	}
}

// DeregisterClient removes a client from the balancer
func (b *Balancer) DeregisterClient(c *Client) {
	if b.activeClient == c {
		b.switchToNextClient()
	}
}

// switchToNextClient moves to the next client in queue or sets active client to nil if queue is empty
func (b *Balancer) switchToNextClient() {
	select {
	case nextClient := <-b.clientQueue:
		b.activeClient = nextClient
	default:
		b.activeClient = nil
	}
}
