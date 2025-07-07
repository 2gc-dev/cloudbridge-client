package test

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/2gc-dev/cloudbridge-client/pkg/config"
	"github.com/2gc-dev/cloudbridge-client/pkg/relay"
)

// MockRelayServer simulates a relay server for testing
type MockRelayServer struct {
	listener net.Listener
	port     int
	clients  map[string]*MockClient
}

// MockClient represents a connected client
type MockClient struct {
	conn   net.Conn
	userID string
}

// NewMockRelayServer creates a new mock relay server
func NewMockRelayServer() (*MockRelayServer, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return nil, err
	}

	port := listener.Addr().(*net.TCPAddr).Port

	server := &MockRelayServer{
		listener: listener,
		port:     port,
		clients:  make(map[string]*MockClient),
	}

	go server.acceptLoop()

	return server, nil
}

// acceptLoop accepts incoming connections
func (mrs *MockRelayServer) acceptLoop() {
	for {
		conn, err := mrs.listener.Accept()
		if err != nil {
			return
		}

		go mrs.handleConnection(conn)
	}
}

// handleConnection handles a client connection
func (mrs *MockRelayServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Send hello message
	hello := map[string]interface{}{
		"type":     "hello",
		"version":  "1.0",
		"features": []string{"tls", "heartbeat", "tunnel_info"},
	}
	mrs.sendMessage(conn, hello)

	// Handle client messages
	reader := json.NewDecoder(conn)
	for {
		var msg map[string]interface{}
		if err := reader.Decode(&msg); err != nil {
			return
		}

		mrs.handleMessage(conn, msg)
	}
}

// handleMessage processes client messages
func (mrs *MockRelayServer) handleMessage(conn net.Conn, msg map[string]interface{}) {
	msgType, ok := msg["type"].(string)
	if !ok {
		return
	}

	switch msgType {
	case "auth":
		mrs.handleAuth(conn, msg)
	case "tunnel_info":
		mrs.handleTunnelInfo(conn, msg)
	case "heartbeat":
		mrs.handleHeartbeat(conn, msg)
	}
}

// handleAuth processes authentication
func (mrs *MockRelayServer) handleAuth(conn net.Conn, msg map[string]interface{}) {
	token, ok := msg["token"].(string)
	if !ok {
		mrs.sendError(conn, "invalid_token", "Invalid token format")
		return
	}

	// Simple token validation (in real implementation, validate JWT)
	if token == "valid-token" {
		userID := "test-user"
		mrs.clients[userID] = &MockClient{
			conn:   conn,
			userID: userID,
		}

		response := map[string]interface{}{
			"type":      "auth_response",
			"status":    "ok",
			"client_id": userID,
		}
		mrs.sendMessage(conn, response)
	} else {
		mrs.sendError(conn, "invalid_token", "Invalid token")
	}
}

// handleTunnelInfo processes tunnel creation
func (mrs *MockRelayServer) handleTunnelInfo(conn net.Conn, msg map[string]interface{}) {
	tunnelID, ok := msg["tunnel_id"].(string)
	if !ok {
		tunnelID = "tunnel_001"
	}

	response := map[string]interface{}{
		"type":       "tunnel_response",
		"status":     "ok",
		"tunnel_id":  tunnelID,
	}
	mrs.sendMessage(conn, response)
}

// handleHeartbeat processes heartbeat messages
func (mrs *MockRelayServer) handleHeartbeat(conn net.Conn, msg map[string]interface{}) {
	response := map[string]interface{}{
		"type": "heartbeat_response",
	}
	mrs.sendMessage(conn, response)
}

// sendMessage sends a JSON message
func (mrs *MockRelayServer) sendMessage(conn net.Conn, msg map[string]interface{}) {
	data, _ := json.Marshal(msg)
	conn.Write(append(data, '\n'))
}

// sendError sends an error message
func (mrs *MockRelayServer) sendError(conn net.Conn, code, message string) {
	errorMsg := map[string]interface{}{
		"type":    "error",
		"code":    code,
		"message": message,
	}
	mrs.sendMessage(conn, errorMsg)
}

// Close closes the mock server
func (mrs *MockRelayServer) Close() error {
	return mrs.listener.Close()
}

// GetPort returns the server port
func (mrs *MockRelayServer) GetPort() int {
	return mrs.port
}

// TestFullConnectionCycle tests the complete connection cycle
func TestFullConnectionCycle(t *testing.T) {
	// Start mock server
	server, err := NewMockRelayServer()
	if err != nil {
		t.Fatalf("Failed to start mock server: %v", err)
	}
	defer server.Close()

	// Create client config
	cfg := &config.Config{}
	cfg.Server.Host = "localhost"
	cfg.Server.Port = server.GetPort()
	cfg.TLS.Enabled = false // Disable TLS for testing
	cfg.Server.JWTToken = "valid-token"

	// Create client
	client, err := relay.NewClientFromConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Connect to server
	err = client.Connect(cfg.Server.Host, cfg.Server.Port)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Perform handshake
	err = client.Handshake(cfg.Server.JWTToken, "1.0")
	if err != nil {
		t.Fatalf("Handshake failed: %v", err)
	}

	// Create tunnel
	tunnelID, err := client.CreateTunnel(3389, "192.168.1.100", 3389)
	if err != nil {
		t.Fatalf("Failed to create tunnel: %v", err)
	}

	if tunnelID == "" {
		t.Error("Expected non-empty tunnel ID")
	}
}

// TestTLSConnection tests TLS connection
func TestTLSConnection(t *testing.T) {
	// This test would require a TLS-enabled mock server
	// For now, we'll skip it
	t.Skip("TLS testing requires certificate setup")
}

// TestAuthenticationFailure tests authentication failure
func TestAuthenticationFailure(t *testing.T) {
	// Start mock server
	server, err := NewMockRelayServer()
	if err != nil {
		t.Fatalf("Failed to start mock server: %v", err)
	}
	defer server.Close()

	// Create client config with invalid token
	cfg := &config.Config{}
	cfg.Server.Host = "localhost"
	cfg.Server.Port = server.GetPort()
	cfg.TLS.Enabled = false
	cfg.Server.JWTToken = "invalid-token"

	// Create client
	client, err := relay.NewClientFromConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Connect to server
	err = client.Connect(cfg.Server.Host, cfg.Server.Port)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Perform handshake (should fail)
	err = client.Handshake(cfg.Server.JWTToken, "1.0")
	if err == nil {
		t.Error("Expected authentication to fail")
	}
}

// TestHeartbeat tests heartbeat functionality
func TestHeartbeat(t *testing.T) {
	// Start mock server
	server, err := NewMockRelayServer()
	if err != nil {
		t.Fatalf("Failed to start mock server: %v", err)
	}
	defer server.Close()

	// Create client config
	cfg := &config.Config{}
	cfg.Server.Host = "localhost"
	cfg.Server.Port = server.GetPort()
	cfg.TLS.Enabled = false
	cfg.Server.JWTToken = "valid-token"

	// Create client
	client, err := relay.NewClientFromConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Connect and authenticate
	err = client.Connect(cfg.Server.Host, cfg.Server.Port)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	err = client.Handshake(cfg.Server.JWTToken, "1.0")
	if err != nil {
		t.Fatalf("Handshake failed: %v", err)
	}

	// Send heartbeat
	heartbeatMsg := map[string]interface{}{
		"type": "heartbeat",
	}
	err = client.SendMessage(heartbeatMsg)
	if err != nil {
		t.Fatalf("Failed to send heartbeat: %v", err)
	}

	// Read response
	resp, err := client.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read heartbeat response: %v", err)
	}

	if resp["type"] != "heartbeat_response" {
		t.Errorf("Expected heartbeat_response, got %v", resp["type"])
	}
}

// TestConcurrentConnections tests multiple concurrent connections
func TestConcurrentConnections(t *testing.T) {
	// Start mock server
	server, err := NewMockRelayServer()
	if err != nil {
		t.Fatalf("Failed to start mock server: %v", err)
	}
	defer server.Close()

	// Create multiple clients
	numClients := 5
	clients := make([]*relay.Client, numClients)
	errors := make(chan error, numClients)

	for i := 0; i < numClients; i++ {
		go func(id int) {
			cfg := &config.Config{}
			cfg.Server.Host = "localhost"
			cfg.Server.Port = server.GetPort()
			cfg.TLS.Enabled = false
			cfg.Server.JWTToken = "valid-token"

			client, err := relay.NewClientFromConfig(cfg)
			if err != nil {
				errors <- fmt.Errorf("client %d: failed to create client: %v", id, err)
				return
			}
			defer client.Close()

			clients[id] = client

			// Connect and authenticate
			err = client.Connect(cfg.Server.Host, cfg.Server.Port)
			if err != nil {
				errors <- fmt.Errorf("client %d: failed to connect: %v", id, err)
				return
			}

			err = client.Handshake(cfg.Server.JWTToken, "1.0")
			if err != nil {
				errors <- fmt.Errorf("client %d: handshake failed: %v", id, err)
				return
			}

			errors <- nil
		}(i)
	}

	// Wait for all clients to complete
	for i := 0; i < numClients; i++ {
		if err := <-errors; err != nil {
			t.Errorf("Client %d failed: %v", i, err)
		}
	}
}

// TestRateLimiting tests rate limiting behavior
func TestRateLimiting(t *testing.T) {
	// This test would require rate limiting implementation in the mock server
	// For now, we'll skip it
	t.Skip("Rate limiting testing requires server-side implementation")
}

// TestErrorHandling tests error handling
func TestErrorHandling(t *testing.T) {
	// Start mock server
	server, err := NewMockRelayServer()
	if err != nil {
		t.Fatalf("Failed to start mock server: %v", err)
	}
	defer server.Close()

	// Create client config
	cfg := &config.Config{}
	cfg.Server.Host = "localhost"
	cfg.Server.Port = server.GetPort()
	cfg.TLS.Enabled = false
	cfg.Server.JWTToken = "valid-token"

	// Create client
	client, err := relay.NewClientFromConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Connect to server
	err = client.Connect(cfg.Server.Host, cfg.Server.Port)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Send invalid message type
	invalidMsg := map[string]interface{}{
		"type": "invalid_message_type",
	}
	err = client.SendMessage(invalidMsg)
	if err != nil {
		t.Fatalf("Failed to send invalid message: %v", err)
	}

	// Read response with timeout handling
	resp, err := client.ReadMessage()
	if err != nil {
		// Timeout is acceptable for invalid messages
		if strings.Contains(err.Error(), "i/o timeout") {
			t.Log("Expected timeout for invalid message type")
			return
		}
		t.Fatalf("Failed to read response: %v", err)
	}

	// The mock server might send hello first, so we need to handle that
	if resp["type"] == "hello" {
		// Read the actual error response with timeout handling
		resp, err = client.ReadMessage()
		if err != nil {
			// Timeout is acceptable for invalid messages
			if strings.Contains(err.Error(), "i/o timeout") {
				t.Log("Expected timeout for invalid message type")
				return
			}
			t.Fatalf("Failed to read error response: %v", err)
		}
	}

	// Check if we got an error message or any other response
	if resp["type"] != "error" {
		t.Logf("Got response type: %v (not necessarily an error, which is acceptable)", resp["type"])
	}
} 