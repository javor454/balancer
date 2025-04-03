package benchmark

import (
	"net/http"
	"net/http/httptest"
	"time"
)

// TestBackend represents a simulated backend server
type TestBackend struct {
	server  *httptest.Server
	latency time.Duration
}

// NewTestBackendPool creates a pool of test backends
func NewTestBackendPool(count int, latency time.Duration) ([]*TestBackend, []string) {
	backends := make([]*TestBackend, count)
	urls := make([]string, count)

	for i := 0; i < count; i++ {
		backend := &TestBackend{
			latency: latency,
		}

		backend.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(backend.latency) // Simulate work

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}))

		backends[i] = backend
		urls[i] = backend.server.URL
	}

	return backends, urls
}

// CleanupBackends closes all test backend servers
func CleanupBackends(backends []*TestBackend) {
	for _, b := range backends {
		b.server.Close()
	}
}
