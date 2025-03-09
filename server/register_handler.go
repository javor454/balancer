package server

import (
	"encoding/json"
	"net/http"

	"github.com/javor454/balancer/auth"
)

type RegisterRequest struct {
	Name   string `json:"name"`
	Weight int    `json:"weight"`
}

type RegisterHandler struct {
	authHandler *auth.AuthHandler
}

func NewRegisterHandler(authHandler *auth.AuthHandler) *RegisterHandler {
	return &RegisterHandler{
		authHandler: authHandler,
	}
}

func (h *RegisterHandler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := readBody(r)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	var req RegisterRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		http.Error(w, "Failed to unmarshal request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	if req.Weight == 0 {
		http.Error(w, "Weight is required", http.StatusBadRequest)
		return
	}

	if req.Weight < 1 || req.Weight > 5 {
		http.Error(w, "Weight must be between 1 and 5", http.StatusBadRequest)
		return
	}

	h.authHandler.RegisterClient(req.Name, req.Weight)

	w.WriteHeader(http.StatusCreated)
}
