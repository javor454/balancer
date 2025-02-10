package balancer

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/javor454/balancer/server"
)

// Strategy interface defines the methods that each balancing strategy must implement
type Strategy interface {
	RegisterClient() (uuid.UUID, error)
	ProcessRequest(clientID uuid.UUID) (uuid.UUID, error)
	GetClientStatus(clientID uuid.UUID) (status string, position int, err error)
	GetJobStatus(jobID uuid.UUID) (status string, err error)
	Deregister(clientID uuid.UUID) error
}

type Balancer struct {
	strategy Strategy
	logger   *log.Logger
}

func NewBalancer(ctx context.Context, config *Config, logger *log.Logger) (*Balancer, error) {
	switch config.Strategy {
	case SingleClient:
		strategy, err := NewSingleClientBalancer(ctx, config.Capacity, logger, config.SessionTimeout.Duration)
		if err != nil {
			return nil, fmt.Errorf("failed to create single client balancer: %w", err)
		}
		return &Balancer{strategy: strategy, logger: logger}, nil
	default:
		return nil, fmt.Errorf("invalid strategy %q", config.Strategy)
	}
}

func (b *Balancer) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("POST /clients", b.handleRegister)
	mux.HandleFunc("DELETE /clients/{client_id}", b.handleDeregister)
	mux.HandleFunc("GET /clients/{client_id}", b.handleClientStatus)
	mux.HandleFunc("GET /jobs/{job_id}", b.handleJobStatus)
	mux.HandleFunc("POST /clients/{client_id}/requests", b.handleProcess)
}

func (b *Balancer) handleRegister(w http.ResponseWriter, r *http.Request) {
	clientID, err := b.strategy.RegisterClient()
	if err != nil {
		b.logger.Printf("Failed to register client: %v", err)
		server.WriteError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	server.WriteSuccess(w, map[string]string{
		"client_id": clientID.String(),
		"message":   "Registration successful",
	}, http.StatusCreated)
}

func (b *Balancer) handleProcess(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("client_id")
	if cid == "" {
		server.WriteError(w, "client ID is required", http.StatusBadRequest)
		return
	}

	clientID, err := uuid.Parse(cid)
	if err != nil {
		server.WriteError(w, "invalid client ID", http.StatusBadRequest)
		return
	}

	jobID, err := b.strategy.ProcessRequest(clientID)
	if err != nil {
		server.WriteError(w, err.Error(), http.StatusBadRequest)
		return
	}

	server.WriteSuccess(w, map[string]string{
		"job_id": jobID.String(),
	}, http.StatusOK)
}

func (b *Balancer) handleClientStatus(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("client_id")
	if cid == "" {
		server.WriteError(w, "client ID is required", http.StatusBadRequest)
		return
	}

	clientID, err := uuid.Parse(cid)
	if err != nil {
		server.WriteError(w, "invalid client ID", http.StatusBadRequest)
		return
	}

	status, position, err := b.strategy.GetClientStatus(clientID)
	if err != nil {
		server.WriteError(w, err.Error(), http.StatusNotFound)
		return
	}

	server.WriteSuccess(w, map[string]interface{}{
		"status":   status,
		"position": position,
	}, http.StatusOK)
}

func (b *Balancer) handleJobStatus(w http.ResponseWriter, r *http.Request) {
	jid := r.PathValue("job_id")
	if jid == "" {
		server.WriteError(w, "job ID is required", http.StatusBadRequest)
		return
	}

	jobID, err := uuid.Parse(jid)
	if err != nil {
		server.WriteError(w, "invalid job ID", http.StatusBadRequest)
		return
	}

	status, err := b.strategy.GetJobStatus(jobID)
	if err != nil {
		server.WriteError(w, err.Error(), http.StatusNotFound)
		return
	}

	server.WriteSuccess(w, map[string]string{
		"status": status,
	}, http.StatusOK)
}

func (b *Balancer) handleDeregister(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("client_id")
	if cid == "" {
		server.WriteError(w, "client ID is required", http.StatusBadRequest)
		return
	}

	clientID, err := uuid.Parse(cid)
	if err != nil {
		server.WriteError(w, "invalid client ID", http.StatusBadRequest)
		return
	}

	err = b.strategy.Deregister(clientID)
	if err != nil {
		server.WriteError(w, err.Error(), http.StatusNotFound)
		return
	}

	server.WriteSuccess(w, map[string]string{
		"message": "Successfully deregistered",
	}, http.StatusOK)
}
