package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

// HttpServer represents the HTTP server with routing and shutdown capabilities
type HttpServer struct {
	srv             *http.Server
	shutdownTimeout time.Duration
}

// Start begins listening for HTTP requests and returns an error channel
func (s *HttpServer) Start() chan error {
	serverError := make(chan error, 1)

	go func() {
		log.Printf("Starting Http server on port %s", s.srv.Addr)
		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Http server error: %v", err)
			serverError <- err
		}
	}()

	log.Print("Http server started")

	return serverError
}

// GracefulShutdown attempts to gracefully shut down the server
func (s *HttpServer) GracefulShutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	if err := s.srv.Shutdown(ctx); err != nil {
		log.Printf("Http server shutdown failed: %v", err)
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	log.Printf("Http server shutdown completed")

	return nil
}

// NewHttpServer creates and configures a new HTTP server instance with logging, panic recovery, and URL whitelisting
func NewHttpServer(port int, shutdownTimeout time.Duration, whitelist []string, proxyServerPool *ProxyServerPool) *HttpServer {
	mux := http.NewServeMux()

	registerHealthCheck(mux)
	registerProxyServer(mux, proxyServerPool)

	wrappedMux := Chain(WithLogging(), WithPanicRecovery(), WithURLWhitelist(whitelist))(mux)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: wrappedMux,
	}

	h := &HttpServer{
		srv:             srv,
		shutdownTimeout: shutdownTimeout,
	}

	return h
}

// registerHealthCheck adds a health check endpoint
func registerHealthCheck(mux *http.ServeMux) {
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	log.Print("Health check registered")
}

// registerProxyServer configures the proxy server with load balancing
func registerProxyServer(mux *http.ServeMux, proxyServerPool *ProxyServerPool) {
	loadBalancer := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler, err := proxyServerPool.NextServer()
		if err != nil {
			http.Error(w, "No available backend servers", http.StatusServiceUnavailable)
			return
		}

		handler.ServeHTTP(w, r)
	})

	// Register the /dummy endpoint specifically
	// mux.Handle("/dummy", loadBalancer)

	// Also register a catch-all handler for other paths
	// The whitelist middleware will filter out unwanted paths
	mux.Handle("/", loadBalancer)

	log.Print("Proxy server registered")
}
