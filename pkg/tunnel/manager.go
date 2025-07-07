package tunnel

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/2gc-dev/cloudbridge-client/pkg/interfaces"
)

// Tunnel represents a tunnel configuration
type Tunnel struct {
	ID         string
	LocalPort  int
	RemoteHost string
	RemotePort int
	Active     bool
	CreatedAt  time.Time
	LastUsed   time.Time
}

// Manager handles tunnel operations
type Manager struct {
	client  interfaces.ClientInterface
	tunnels map[string]*Tunnel
	mu      sync.RWMutex
}

// NewManager creates a new tunnel manager
func NewManager(client interfaces.ClientInterface) *Manager {
	return &Manager{
		client:  client,
		tunnels: make(map[string]*Tunnel),
	}
}

// RegisterTunnel registers a new tunnel
func (m *Manager) RegisterTunnel(tunnelID string, localPort int, remoteHost string, remotePort int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate tunnel parameters
	if err := m.validateTunnelParams(localPort, remoteHost, remotePort); err != nil {
		return fmt.Errorf("invalid tunnel parameters: %w", err)
	}

	// Check if tunnel already exists
	if _, exists := m.tunnels[tunnelID]; exists {
		return fmt.Errorf("tunnel %s already exists", tunnelID)
	}

	// Create tunnel
	tunnel := &Tunnel{
		ID:         tunnelID,
		LocalPort:  localPort,
		RemoteHost: remoteHost,
		RemotePort: remotePort,
		Active:     true,
		CreatedAt:  time.Now(),
		LastUsed:   time.Now(),
	}

	m.tunnels[tunnelID] = tunnel

	// Start tunnel proxy
	go m.startTunnelProxy(tunnel)

	return nil
}

// UnregisterTunnel removes a tunnel
func (m *Manager) UnregisterTunnel(tunnelID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tunnel, exists := m.tunnels[tunnelID]
	if !exists {
		return fmt.Errorf("tunnel %s not found", tunnelID)
	}

	tunnel.Active = false
	delete(m.tunnels, tunnelID)

	return nil
}

// GetTunnel returns a tunnel by ID
func (m *Manager) GetTunnel(tunnelID string) (*Tunnel, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tunnel, exists := m.tunnels[tunnelID]
	return tunnel, exists
}

// ListTunnels returns all registered tunnels
func (m *Manager) ListTunnels() []*Tunnel {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tunnels := make([]*Tunnel, 0, len(m.tunnels))
	for _, tunnel := range m.tunnels {
		tunnels = append(tunnels, tunnel)
	}

	return tunnels
}

// validateTunnelParams validates tunnel parameters
func (m *Manager) validateTunnelParams(localPort int, remoteHost string, remotePort int) error {
	// Validate local port
	if localPort <= 0 || localPort > 65535 {
		return fmt.Errorf("invalid local port: %d", localPort)
	}

	// Validate remote host
	if remoteHost == "" {
		return fmt.Errorf("remote host cannot be empty")
	}

	// Validate remote port
	if remotePort <= 0 || remotePort > 65535 {
		return fmt.Errorf("invalid remote port: %d", remotePort)
	}

	// Check if local port is already in use
	if m.isPortInUse(localPort) {
		return fmt.Errorf("local port %d is already in use", localPort)
	}

	return nil
}

// isPortInUse checks if a port is already in use
func (m *Manager) isPortInUse(port int) bool {
	// Check if any existing tunnel uses this port
	for _, tunnel := range m.tunnels {
		if tunnel.LocalPort == port && tunnel.Active {
			return true
		}
	}

	// Check if port is actually in use by trying to bind to it
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return true
	}
	if err := ln.Close(); err != nil {
		fmt.Printf("Error closing listener: %v\n", err)
	}
	return false
}

// startTunnelProxy starts a proxy for the tunnel
func (m *Manager) startTunnelProxy(tunnel *Tunnel) {
	// Listen on local port
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", tunnel.LocalPort))
	if err != nil {
		fmt.Printf("Failed to start tunnel %s: %v\n", tunnel.ID, err)
		return
	}
	defer listener.Close()

	fmt.Printf("Tunnel %s started: localhost:%d -> %s:%d\n", 
		tunnel.ID, tunnel.LocalPort, tunnel.RemoteHost, tunnel.RemotePort)

	for tunnel.Active {
		// Accept local connection
		localConn, err := listener.Accept()
		if err != nil {
			if tunnel.Active {
				fmt.Printf("Failed to accept connection for tunnel %s: %v\n", tunnel.ID, err)
			}
			continue
		}

		// Handle connection in goroutine
		go m.handleTunnelConnection(tunnel, localConn)
	}
}

// handleTunnelConnection handles a single tunnel connection
func (m *Manager) handleTunnelConnection(tunnel *Tunnel, localConn net.Conn) {
	defer localConn.Close()

	// Update last used time
	tunnel.LastUsed = time.Now()

	// Connect to remote host
	remoteConn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", tunnel.RemoteHost, tunnel.RemotePort))
	if err != nil {
		fmt.Printf("Failed to connect to remote host for tunnel %s: %v\n", tunnel.ID, err)
		return
	}
	defer remoteConn.Close()

	// Start bidirectional data transfer
	done := make(chan bool, 2)

	// Local to remote
	go func() {
		buffer := make([]byte, 4096)
		for {
			n, err := localConn.Read(buffer)
			if err != nil {
				break
			}
			if n > 0 {
				_, err = remoteConn.Write(buffer[:n])
				if err != nil {
					break
				}
			}
		}
		done <- true
	}()

	// Remote to local
	go func() {
		buffer := make([]byte, 4096)
		for {
			n, err := remoteConn.Read(buffer)
			if err != nil {
				break
			}
			if n > 0 {
				_, err = localConn.Write(buffer[:n])
				if err != nil {
					break
				}
			}
		}
		done <- true
	}()

	// Wait for both directions to complete
	<-done
	<-done
}

// GetTunnelStats returns statistics for all tunnels
func (m *Manager) GetTunnelStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_tunnels"] = len(m.tunnels)
	
	activeCount := 0
	for _, tunnel := range m.tunnels {
		if tunnel.Active {
			activeCount++
		}
	}
	stats["active_tunnels"] = activeCount

	return stats
} 