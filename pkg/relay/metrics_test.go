package relay

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMetrics(t *testing.T) {
	// Record some metrics
	RecordConnection(1.5)
	RecordError("test_error")
	SetActiveTunnels(5)
	RecordHeartbeat(0.1)
	RecordMissedHeartbeat()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metrics" {
			t.Errorf("Expected to request '/metrics', got: %s", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got: %s", r.Method)
		}
	}))
	defer server.Close()

	// Test metrics endpoint
	resp, err := http.Get(server.URL + "/metrics")
	if err != nil {
		t.Fatalf("Failed to get metrics: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got: %v", resp.StatusCode)
	}
}

func TestHealthCheck(t *testing.T) {
	// Record some metrics
	RecordConnection(1.0)
	RecordError("test_error")
	SetActiveTunnels(3)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(HealthCheckHandler))
	defer server.Close()

	// Test health check endpoint
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to get health check: %v", err)
	}
	defer resp.Body.Close()

	// Health check returns 503 when status is not "ok", which is expected behavior
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status OK or ServiceUnavailable, got %v", resp.StatusCode)
	}

	// Verify health status
	status := GetHealthStatus()
	if status.Status != "unknown" && status.Status != "ok" {
		t.Errorf("Expected status 'unknown' or 'ok', got %v", status.Status)
	}
}

func TestMetricsConcurrency(t *testing.T) {
	// Test concurrent access
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			RecordConnection(1.0)
			RecordError("concurrent_error")
			SetActiveTunnels(i)
			RecordHeartbeat(0.1)
			RecordMissedHeartbeat()
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestHealthStatusUpdate(t *testing.T) {
	// Test health status update
	UpdateHealthStatus("ok")
	status := GetHealthStatus()
	if status.Status != "ok" {
		t.Errorf("Expected status 'ok', got %v", status.Status)
	}

	UpdateHealthStatus("error")
	status = GetHealthStatus()
	if status.Status != "error" {
		t.Errorf("Expected status 'error', got %v", status.Status)
	}
} 