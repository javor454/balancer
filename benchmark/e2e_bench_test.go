package benchmark

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/javor454/balancer/server"
)

// BenchmarkE2EThroughput tests the throughput at different concurrency levels
func BenchmarkE2EThroughput(b *testing.B) {
	// Suppress logs
	originalOutput := log.Writer()
	log.SetOutput(io.Discard)
	defer log.SetOutput(originalOutput)

	const (
		backendCount           = 3
		backendLatency         = 10 * time.Millisecond
		capacityLimit          = 100
		acquireCapacityTimeout = 10 * time.Second
		clientRequestTimeout   = 30 * time.Second
		healthCheckInterval    = 5 * time.Second
	)

	ctx := context.Background()

	backends, urls := NewTestBackendPool(backendCount, backendLatency)
	defer CleanupBackends(backends)

	httpClient := &http.Client{Timeout: clientRequestTimeout}
	proxyServerPool, err := server.NewProxyServerPool(ctx, urls, healthCheckInterval, httpClient, capacityLimit, acquireCapacityTimeout)
	if err != nil {
		b.Fatalf("Failed to create proxy server pool: %v", err)
	}

	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handler, err := proxyServerPool.NextServer(r.Context())
			if err != nil {
				http.Error(w, "No available backend servers", http.StatusServiceUnavailable)
				return
			}

			handler.ServeHTTP(w, r)
		}))

	defer ts.Close()

	for _, concurrentRequests := range []int{1, 10, 25} {
		b.Run(fmt.Sprintf("Concurrency-%d", concurrentRequests), func(b *testing.B) {
			b.ResetTimer() // Reset the timer to exclude setup time

			// total number of iterations / number of concurrent requests
			requestsPerGoroutine := max(b.N/concurrentRequests, 1)

			var wg sync.WaitGroup
			errCount := atomic.Int32{}
			successCount := atomic.Int32{}

			for range concurrentRequests {
				wg.Add(1)
				go func() {
					defer wg.Done()

					client := &http.Client{
						Timeout: clientRequestTimeout,
					}

					for range requestsPerGoroutine {
						req, _ := http.NewRequest("GET", ts.URL+"/test", nil)
						resp, err := client.Do(req)
						if err != nil {
							errCount.Add(1)
							continue
						}

						if resp.StatusCode == http.StatusOK {
							successCount.Add(1)
						} else if resp.StatusCode == http.StatusServiceUnavailable {
							errCount.Add(1)
						} else {
							errCount.Add(1)
						}

						resp.Body.Close()
					}
				}()
			}

			wg.Wait()

			b.ReportMetric(float64(successCount.Load())/float64(b.N), "success-rate")
			b.ReportMetric(float64(errCount.Load())/float64(b.N), "error-rate")
		})
	}
}
