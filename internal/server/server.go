package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Server struct {
	mux             *http.ServeMux
	logger          *log.Logger
	srv             *http.Server
	shutdownTimeout time.Duration
}

func NewServer(logger *log.Logger, port int, shutdownTimeout time.Duration) *Server {
	mux := http.NewServeMux()

	// Global middlewares
	wrappedMux := Chain(mux, WithLogging(logger), WithPanicRecovery(logger))

	// Create http.Server instance
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: wrappedMux,
	}

	return &Server{
		mux:             mux,
		logger:          logger,
		srv:             srv,
		shutdownTimeout: shutdownTimeout,
	}
}

func (s *Server) Start(cancelFn context.CancelFunc) error {
	// Create channel for shutdown signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Channel to catch server errors
	serverError := make(chan error, 1)

	// Start server in goroutine
	go func() {
		s.logger.Printf("Starting server on port %s", s.srv.Addr)
		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverError <- err
		}
	}()

	// Wait for shutdown signal or server error
	select {
	case err := <-serverError:
		return fmt.Errorf("server error: %w", err)
	case sig := <-shutdown:
		s.logger.Printf("Shutdown signal received: %v", sig)
		cancelFn()

		return s.Shutdown()
	}
}

func (s *Server) Shutdown() error {
	// Create context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	// Attempt graceful shutdown
	if err := s.srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	s.logger.Printf("Server shutdown completed")

	return nil
}

// Mux Use to register routes
func (s *Server) Mux() *http.ServeMux {
	return s.mux
}
