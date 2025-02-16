package balancer

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

type ClientState struct {
	*Client
	pendingJobs []uuid.UUID // FIFO queue of jobs waiting to be processed
}

type RoundRobinBalancer struct {
	capacity        int
	clients         map[uuid.UUID]*ClientState
	activeJobs      map[uuid.UUID]Job
	completedJobs   map[uuid.UUID]Job
	clientOrder     []uuid.UUID // Maintains order of clients for round-robin
	currentIndex    int         // Current position in the rotation
	mutex           sync.Mutex
	logger          *log.Logger
	cleanupInterval time.Duration
	processJobFn    func(jobID uuid.UUID)
}

func NewRoundRobinBalancer(ctx context.Context, capacity int, logger *log.Logger, cleanupInterval time.Duration, jobDuration time.Duration) (*RoundRobinBalancer, error) {
	b := &RoundRobinBalancer{
		capacity:        capacity,
		clients:         make(map[uuid.UUID]*ClientState),
		activeJobs:      make(map[uuid.UUID]Job),
		completedJobs:   make(map[uuid.UUID]Job),
		clientOrder:     make([]uuid.UUID, 0),
		currentIndex:    0,
		logger:          logger,
		cleanupInterval: cleanupInterval,
	}

	b.processJobFn = func(jobID uuid.UUID) {
		time.Sleep(jobDuration)
		b.processJob(jobID)
	}

	logger.Printf("Round-Robin balancer created with capacity: %d", capacity)

	go b.cleanupInactiveClients(ctx)
	go b.cleanupFinishedJobs(ctx)

	return b, nil
}

func (b *RoundRobinBalancer) RegisterClient() (uuid.UUID, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	client := NewClient()
	b.clients[client.ID] = &ClientState{
		Client: client,
	}
	b.logger.Printf("Client %s registered", client.ID)

	b.clientOrder = append(b.clientOrder, client.ID)

	return client.ID, nil
}

func (b *RoundRobinBalancer) RegisterJob(clientID uuid.UUID) (uuid.UUID, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	clientState, exists := b.clients[clientID]
	if !exists {
		return uuid.Nil, ErrorClientNotActive
	}

	// Create and queue the job
	jobID := uuid.New()
	b.activeJobs[jobID] = Job{
		ID:        jobID,
		CreatedAt: time.Now(),
	}

	// Add to client's pending jobs queue
	clientState.pendingJobs = append(clientState.pendingJobs, jobID)
	clientState.LastActive = time.Now()

	b.logger.Printf("Job %s queued for client %s (queue length: %d)",
		jobID, clientID, len(clientState.pendingJobs))

	// If it's this client's turn, process their next job
	if b.clientOrder[b.currentIndex] == clientID {
		b.processNextJob()
	}

	return jobID, nil
}

func (b *RoundRobinBalancer) processNextJob() {
	// Check capacity first
	if len(b.activeJobs) >= b.capacity {
		return // Wait for some jobs to complete before processing more
	}

	if len(b.clientOrder) == 0 {
		return
	}

	startIndex := b.currentIndex
	for {
		currentClientID := b.clientOrder[b.currentIndex]
		clientState := b.clients[currentClientID]

		if len(clientState.pendingJobs) > 0 {
			// Process the next job for current client
			jobID := clientState.pendingJobs[0]
			clientState.pendingJobs = clientState.pendingJobs[1:]

			b.logger.Printf("Processing job %s for client %s (remaining queue: %d)",
				jobID, currentClientID, len(clientState.pendingJobs))

			go b.processJobFn(jobID)

			// Move to next client
			b.currentIndex = (b.currentIndex + 1) % len(b.clientOrder)
			return
		}

		// Move to next client
		b.currentIndex = (b.currentIndex + 1) % len(b.clientOrder)

		// If we've checked all clients, stop
		if b.currentIndex == startIndex {
			return
		}
	}
}

func (b *RoundRobinBalancer) processJob(jobID uuid.UUID) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if job, exists := b.activeJobs[jobID]; exists {
		job.CompletedAt = time.Now()
		b.completedJobs[jobID] = job
		delete(b.activeJobs, jobID)
		b.logger.Printf("Job %s completed", jobID)

		// Try to process next job in rotation
		b.processNextJob()
	}
}

func (b *RoundRobinBalancer) GetClientStatus(clientID uuid.UUID) (status string, position int, err error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if _, exists := b.clients[clientID]; !exists {
		return "", 0, ErrorClientNotFound
	}

	// Find position in rotation
	for i, id := range b.clientOrder {
		if id == clientID {
			// Calculate position relative to current turn
			position = (i - b.currentIndex)
			if position < 0 {
				position += len(b.clientOrder)
			}
			return StatusActive, position, nil
		}
	}

	return StatusActive, 0, nil
}

func (b *RoundRobinBalancer) GetJobStatus(jobID uuid.UUID) (status string, err error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	job, exists := b.activeJobs[jobID]
	if !exists {
		return "", ErrorJobNotFound
	}

	if job.CompletedAt.IsZero() {
		return StatusPending, nil
	}
	return StatusFinished, nil
}

func (b *RoundRobinBalancer) Deregister(clientID uuid.UUID) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if _, exists := b.clients[clientID]; !exists {
		return ErrorClientNotFound
	}

	// Remove from clients map
	delete(b.clients, clientID)

	// Remove from rotation order
	for i, id := range b.clientOrder {
		if id == clientID {
			b.clientOrder = append(b.clientOrder[:i], b.clientOrder[i+1:]...)
			// Adjust currentIndex if needed
			if i < b.currentIndex {
				b.currentIndex--
			}
			if b.currentIndex >= len(b.clientOrder) {
				b.currentIndex = 0
			}
			break
		}
	}

	b.logger.Printf("Client %s deregistered, rotation size: %d, current index: %d",
		clientID, len(b.clientOrder), b.currentIndex)
	return nil
}

func (b *RoundRobinBalancer) cleanupInactiveClients(ctx context.Context) {
	ticker := time.NewTicker(b.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.mutex.Lock()
			now := time.Now()
			newOrder := make([]uuid.UUID, 0)

			// Keep track of removed clients to update rotation
			for id, client := range b.clients {
				if now.Sub(client.LastActive) >= time.Minute && len(client.pendingJobs) == 0 {
					// Only remove if client has no pending jobs
					delete(b.clients, id)
					b.logger.Printf("Client %s removed due to inactivity", id)
				} else {
					// Keep active clients in the rotation
					newOrder = append(newOrder, id)
				}
			}

			// Update rotation order and adjust currentIndex
			b.clientOrder = newOrder
			if b.currentIndex >= len(b.clientOrder) {
				b.currentIndex = 0
			}

			b.mutex.Unlock()
		}
	}
}

func (b *RoundRobinBalancer) cleanupFinishedJobs(ctx context.Context) {
	ticker := time.NewTicker(b.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.mutex.Lock()
			now := time.Now()

			for jobID, job := range b.completedJobs {
				if !job.CompletedAt.IsZero() && now.Sub(job.CompletedAt) > time.Minute {
					delete(b.completedJobs, jobID)
					b.logger.Printf("Job %s cleaned up", jobID)
				}
			}
			b.mutex.Unlock()
		}
	}
}
