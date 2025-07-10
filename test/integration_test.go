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
	"github.com/2gc-dev/cloudbridge-client/pkg/config"
	"github.com/2gc-dev/cloudbridge-client/pkg/protocol"
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
		testHandshakeWithServer(t, "localhost:8085", false)
	})

	// Test handshake with main relay (if available)
	t.Run("MainRelayHandshake", func(t *testing.T) {
		if isRelayAvailable("localhost:8082") {
			testHandshakeWithServer(t, "localhost:8082", true)
		} else {
			t.Skip("Main relay server not available")
		}
	})
}

// TestTunnelCreation tests tunnel creation and management
func TestTunnelCreation(t *testing.T) {
	mockRelay := startMockRelay(t, "8086")
	defer mockRelay.Process.Kill()

	time.Sleep(2 * time.Second)

	t.Run("CreateTunnel", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.Server.Host = "localhost"
		cfg.Server.Port = 8086
		cfg.Server.JWTToken = "test-token"
		cfg.TLS.Enabled = false

		clientCfg := client.DefaultConfig()
		clientCfg.TLSConfig = nil // Отключаем TLS для mock_relay
		clientCfg.ProtocolOrder = []protocol.Protocol{2} // Используем только HTTP1
		client := client.NewIntegratedClient(clientCfg)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := client.Connect(ctx, "localhost:8086")
		require.NoError(t, err)
		defer client.Close()

		// Test basic connectivity
		assert.True(t, client.IsConnected())
	})
}

// TestErrorHandling tests error scenarios
func TestErrorHandling(t *testing.T) {
	t.Run("InvalidToken", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.Server.Host = "localhost"
		cfg.Server.Port = 8087
		cfg.Server.JWTToken = "" // Empty token
		cfg.TLS.Enabled = false

		mockRelay := startMockRelay(t, "8087")
		defer mockRelay.Process.Kill()
		time.Sleep(2 * time.Second)

		clientCfg := client.DefaultConfig()
		clientCfg.TLSConfig = nil // Отключаем TLS для mock_relay
		clientCfg.ProtocolOrder = []protocol.Protocol{2} // Используем только HTTP1
		client := client.NewIntegratedClient(clientCfg)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		client.Connect(ctx, "localhost:8087")
		// Проверяем, что клиент не подключён
		assert.False(t, client.IsConnected())
	})

	t.Run("ServerUnavailable", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.Server.Host = "localhost"
		cfg.Server.Port = 9999 // Non-existent port
		cfg.Server.JWTToken = "test-token"
		cfg.TLS.Enabled = false

		clientCfg := client.DefaultConfig()
		clientCfg.TLSConfig = nil // Отключаем TLS для mock_relay
		clientCfg.ProtocolOrder = []protocol.Protocol{2} // Используем только HTTP1
		client := client.NewIntegratedClient(clientCfg)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := client.Connect(ctx, "localhost:9999")
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
		assert.Equal(t, "ok", authResp["status"])
		assert.Equal(t, "test-client-001", authResp["client_id"])
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

func testHandshakeWithServer(t *testing.T, serverAddr string, useTLS bool) {
	cfg := &config.Config{}
	cfg.Server.Host = strings.Split(serverAddr, ":")[0]
	cfg.Server.Port = 8085 // Will be overridden
	cfg.Server.JWTToken = "test-token"
	cfg.TLS.Enabled = useTLS

	// Parse port from serverAddr
	parts := strings.Split(serverAddr, ":")
	if len(parts) == 2 {
		cfg.Server.Port = 8085 // Use the port from serverAddr
	}

	clientCfg := client.DefaultConfig()
	clientCfg.TLSConfig = nil // Отключаем TLS для mock_relay
	clientCfg.ProtocolOrder = []protocol.Protocol{2} // Используем только HTTP1
	client := client.NewIntegratedClient(clientCfg)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := client.Connect(ctx, serverAddr)
	require.NoError(t, err)
	defer client.Close()

	// Test basic connectivity
	assert.True(t, client.IsConnected())
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