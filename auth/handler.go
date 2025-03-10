package auth

import (
	"context"
	"log"
	"sync"
	"time"
)

type Client struct {
	Name         string
	Weight       int
	RegisteredAt time.Time
}

type AuthHandler struct {
	clients map[string]Client
	mu      sync.RWMutex
}

func NewAuthHandler(ctx context.Context) *AuthHandler {
	h := &AuthHandler{
		clients: make(map[string]Client),
	}
	go h.cleanupClients(ctx)

	return h
}

// VerifyRegistered validates if the client is registered
func (h *AuthHandler) VerifyRegistered(name string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	_, ok := h.clients[name]
	return ok
}

// RegisterClient dummy implementation of registering a client TODO improve?
func (h *AuthHandler) RegisterClient(name string, weight int) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[name] = Client{
		Name:         name,
		Weight:       weight,
		RegisteredAt: time.Now(),
	}
	log.Printf("Registered client \"%s\" with weight %d", name, weight)
}

// cleanupClients cleans up clients that have been registered for more than 5 minutes every 5 seconds
func (h *AuthHandler) cleanupClients(ctx context.Context) {
	log.Println("Starting cleanup of clients")
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping cleanup of clients")
			return
		case <-ticker.C:
			h.mu.Lock()
			for name, client := range h.clients {
				if time.Since(client.RegisteredAt) > 5*time.Minute {
					log.Printf("Cleaning up client %s", name)
					delete(h.clients, name)
				}
			}
			h.mu.Unlock()
		}
	}
}
