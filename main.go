package main

import (
	"context"
	"log"
	"os"

	"github.com/javor454/balancer/internal/balancer"
	"github.com/javor454/balancer/server"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create custom logger with file name and line number
	logger := log.New(os.Stdout, "[API] ", log.Ldate|log.Ltime|log.Lshortfile)

	config, err := balancer.LoadConfig()
	if err != nil {
		logger.Fatalf("Failed to load config: %s", err)
	}
	logger.Printf("Config loaded: %+v", config)

	server := server.NewServer(logger, config.Port, config.ShutdownTimeout.Duration)

	balanc, err := balancer.NewBalancer(ctx, config, logger)
	if err != nil {
		logger.Fatalf("Failed to create balancer: %s", err)
	}

	balanc.RegisterHandlers(server.Mux())

	// Start server and handle potential shutdown error
	if err := server.Start(cancel); err != nil {
		logger.Fatalf("Server shutdown: %s", err)
	}
}
