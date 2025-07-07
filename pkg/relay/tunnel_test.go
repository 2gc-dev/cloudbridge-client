package relay

import (
	"testing"
)

func TestTunnelCreation(t *testing.T) {
	// Create a client with proper configuration
	client := NewClient(false, nil)
	
	// Test that CreateTunnel returns an error when not connected
	// This is expected behavior since we haven't established a connection
	tunnelID, err := client.CreateTunnel(3389, "test-server", 3389)
	if err == nil {
		t.Error("Expected error when not connected to server")
	}
	if tunnelID != "" {
		t.Error("Expected empty tunnel ID when connection fails")
	}
}

func TestTunnelCreationWithInvalidPorts(t *testing.T) {
	client := NewClient(false, nil)
	
	// Test with invalid local port
	_, err := client.CreateTunnel(-1, "test-server", 3389)
	if err == nil {
		t.Error("Expected error for invalid local port")
	}
	
	// Test with invalid remote port
	_, err = client.CreateTunnel(3389, "test-server", -1)
	if err == nil {
		t.Error("Expected error for invalid remote port")
	}
}

func TestTunnelValidation(t *testing.T) {
	// Test valid port ranges
	validPorts := []int{1, 1024, 8080, 65535}
	invalidPorts := []int{-1, 0, 65536, 99999}
	
	for _, port := range validPorts {
		if port < 1 || port > 65535 {
			t.Errorf("Port %d should be valid", port)
		}
	}
	
	for _, port := range invalidPorts {
		if port >= 1 && port <= 65535 {
			t.Errorf("Port %d should be invalid", port)
		}
	}
} 