package balancer

import (
	"time"

	"github.com/google/uuid"
)

type Client struct {
	ID         uuid.UUID
	LastActive time.Time
}

func NewClient() *Client {
	return &Client{
		ID:         uuid.New(),
		LastActive: time.Now(),
	}
}

type Job struct {
	ID uuid.UUID

	CreatedAt   time.Time
	CompletedAt time.Time
}
