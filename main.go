package main

import (
	"log"
	"net/http"
	"time"

	"github.com/javor454/balancer/server"
)

func main() {
	httpConfig := server.HttpConfig{
		Port:                8080,
		ShutdownTimeout:     10 * time.Second,
		RequestTimeout:      10 * time.Second,
		Whitelist:           []string{"/dummy"},
		ProxyServers:        []string{"http://wiremock1:8080", "http://wiremock2:8080", "http://wiremock3:8080"},
		HealthCheckInterval: 10 * time.Second,
	}

	shutdownHandler := server.NewShutdownHandler()
	rootCtx := shutdownHandler.CreateRootCtxWithShutdown()

	httpClient := &http.Client{
		Timeout: httpConfig.RequestTimeout,
	}

	proxyServerPool, err := server.NewProxyServerPool(rootCtx, httpConfig.ProxyServers, httpConfig.HealthCheckInterval, httpClient)
	if err != nil {
		log.Fatalf("Failed to create proxy server pool: %v", err)
	}

	httpServer := server.NewHttpServer(httpConfig.Port, httpConfig.ShutdownTimeout, httpConfig.Whitelist, proxyServerPool)
	httpServerErrChan := httpServer.Start()

	var shutdownErr error
	select {
	case err := <-httpServerErrChan:
		shutdownHandler.SignalShutdown()
		shutdownErr = err
	case <-rootCtx.Done():
		log.Print("Received shutdown signal...")
	}

	// Perform graceful shutdown
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
