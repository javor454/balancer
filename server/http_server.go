package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/javor454/balancer/auth"
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
func NewHttpServer(port int, shutdownTimeout time.Duration, whitelistedPaths []string, authBlacklistedPaths []string, proxyServerPool *ProxyServerPool, registerHandler *RegisterHandler, authHandler *auth.AuthHandler) *HttpServer {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/register", registerHandler.RegisterHandler)
	registerProxyServer(mux, proxyServerPool)

	wrappedMux := Chain(WithPanicRecovery(), WithLogging(), WithWhitelistedPaths(whitelistedPaths), WithConditionalAuth(authBlacklistedPaths, authHandler))(mux)

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

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// registerProxyServer registers the proxy server with load balancing
func registerProxyServer(mux *http.ServeMux, proxyServerPool *ProxyServerPool) {
	loadBalancer := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler, err := proxyServerPool.NextServer()
		if err != nil {
			http.Error(w, "No available backend servers", http.StatusServiceUnavailable)
			return
		}

		handler.ServeHTTP(w, r)
	})

	mux.Handle("/", loadBalancer)

	log.Print("Proxy server registered")
}
