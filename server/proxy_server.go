package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"time"
)

var (
	ErrNoHealthyServers = errors.New("no healthy servers found")
	ErrNoServers        = errors.New("no servers found")
	ErrNoCapacity       = errors.New("no capacity available")
)

// ProxyServerPool manages a pool of backend servers with health checks
type ProxyServerPool struct {
	strategy BalancingStrategy
}

// NewProxyServerPool creates a new pool of proxy servers with health checking
func NewProxyServerPool(ctx context.Context, urls []string, healthCheckInterval time.Duration, httpClient *http.Client, maxCapacity int, acquireCapacityTimeout time.Duration) (*ProxyServerPool, error) {
	servers := make([]*server, 0, len(urls))
	for _, v := range urls {
		server, err := newServer(v)
		if err != nil {
			return nil, err
		}
		server.startHealthCheck(ctx, healthCheckInterval, httpClient)
		servers = append(servers, server)
	}

	// Use RoundRobin strategy by default
	strategy := NewRoundRobinStrategy(servers, maxCapacity, acquireCapacityTimeout)

	return &ProxyServerPool{
		strategy: strategy,
	}, nil
}

// ServeHTTP implements http.Handler for the proxy server pool
func (p *ProxyServerPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler, err := p.strategy.NextServer(r.Context())
	if err != nil {
		http.Error(w, "No available backend servers", http.StatusServiceUnavailable)
		return
	}

	handler.ServeHTTP(w, r)

	p.strategy.ReleaseCapacity()
}

// GetMaxCapacity returns the maximum capacity from the strategy
func (p *ProxyServerPool) GetMaxCapacity() int {
	return p.strategy.GetMaxCapacity()
}

// GetAvailableCapacity returns the available capacity from the strategy
func (p *ProxyServerPool) GetAvailableCapacity() int {
	return p.strategy.GetAvailableCapacity()
}

// server represents a single backend server with health check status
type server struct {
	url          *url.URL
	alive        *atomic.Bool
	reverseProxy *httputil.ReverseProxy
}

// newServer creates a new backend server instance
func newServer(rawUrl string) (*server, error) {
	parsedUrl, err := url.Parse(rawUrl)
	if err != nil {
		return nil, fmt.Errorf("error parsing url: %w", err)
	}

	alive := &atomic.Bool{}
	alive.Store(true)

	reverseProxy := httputil.NewSingleHostReverseProxy(parsedUrl)
	reverseProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Proxy error: %v", err)
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
	}

	return &server{url: parsedUrl, alive: alive, reverseProxy: reverseProxy}, nil
}

// startHealthCheck begins periodic health checking of the server
func (s *server) startHealthCheck(ctx context.Context, healthCheckInterval time.Duration, httpClient *http.Client) {
	url := fmt.Sprintf("%s/health", s.url.String())

	go func() {
		log.Printf("Starting health check for %s", s.url.String())
		ticker := time.NewTicker(healthCheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Printf("Health check for %s stopped", s.url.String())
				return
			case <-ticker.C:
				resp, err := httpClient.Get(url)
				if err != nil || resp.StatusCode != http.StatusOK {
					log.Printf("Health check failed for %s", url)
					s.alive.Store(false)
				} else {
					log.Printf("Health check passed for %s", url)
					s.alive.Store(true)
				}
			}
		}
	}()
}

// IsAlive returns whether the server is currently considered healthy
func (s *server) IsAlive() bool {
	return s.alive.Load()
}
