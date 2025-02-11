package balancer

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	StatusActive   = "active"
	StatusQueued   = "queued"
	StatusFinished = "finished"
	StatusPending  = "pending"
)

var (
	ErrorClientNotFound   = errors.New("client not found")
	ErrorServerAtCapacity = errors.New("server at capacity")
	ErrorClientNotActive  = errors.New("client not active")
	ErrorJobNotFound      = errors.New("job not found")
)

type SingleClientBalancer struct {
	capacity        int
	activeClient    *Client
	waitingClients  []Client
	jobs            map[uuid.UUID]Job
	mutex           sync.Mutex
	sessionTimeout  time.Duration
	jobDuration     time.Duration
	cleanupInterval time.Duration
	logger          *log.Logger
	completeJob     func(jobID uuid.UUID)
}

func NewSingleClientBalancer(ctx context.Context, capacity int, logger *log.Logger, sessionTimeout time.Duration, jobDuration time.Duration, cleanupInterval time.Duration) (*SingleClientBalancer, error) {
	b := &SingleClientBalancer{
		capacity:        capacity,
		waitingClients:  make([]Client, 0),
		jobs:            make(map[uuid.UUID]Job, 0),
		sessionTimeout:  sessionTimeout,
		jobDuration:     jobDuration,
		cleanupInterval: cleanupInterval,
		logger:          logger,
	}

	// Default job completion behavior
	b.completeJob = func(jobID uuid.UUID) {
		time.Sleep(jobDuration)
		b.completeRequest(jobID)
	}

	logger.Printf("Single-Client balancer created with capacity: %d", capacity)

	go b.cleanupInactiveClients(ctx)
	go b.cleanupFinishedJobs(ctx)

	return b, nil
}

func (b *SingleClientBalancer) RegisterClient() (uuid.UUID, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	client := NewClient()

	if b.activeClient == nil {
		b.activeClient = client
		b.logger.Printf("Client %s registered, is currently active", client.ID)
	} else {
		b.waitingClients = append(b.waitingClients, *client)
		b.logger.Printf("Client %s queued, position %d", client.ID, len(b.waitingClients))
	}

	return client.ID, nil
}

func (b *SingleClientBalancer) RegisterJob(clientID uuid.UUID) (uuid.UUID, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if b.activeClient == nil || b.activeClient.ID != clientID {
		return uuid.Nil, ErrorClientNotActive
	}

	if time.Since(b.activeClient.LastActive) > b.sessionTimeout {
		b.activateNextClient()
		return uuid.Nil, ErrorClientNotActive
	}

	if len(b.jobs) >= b.capacity {
		return uuid.Nil, ErrorServerAtCapacity
	}

	jobID := uuid.New()
	b.activeClient.LastActive = time.Now()

	b.jobs[jobID] = Job{
		ID:        jobID,
		CreatedAt: time.Now(),
	}

	b.logger.Printf("Job %s added", jobID)

	go b.completeJob(jobID)

	return jobID, nil
}

func (b *SingleClientBalancer) GetClientStatus(clientID uuid.UUID) (status string, position int, err error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if b.activeClient != nil && b.activeClient.ID == clientID {
		return StatusActive, 0, nil
	}

	for i, client := range b.waitingClients {
		if client.ID == clientID {
			return StatusQueued, i + 1, nil
		}
	}

	return "", 0, ErrorClientNotFound
}

func (b *SingleClientBalancer) GetJobStatus(jobID uuid.UUID) (status string, err error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if job, exists := b.jobs[jobID]; exists {
		if job.CompletedAt.IsZero() {
			return StatusPending, nil
		}

		return StatusFinished, nil
	}

	return "", ErrorJobNotFound
}

func (b *SingleClientBalancer) Deregister(clientID uuid.UUID) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if b.activeClient != nil && b.activeClient.ID == clientID {
		b.activateNextClient()
		return nil
	}

	for i, client := range b.waitingClients {
		if client.ID == clientID {
			b.waitingClients = append(b.waitingClients[:i], b.waitingClients[i+1:]...)

			b.logger.Printf("Client %s deregistered", clientID)

			return nil
		}
	}

	return ErrorClientNotFound
}

func (b *SingleClientBalancer) completeRequest(jobID uuid.UUID) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if job, exists := b.jobs[jobID]; exists {
		job.CompletedAt = time.Now()
		b.jobs[jobID] = job

		b.logger.Printf("Job %s completed", jobID)

		return nil
	}

	return ErrorJobNotFound
}

func (b *SingleClientBalancer) cleanupInactiveClients(ctx context.Context) {
	b.logger.Printf("Starting cleanup of inactive clients...")
	ticker := time.NewTicker(b.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			b.logger.Printf("Cleanup of inactive clients stopped")
			return
		case <-ticker.C:
			b.mutex.Lock()
			// Check active client
			if b.activeClient != nil && time.Since(b.activeClient.LastActive) > b.sessionTimeout {
				b.logger.Printf("Client %s timed out", b.activeClient.ID)
				b.activateNextClient()
			}

			// Check waiting clients
			var activeClients []Client
			for _, client := range b.waitingClients {
				if time.Since(client.LastActive) <= b.sessionTimeout {
					activeClients = append(activeClients, client)
				} else {
					b.logger.Printf("Queued client %s cleaned up", client.ID)
				}
			}
			b.waitingClients = activeClients
			b.mutex.Unlock()
		}
	}
}

func (b *SingleClientBalancer) cleanupFinishedJobs(ctx context.Context) {
	b.logger.Printf("Starting cleanup of finished jobs...")
	ticker := time.NewTicker(b.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			b.logger.Printf("Cleanup of finished jobs stopped")
			return
		case <-ticker.C:
			b.mutex.Lock()
			for jobID, job := range b.jobs {
				// Only clean up finished jobs that have been completed for a while
				if !job.CompletedAt.IsZero() && time.Since(job.CompletedAt) > b.sessionTimeout {
					delete(b.jobs, jobID)
					b.logger.Printf("Job %s cleaned up", jobID)
				}
			}
			b.mutex.Unlock()
		}
	}
}

func (b *SingleClientBalancer) activateNextClient() {
	b.activeClient = nil

	if len(b.waitingClients) > 0 {
		nextClient := b.waitingClients[0]
		b.waitingClients = b.waitingClients[1:]
		nextClient.LastActive = time.Now()
		b.activeClient = &nextClient
	}
}
