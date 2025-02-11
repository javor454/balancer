package balancer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

const (
	clientsEndpoint = "/clients"
	jobsEndpoint    = "/jobs"
	statusQueued    = "queued"
	statusPending   = "pending"
)

func setupTestBalancer(t *testing.T) (*Balancer, *httptest.Server) {
	logger := log.New(os.Stdout, "[TEST] ", log.Ldate|log.Ltime|log.Lshortfile)

	config := &Config{
		Strategy:        SingleClient,
		Capacity:        3,
		SessionTimeout:  Duration{Duration: 50 * time.Millisecond},
		ShutdownTimeout: Duration{Duration: 50 * time.Millisecond},
		JobDuration:     Duration{Duration: 10 * time.Millisecond},
		CleanupInterval: Duration{Duration: 20 * time.Millisecond},
	}

	b, err := NewBalancer(context.Background(), config, logger)
	if err != nil {
		t.Fatalf("Failed to create balancer: %v", err)
	}

	mux := http.NewServeMux()
	b.RegisterHandlers(mux)
	srv := httptest.NewServer(mux)

	return b, srv
}

func TestClientRegistrationWorkflow(t *testing.T) {
	_, srv := setupTestBalancer(t)
	defer srv.Close()

	// Test registering first client
	resp, err := http.Post(fmt.Sprintf("%s%s", srv.URL, clientsEndpoint), "application/json", nil)
	if err != nil {
		t.Fatalf("Failed to register client: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	resp.Body.Close()

	if result["client_id"] == "" {
		t.Error("Expected client_id in response")
	}

	// Test registering second client (should be queued)
	resp2, err := http.Post(fmt.Sprintf("%s%s", srv.URL, clientsEndpoint), "application/json", nil)
	if err != nil {
		t.Fatalf("Failed to register second client: %v", err)
	}
	if resp2.StatusCode != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, resp2.StatusCode)
	}

	var result2 map[string]string
	if err := json.NewDecoder(resp2.Body).Decode(&result2); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	resp2.Body.Close()

	clientID2 := result2["client_id"]
	resp3, err := http.Get(fmt.Sprintf("%s%s/%s", srv.URL, clientsEndpoint, clientID2))
	if err != nil {
		t.Fatalf("Failed to get client status: %v", err)
	}
	defer resp3.Body.Close()

	var statusResult map[string]interface{}
	if err := json.NewDecoder(resp3.Body).Decode(&statusResult); err != nil {
		t.Fatalf("Failed to decode status response: %v", err)
	}

	if statusResult["status"] != statusQueued {
		t.Errorf("Expected status %q, got %v", statusQueued, statusResult["status"])
	}
	if statusResult["position"].(float64) != 1 {
		t.Errorf("Expected position 1, got %v", statusResult["position"])
	}
}

func TestJobWorkflow(t *testing.T) {
	_, srv := setupTestBalancer(t)
	defer srv.Close()

	// Register a client
	resp, err := http.Post(fmt.Sprintf("%s%s", srv.URL, clientsEndpoint), "application/json", nil)
	if err != nil {
		t.Fatalf("Failed to register client: %v", err)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	resp.Body.Close()

	clientID := result["client_id"]

	// Register a job
	resp2, err := http.Post(fmt.Sprintf("%s%s/%s/jobs", srv.URL, clientsEndpoint, clientID), "application/json", nil)
	if err != nil {
		t.Fatalf("Failed to register job: %v", err)
	}
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp2.StatusCode)
	}

	var jobResult map[string]string
	if err := json.NewDecoder(resp2.Body).Decode(&jobResult); err != nil {
		t.Fatalf("Failed to decode job response: %v", err)
	}
	resp2.Body.Close()

	jobID := jobResult["job_id"]
	if jobID == "" {
		t.Error("Expected job_id in response")
	}

	// Check job status
	resp3, err := http.Get(fmt.Sprintf("%s%s/%s", srv.URL, jobsEndpoint, jobID))
	if err != nil {
		t.Fatalf("Failed to get job status: %v", err)
	}
	defer resp3.Body.Close()

	var statusResult map[string]string
	if err := json.NewDecoder(resp3.Body).Decode(&statusResult); err != nil {
		t.Fatalf("Failed to decode status response: %v", err)
	}

	if statusResult["status"] != statusPending {
		t.Errorf("Expected status %q, got %v", statusPending, statusResult["status"])
	}
}

func TestConcurrentClients(t *testing.T) {
	_, srv := setupTestBalancer(t)
	defer srv.Close()

	// Register multiple clients concurrently
	clientCount := 5
	done := make(chan bool, clientCount)

	for i := 0; i < clientCount; i++ {
		go func() {
			resp, err := http.Post(fmt.Sprintf("%s%s", srv.URL, clientsEndpoint), "application/json", nil)
			if err != nil {
				t.Errorf("Failed to register client: %v", err)
				done <- false
				return
			}
			resp.Body.Close()
			done <- true
		}()
	}

	// Wait for all registrations
	for i := 0; i < clientCount; i++ {
		success := <-done
		if !success {
			t.Error("Client registration failed")
		}
	}
}

func TestClientTimeout(t *testing.T) {
	_, srv := setupTestBalancer(t)
	defer srv.Close()

	// Register a client
	resp, err := http.Post(fmt.Sprintf("%s%s", srv.URL, clientsEndpoint), "application/json", nil)
	if err != nil {
		t.Fatalf("Failed to register client: %v", err)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	resp.Body.Close()

	clientID := result["client_id"]

	// Wait for timeout (now just 50ms + a small buffer)
	time.Sleep(60 * time.Millisecond)

	// Try to register a job with timed-out client
	resp2, err := http.Post(fmt.Sprintf("%s%s/%s/jobs", srv.URL, clientsEndpoint, clientID), "application/json", nil)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp2.Body.Close()

	var errorResponse map[string]interface{}
	if err := json.NewDecoder(resp2.Body).Decode(&errorResponse); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if resp2.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, resp2.StatusCode)
	}

	if errorResponse["error"] == nil {
		t.Error("Expected error message in response")
	}
}

func TestServerAtCapacity(t *testing.T) {
	_, srv := setupTestBalancer(t)
	defer srv.Close()

	// Register a client
	resp, err := http.Post(fmt.Sprintf("%s%s", srv.URL, clientsEndpoint), "application/json", nil)
	if err != nil {
		t.Fatalf("Failed to register client: %v", err)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	resp.Body.Close()

	clientID := result["client_id"]

	// Register jobs until capacity (3) is reached
	for i := 0; i < 3; i++ {
		resp, err := http.Post(fmt.Sprintf("%s%s/%s/jobs", srv.URL, clientsEndpoint, clientID), "application/json", nil)
		if err != nil {
			t.Fatalf("Failed to register job %d: %v", i+1, err)
		}
		resp.Body.Close()
	}

	// Try to register one more job
	resp2, err := http.Post(fmt.Sprintf("%s%s/%s/jobs", srv.URL, clientsEndpoint, clientID), "application/json", nil)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp2.Body.Close()

	var errorResponse map[string]interface{}
	if err := json.NewDecoder(resp2.Body).Decode(&errorResponse); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if resp2.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, resp2.StatusCode)
	}

	if errorResponse["error"] == nil {
		t.Error("Expected error message in response")
	}
}

func TestDeregisterClient(t *testing.T) {
	_, srv := setupTestBalancer(t)
	defer srv.Close()

	// Register first client (will be active)
	resp, err := http.Post(fmt.Sprintf("%s%s", srv.URL, clientsEndpoint), "application/json", nil)
	if err != nil {
		t.Fatalf("Failed to register first client: %v", err)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	resp.Body.Close()

	firstClientID := result["client_id"]

	// Register second client (will be queued)
	resp2, err := http.Post(fmt.Sprintf("%s%s", srv.URL, clientsEndpoint), "application/json", nil)
	if err != nil {
		t.Fatalf("Failed to register second client: %v", err)
	}

	var result2 map[string]string
	if err := json.NewDecoder(resp2.Body).Decode(&result2); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	resp2.Body.Close()

	secondClientID := result2["client_id"]

	// Verify second client is queued
	resp3, err := http.Get(fmt.Sprintf("%s%s/%s", srv.URL, clientsEndpoint, secondClientID))
	if err != nil {
		t.Fatalf("Failed to get second client status: %v", err)
	}

	var statusResult map[string]interface{}
	if err := json.NewDecoder(resp3.Body).Decode(&statusResult); err != nil {
		t.Fatalf("Failed to decode status response: %v", err)
	}
	resp3.Body.Close()

	if statusResult["status"] != statusQueued {
		t.Errorf("Expected second client status %q, got %v", statusQueued, statusResult["status"])
	}

	// Deregister first client
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s%s/%s", srv.URL, clientsEndpoint, firstClientID), nil)
	if err != nil {
		t.Fatalf("Failed to create delete request: %v", err)
	}

	resp4, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to deregister first client: %v", err)
	}
	resp4.Body.Close()

	if resp4.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d for deregister, got %d", http.StatusOK, resp4.StatusCode)
	}

	// Verify second client is now active
	resp5, err := http.Get(fmt.Sprintf("%s%s/%s", srv.URL, clientsEndpoint, secondClientID))
	if err != nil {
		t.Fatalf("Failed to get second client status after deregister: %v", err)
	}
	defer resp5.Body.Close()

	var finalStatus map[string]interface{}
	if err := json.NewDecoder(resp5.Body).Decode(&finalStatus); err != nil {
		t.Fatalf("Failed to decode final status response: %v", err)
	}

	if finalStatus["status"] != StatusActive {
		t.Errorf("Expected second client status %q after deregister, got %v", StatusActive, finalStatus["status"])
	}

	// Try to deregister non-existent client
	req2, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s%s/%s", srv.URL, clientsEndpoint, uuid.New().String()), nil)
	if err != nil {
		t.Fatalf("Failed to create delete request: %v", err)
	}

	resp6, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp6.Body.Close()

	if resp6.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status %d for non-existent client, got %d", http.StatusNotFound, resp6.StatusCode)
	}
}

func TestQueuedClientCannotAddJob(t *testing.T) {
	_, srv := setupTestBalancer(t)
	defer srv.Close()

	// Register first client (will be active)
	resp, err := http.Post(fmt.Sprintf("%s%s", srv.URL, clientsEndpoint), "application/json", nil)
	if err != nil {
		t.Fatalf("Failed to register first client: %v", err)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	resp.Body.Close()

	// Register second client (will be queued)
	resp2, err := http.Post(fmt.Sprintf("%s%s", srv.URL, clientsEndpoint), "application/json", nil)
	if err != nil {
		t.Fatalf("Failed to register second client: %v", err)
	}

	var result2 map[string]string
	if err := json.NewDecoder(resp2.Body).Decode(&result2); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	resp2.Body.Close()

	queuedClientID := result2["client_id"]

	// Verify second client is queued
	resp3, err := http.Get(fmt.Sprintf("%s%s/%s", srv.URL, clientsEndpoint, queuedClientID))
	if err != nil {
		t.Fatalf("Failed to get queued client status: %v", err)
	}

	var statusResult map[string]interface{}
	if err := json.NewDecoder(resp3.Body).Decode(&statusResult); err != nil {
		t.Fatalf("Failed to decode status response: %v", err)
	}
	resp3.Body.Close()

	if statusResult["status"] != statusQueued {
		t.Errorf("Expected client status %q, got %v", statusQueued, statusResult["status"])
	}

	// Try to add job with queued client
	resp4, err := http.Post(fmt.Sprintf("%s%s/%s/jobs", srv.URL, clientsEndpoint, queuedClientID), "application/json", nil)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp4.Body.Close()

	var errorResponse map[string]interface{}
	if err := json.NewDecoder(resp4.Body).Decode(&errorResponse); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if resp4.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, resp4.StatusCode)
	}

	if errorResponse["error"] != "Client is not active or has timed out" {
		t.Errorf("Expected error message %q, got %v", "Client is not active or has timed out", errorResponse["error"])
	}
}

func TestQueuedClientDeregister(t *testing.T) {
	_, srv := setupTestBalancer(t)
	defer srv.Close()

	// Register first client (will be active)
	resp, err := http.Post(fmt.Sprintf("%s%s", srv.URL, clientsEndpoint), "application/json", nil)
	if err != nil {
		t.Fatalf("Failed to register first client: %v", err)
	}
	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	resp.Body.Close()
	firstClientID := result["client_id"]

	// Register second and third clients (will be queued)
	var queuedClients []string
	for i := 0; i < 2; i++ {
		resp, err := http.Post(fmt.Sprintf("%s%s", srv.URL, clientsEndpoint), "application/json", nil)
		if err != nil {
			t.Fatalf("Failed to register client %d: %v", i+2, err)
		}
		var result map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
		resp.Body.Close()
		queuedClients = append(queuedClients, result["client_id"])
	}

	// Deregister second client (first in queue)
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s%s/%s", srv.URL, clientsEndpoint, queuedClients[0]), nil)
	if err != nil {
		t.Fatalf("Failed to create delete request: %v", err)
	}
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to deregister queued client: %v", err)
	}
	resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d for deregister, got %d", http.StatusOK, resp2.StatusCode)
	}

	// Verify first client is still active
	resp3, err := http.Get(fmt.Sprintf("%s%s/%s", srv.URL, clientsEndpoint, firstClientID))
	if err != nil {
		t.Fatalf("Failed to get first client status: %v", err)
	}
	var statusResult map[string]interface{}
	if err := json.NewDecoder(resp3.Body).Decode(&statusResult); err != nil {
		t.Fatalf("Failed to decode status response: %v", err)
	}
	resp3.Body.Close()

	if statusResult["status"] != StatusActive {
		t.Errorf("Expected first client status %q, got %v", StatusActive, statusResult["status"])
	}

	// Verify third client moved up in queue
	resp4, err := http.Get(fmt.Sprintf("%s%s/%s", srv.URL, clientsEndpoint, queuedClients[1]))
	if err != nil {
		t.Fatalf("Failed to get third client status: %v", err)
	}
	var queuedStatus map[string]interface{}
	if err := json.NewDecoder(resp4.Body).Decode(&queuedStatus); err != nil {
		t.Fatalf("Failed to decode status response: %v", err)
	}
	resp4.Body.Close()

	if queuedStatus["status"] != statusQueued {
		t.Errorf("Expected third client status %q, got %v", statusQueued, queuedStatus["status"])
	}
	if queuedStatus["position"].(float64) != 1 {
		t.Errorf("Expected position 1, got %v", queuedStatus["position"])
	}
}

func TestJobCompletion(t *testing.T) {
	_, srv := setupTestBalancer(t)
	defer srv.Close()

	// Register client and add job
	resp, err := http.Post(fmt.Sprintf("%s%s", srv.URL, clientsEndpoint), "application/json", nil)
	if err != nil {
		t.Fatalf("Failed to register client: %v", err)
	}
	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	resp.Body.Close()
	clientID := result["client_id"]

	resp2, err := http.Post(fmt.Sprintf("%s%s/%s/jobs", srv.URL, clientsEndpoint, clientID), "application/json", nil)
	if err != nil {
		t.Fatalf("Failed to register job: %v", err)
	}
	var jobResult map[string]string
	if err := json.NewDecoder(resp2.Body).Decode(&jobResult); err != nil {
		t.Fatalf("Failed to decode job response: %v", err)
	}
	resp2.Body.Close()
	jobID := jobResult["job_id"]

	// Small wait to ensure job completes (much shorter now)
	time.Sleep(20 * time.Millisecond)

	// Check job status
	resp3, err := http.Get(fmt.Sprintf("%s%s/%s", srv.URL, jobsEndpoint, jobID))
	if err != nil {
		t.Fatalf("Failed to get job status: %v", err)
	}
	defer resp3.Body.Close()

	var statusResult map[string]string
	if err := json.NewDecoder(resp3.Body).Decode(&statusResult); err != nil {
		t.Fatalf("Failed to decode status response: %v", err)
	}

	if statusResult["status"] != StatusFinished {
		t.Errorf("Expected job status %q, got %q", StatusFinished, statusResult["status"])
	}
}
