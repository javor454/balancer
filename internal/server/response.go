package server

import (
	"encoding/json"
	"net/http"
	"time"

	"log"
)

// TODO mapping of errors?

type ErrorResponse struct {
	Error     string `json:"error"`
	Timestamp string `json:"timestamp"`
}

func WriteError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(ErrorResponse{
		Error:     message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		log.Printf("Failed to encode response: %s", err)
	}
}

func WriteSuccess[T any](w http.ResponseWriter, data T, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode response: %s", err)
	}
}
