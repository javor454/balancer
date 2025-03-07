package server

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type ShutdownHandler struct {
	shutdownChan chan os.Signal
	ctx          context.Context
	cancel       context.CancelFunc
	once         sync.Once
}

func NewShutdownHandler() *ShutdownHandler {
	ctx, cancel := context.WithCancel(context.Background())
	return &ShutdownHandler{
		shutdownChan: make(chan os.Signal, 1),
		ctx:          ctx,
		cancel:       cancel,
	}
}

// CreateRootCtxWithShutdown Creates a context which is cancelled on SIGINT or SIGTERM.
func (s *ShutdownHandler) CreateRootCtxWithShutdown() context.Context {
	log.Print("Setting up shutdown handler...")
	signal.Notify(s.shutdownChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-s.shutdownChan
		log.Printf("Received shutdown signal: %v", sig)
		s.triggerShutdown()
	}()

	return s.ctx
}

func (s *ShutdownHandler) triggerShutdown() {
	s.once.Do(func() {
		s.cancel()
	})
}

func (s *ShutdownHandler) SignalShutdown() {
	s.triggerShutdown()
}
