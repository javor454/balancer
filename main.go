package main

import (
	"log"
	"net/http"

	"github.com/javor454/balancer/auth"
	"github.com/javor454/balancer/server"
)

func main() {
	httpConfig := server.NewDefaultHttpConfig()

	shutdownHandler := server.NewShutdownHandler()
	rootCtx := shutdownHandler.CreateRootCtxWithShutdown()

	httpClient := &http.Client{
		Timeout: httpConfig.RequestTimeout,
	}

	proxyServerPool, err := server.NewProxyServerPool(rootCtx, httpConfig.ProxyServers, httpConfig.HealthCheckInterval, httpClient, httpConfig.MaxCapacity, httpConfig.AcquireCapacityTimeout)
	if err != nil {
		log.Fatalf("Failed to create proxy server pool: %v", err)
	}

	authHandler := auth.NewAuthHandler(rootCtx)
	registerHandler := server.NewRegisterHandler(authHandler)


	httpServer := server.NewHttpServer(httpConfig.Port, httpConfig.ShutdownTimeout, httpConfig.WhitelistedPaths, httpConfig.AuthBlacklistedPaths, proxyServerPool, registerHandler, authHandler)
	httpServerErrChan := httpServer.Serve()

	var shutdownErr error
	select {
	case err := <-httpServerErrChan:
		// only one goroutine in this app, why do it so complicated
		shutdownHandler.SignalShutdown()
		shutdownErr = err
	case <-rootCtx.Done():
		log.Print("Received shutdown signal...")
	}

	if err := httpServer.GracefulShutdown(); err != nil {
		if shutdownErr == nil {
			shutdownErr = err
		}
	}

	if shutdownErr != nil {
		log.Fatalf("Shutdown error: %v", shutdownErr)
	}
	log.Print("Shutdown completed")
}
