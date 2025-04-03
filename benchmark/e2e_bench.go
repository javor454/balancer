package benchmark

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/javor454/balancer/server"
)

func BenchmarkThroughput(b *testing.B) {
	rootCtx := context.Background()
	// Setup test server and load balancer
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer ts.Close()

	pool, err := server.NewProxyServerPool(rootCtx, []string{ts.URL}, 5*time.Second, &http.Client{}, 100, 10*time.Second)
	if err != nil {
		b.Fatal(err)
	}

	handler := pool.GetHandler()
	server := httptest.NewServer(handler)
	defer server.Close()

	// Run benchmark with different concurrency levels
	for _, concurrency := range []int{1, 10, 50, 100, 200} {
		b.Run(fmt.Sprintf("Concurrency-%d", concurrency), func(b *testing.B) {
			var wg sync.WaitGroup
			requestsPerGoroutine := b.N / concurrency

			b.ResetTimer()
			for i := 0; i < concurrency; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					client := &http.Client{}
					for j := 0; j < requestsPerGoroutine; j++ {
						req, _ := http.NewRequest("GET", server.URL, nil)
						client.Do(req)
					}
				}()
			}
			wg.Wait()
		})
	}
}
