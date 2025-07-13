package test

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/2gc-dev/cloudbridge-client/pkg/client"
	"github.com/2gc-dev/cloudbridge-client/pkg/protocol"
	"github.com/2gc-dev/cloudbridge-client/pkg/relay"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHandshakeProtocol tests the complete handshake protocol
func TestHandshakeProtocol(t *testing.T) {
	// Start mock relay server
	mockRelay := startMockRelay(t, "8085")
	defer mockRelay.Process.Kill()

	// Wait for server to start
	time.Sleep(2 * time.Second)

	// Test handshake with mock relay
	t.Run("MockRelayHandshake", func(t *testing.T) {
		// Use relay client directly for TCP connection
		relayClient := relay.NewClient(false, nil)
		defer relayClient.Close()

		err := relayClient.Connect("localhost", 8085)
		require.NoError(t, err)

		err = relayClient.Handshake("test-token")
		require.NoError(t, err)

		assert.True(t, relayClient.IsConnected())
	})

	// Test handshake with main relay server (port 3456)
	t.Run("MainRelayHandshake", func(t *testing.T) {
		if isRelayAvailable("localhost:3456") {
			// Use relay client directly for TCP connection to main relay
			relayClient := relay.NewClient(false, nil) // No TLS for now
			defer relayClient.Close()

			err := relayClient.Connect("localhost", 3456)
			require.NoError(t, err)

			err = relayClient.Handshake("test-token")
			require.NoError(t, err)

			assert.True(t, relayClient.IsConnected())
		} else {
			t.Skip("Main relay server not available on port 3456")
		}
	})

	// Test handshake with relay API (port 8082)
	t.Run("RelayAPIHandshake", func(t *testing.T) {
		if isRelayAvailable("localhost:8082") {
			// Use integrated client for API interaction
			clientCfg := client.DefaultConfig()
			clientCfg.TLSConfig = nil
			clientCfg.ProtocolOrder = []protocol.Protocol{2} // HTTP1
			client := client.NewIntegratedClient(clientCfg)
			defer client.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err := client.Connect(ctx, "localhost:8082")
			require.NoError(t, err)

			assert.True(t, client.IsConnected())
		} else {
			t.Skip("Relay API not available on port 8082")
		}
	})
}

// TestTunnelCreation tests tunnel creation and management
func TestTunnelCreation(t *testing.T) {
	mockRelay := startMockRelay(t, "8086")
	defer mockRelay.Process.Kill()

	time.Sleep(2 * time.Second)

	t.Run("CreateTunnel", func(t *testing.T) {
		// Use relay client directly for TCP connection
		relayClient := relay.NewClient(false, nil)
		defer relayClient.Close()

		err := relayClient.Connect("localhost", 8086)
		require.NoError(t, err)

		err = relayClient.Handshake("test-token")
		require.NoError(t, err)

		// Test tunnel creation
		tunnelID, err := relayClient.CreateTunnel(3389, "test-server", 3389)
		require.NoError(t, err)
		assert.NotEmpty(t, tunnelID)

		assert.True(t, relayClient.IsConnected())
	})
}

// TestErrorHandling tests error scenarios
func TestErrorHandling(t *testing.T) {
	t.Run("InvalidToken", func(t *testing.T) {
		mockRelay := startMockRelay(t, "8087")
		defer mockRelay.Process.Kill()
		time.Sleep(2 * time.Second)

		// Use relay client directly
		relayClient := relay.NewClient(false, nil)
		defer relayClient.Close()

		err := relayClient.Connect("localhost", 8087)
		require.NoError(t, err)

		// Try to authenticate with empty token
		err = relayClient.Handshake("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "authentication failed")
	})

	t.Run("ServerUnavailable", func(t *testing.T) {
		// Use relay client directly
		relayClient := relay.NewClient(false, nil)
		defer relayClient.Close()

		err := relayClient.Connect("localhost", 9999)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to connect")
	})
}

// TestProtocolMessages tests individual protocol messages
func TestProtocolMessages(t *testing.T) {
	mockRelay := startMockRelay(t, "8088")
	defer mockRelay.Process.Kill()
	time.Sleep(2 * time.Second)

	t.Run("HelloMessage", func(t *testing.T) {
		conn, err := net.Dial("tcp", "localhost:8088")
		require.NoError(t, err)
		defer conn.Close()

		// Read hello message
		reader := bufio.NewReader(conn)
		line, err := reader.ReadString('\n')
		require.NoError(t, err)

		var hello map[string]interface{}
		err = json.Unmarshal([]byte(strings.TrimSpace(line)), &hello)
		require.NoError(t, err)

		assert.Equal(t, "hello", hello["type"])
		assert.Equal(t, "1.0.0", hello["version"])
		assert.Contains(t, hello["features"], "tls")
		assert.Contains(t, hello["features"], "jwt")
		assert.Contains(t, hello["features"], "tunneling")
	})

	t.Run("AuthMessage", func(t *testing.T) {
		conn, err := net.Dial("tcp", "localhost:8088")
		require.NoError(t, err)
		defer conn.Close()

		reader := bufio.NewReader(conn)
		writer := bufio.NewWriter(conn)

		// Skip hello
		reader.ReadString('\n')

		// Send auth message
		authMsg := map[string]interface{}{
			"type":    "auth",
			"token":   "test-token",
			"version": "1.0.0",
			"client_info": map[string]interface{}{
				"os":   "linux",
				"arch": "amd64",
			},
		}

		data, _ := json.Marshal(authMsg)
		writer.Write(append(data, '\n'))
		writer.Flush()

		// Read auth response
		line, err := reader.ReadString('\n')
		require.NoError(t, err)

		var authResp map[string]interface{}
		err = json.Unmarshal([]byte(strings.TrimSpace(line)), &authResp)
		require.NoError(t, err)

		assert.Equal(t, "auth_response", authResp["type"])
		assert.Equal(t, "success", authResp["status"])
		assert.Equal(t, "test-client-001", authResp["client_id"])
	})
}

// TestBasicClientFunctionality tests basic client functionality without external dependencies
func TestBasicClientFunctionality(t *testing.T) {
	t.Run("RelayClientCreation", func(t *testing.T) {
		// Test relay client creation
		relayClient := relay.NewClient(false, nil)
		assert.NotNil(t, relayClient)
		assert.Equal(t, "2.0", relayClient.GetVersion())
		assert.Contains(t, relayClient.GetFeatures(), "tls")
		assert.Contains(t, relayClient.GetFeatures(), "heartbeat")
		
		// Test client is not connected initially
		assert.False(t, relayClient.IsConnected())
		
		relayClient.Close()
	})

	t.Run("IntegratedClientCreation", func(t *testing.T) {
		// Test integrated client creation
		clientCfg := client.DefaultConfig()
		clientCfg.MetricsEnabled = true
		clientCfg.HealthCheckEnabled = true
		
		integratedClient := client.NewIntegratedClient(clientCfg)
		assert.NotNil(t, integratedClient)
		assert.Equal(t, "2.0", integratedClient.GetVersion())
		assert.Contains(t, integratedClient.GetFeatures(), "tls")
		
		// Test client is not connected initially
		assert.False(t, integratedClient.IsConnected())
		
		integratedClient.Close()
	})

	t.Run("ProtocolEngine", func(t *testing.T) {
		// Test protocol engine functionality
		engine := protocol.NewProtocolEngine()
		assert.NotNil(t, engine)
		
		// Test protocol selection
		optimalProtocol := engine.GetOptimalProtocolForConnection(context.Background(), "localhost:8082")
		assert.NotNil(t, optimalProtocol)
		
		// Test stats
		stats := engine.GetStats()
		assert.NotNil(t, stats)
	})
}

// Helper functions

func startMockRelay(t *testing.T, port string) *exec.Cmd {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	// Поднимаемся на уровень выше (корень репозитория)
	repoRoot := cwd
	if strings.HasSuffix(cwd, "/test") {
		repoRoot = cwd[:len(cwd)-len("/test")]
	}
	mainGoPath := repoRoot + "/test/mock_relay/main.go"
	outputPath := repoRoot + "/test/mock_relay/mock_relay"

	// Build mock relay first
	buildCmd := exec.Command("go", "build", "-o", outputPath, mainGoPath)
	buildCmd.Dir = repoRoot
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	
	err = buildCmd.Run()
	require.NoError(t, err)
	
	// Start mock relay
	cmd := exec.Command(outputPath, port)
	cmd.Dir = repoRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	err = cmd.Start()
	require.NoError(t, err)
	
	return cmd
}



func isRelayAvailable(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// Benchmark tests

func BenchmarkHandshake(b *testing.B) {
	// Note: This benchmark is simplified since we can't easily start mock relay in benchmark
	clientCfg := client.DefaultConfig()
	clientCfg.TLSConfig = nil // Отключаем TLS для mock_relay
	clientCfg.ProtocolOrder = []protocol.Protocol{2} // Используем только HTTP1
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client := client.NewIntegratedClient(clientCfg)
		
		// Just test client creation and configuration
		_ = client.GetCurrentProtocol()
		_ = client.GetStats()
		
		client.Close()
	}
} 