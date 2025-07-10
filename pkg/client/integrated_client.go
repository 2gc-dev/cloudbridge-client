package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/2gc-dev/cloudbridge-client/pkg/circuitbreaker"
	"github.com/2gc-dev/cloudbridge-client/pkg/health"
	"github.com/2gc-dev/cloudbridge-client/pkg/metrics"
	"github.com/2gc-dev/cloudbridge-client/pkg/protocol"
	"github.com/2gc-dev/cloudbridge-client/pkg/relay"
	"github.com/prometheus/client_golang/prometheus"
)

// IntegratedClient represents a client that supports multiple protocols with circuit breaker
type IntegratedClient struct {
	protocolEngine *protocol.ProtocolEngine
	circuitBreaker *circuitbreaker.CircuitBreaker
	currentProtocol protocol.Protocol
	clients        map[protocol.Protocol]interface{}
	mu             sync.RWMutex
	config         *Config
	
	// New fields for v2.0
	metrics       *metrics.Metrics
	healthChecker *health.HealthChecker
	tenantID      string
	version       string
	features      []string
}

// Config holds integrated client configuration
type Config struct {
	TLSConfig        *tls.Config
	CircuitBreaker   *circuitbreaker.Config
	ProtocolOrder    []protocol.Protocol
	SwitchThreshold  float64
	ConnectTimeout   time.Duration
	RequestTimeout   time.Duration
	
	// New fields for v2.0
	TenantID         string
	Version          string
	Features         []string
	MetricsEnabled   bool
	HealthCheckEnabled bool
	HealthCheckConfig *health.Config
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		ProtocolOrder:   []protocol.Protocol{0, 1, 2}, // QUIC=0, HTTP2=1, HTTP1=2
		SwitchThreshold: 0.8,
		ConnectTimeout:  10 * time.Second,
		RequestTimeout:  30 * time.Second,
		CircuitBreaker:  circuitbreaker.DefaultConfig(),
		Version:         protocol.ProtocolVersionV2,
		Features:        []string{
			protocol.FeatureTLS, protocol.FeatureHeartbeat, protocol.FeatureTunnelInfo,
			protocol.FeatureMultiTenant, protocol.FeatureProxy, protocol.FeatureQUIC, protocol.FeatureMetrics,
		},
		MetricsEnabled:     true,
		HealthCheckEnabled: true,
		HealthCheckConfig:  health.DefaultConfig(),
	}
}

// NewIntegratedClient creates a new integrated client
func NewIntegratedClient(config *Config) *IntegratedClient {
	if config == nil {
		config = DefaultConfig()
	}

	// Create protocol engine based on version
	var protocolEngine *protocol.ProtocolEngine
	if config.Version == protocol.ProtocolVersionV1 {
		protocolEngine = protocol.NewProtocolEngineV1()
	} else {
		protocolEngine = protocol.NewProtocolEngine()
	}

	ic := &IntegratedClient{
		protocolEngine: protocolEngine,
		circuitBreaker: circuitbreaker.NewCircuitBreaker(config.CircuitBreaker),
		clients:        make(map[protocol.Protocol]interface{}),
		config:         config,
		tenantID:       config.TenantID,
		version:        config.Version,
		features:       config.Features,
	}

	// Initialize metrics if enabled
	if config.MetricsEnabled {
		ic.metrics = metrics.NewMetrics(prometheus.DefaultRegisterer)
		ic.metrics.SetClientVersion(config.Version)
	}

	// Initialize health checker if enabled
	if config.HealthCheckEnabled {
		ic.healthChecker = health.NewHealthChecker(config.HealthCheckConfig)
		ic.setupHealthChecks()
	}

	ic.protocolEngine.SetPreferredOrder(config.ProtocolOrder)

	return ic
}

// setupHealthChecks sets up default health checks
func (ic *IntegratedClient) setupHealthChecks() {
	if ic.healthChecker == nil {
		return
	}

	// Add connection health check
	ic.healthChecker.AddCheck("connection", health.CustomHealthCheck(
		"connection",
		"Check if client can establish connections",
		func(ctx context.Context) error {
			return ic.Ping()
		},
	))

	// Add protocol health check
	ic.healthChecker.AddCheck("protocol", health.CustomHealthCheck(
		"protocol",
		"Check if current protocol is working",
		func(ctx context.Context) error {
			stats := ic.protocolEngine.GetStats()
			for _, stat := range stats {
				if statMap, ok := stat.(map[string]interface{}); ok {
					if isAvailable, ok := statMap["is_available"].(bool); ok && isAvailable {
						return nil
					}
				}
			}
			return fmt.Errorf("no protocols available")
		},
	))

	// Start health checker
	ic.healthChecker.Start()
}

// SetTenantID sets the tenant ID for multi-tenancy support
func (ic *IntegratedClient) SetTenantID(tenantID string) {
	ic.mu.Lock()
	defer ic.mu.Unlock()
	ic.tenantID = tenantID
}

// GetTenantID returns the current tenant ID
func (ic *IntegratedClient) GetTenantID() string {
	ic.mu.RLock()
	defer ic.mu.RUnlock()
	return ic.tenantID
}

// GetVersion returns the protocol version
func (ic *IntegratedClient) GetVersion() string {
	return ic.version
}

// GetFeatures returns the supported features
func (ic *IntegratedClient) GetFeatures() []string {
	return ic.features
}

// GetMetrics returns the metrics instance
func (ic *IntegratedClient) GetMetrics() *metrics.Metrics {
	return ic.metrics
}

// GetHealthChecker returns the health checker instance
func (ic *IntegratedClient) GetHealthChecker() *health.HealthChecker {
	return ic.healthChecker
}

// Connect establishes a connection using the best available protocol
func (ic *IntegratedClient) Connect(ctx context.Context, address string) error {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	startTime := time.Now()
	defer func() {
		if ic.metrics != nil {
			ic.metrics.ObserveConnectionDuration(time.Since(startTime))
		}
	}()

	// Get optimal protocol for this connection using enhanced protocol engine
	optimalProtocol := ic.protocolEngine.GetOptimalProtocolForConnection(ctx, address)
	
	// Try the optimal protocol first
	if err := ic.tryConnect(ctx, address, optimalProtocol); err == nil {
		ic.currentProtocol = optimalProtocol
		latency := time.Since(startTime)
		ic.protocolEngine.RecordSuccess(optimalProtocol, latency)
		
		if ic.metrics != nil {
			ic.metrics.IncConnections()
			ic.metrics.ObserveProtocolLatency(optimalProtocol.String(), latency)
			ic.metrics.IncProtocolSuccess(optimalProtocol.String())
		}
		
		return nil
	} else {
		// Record failure with reason
		ic.protocolEngine.RecordFailure(optimalProtocol, err.Error())
		if ic.metrics != nil {
			ic.metrics.IncProtocolErrors(optimalProtocol.String())
		}
	}

	// If optimal protocol failed, try fallback protocols in order
	fallbackProtocols := ic.getFallbackProtocols(optimalProtocol)
	
	for _, protocol := range fallbackProtocols {
		if err := ic.tryConnect(ctx, address, protocol); err == nil {
			ic.currentProtocol = protocol
			latency := time.Since(startTime)
			ic.protocolEngine.RecordSuccess(protocol, latency)
			
			if ic.metrics != nil {
				ic.metrics.IncConnections()
				ic.metrics.ObserveProtocolLatency(protocol.String(), latency)
				ic.metrics.IncProtocolSuccess(protocol.String())
			}
			
			return nil
		} else {
			ic.protocolEngine.RecordFailure(protocol, err.Error())
			if ic.metrics != nil {
				ic.metrics.IncProtocolErrors(protocol.String())
			}
		}
	}

	if ic.metrics != nil {
		ic.metrics.IncRejectedConnections()
		ic.metrics.IncConnectionErrors("all_protocols_failed")
	}

	return fmt.Errorf("failed to connect using any protocol")
}

// getFallbackProtocols returns the list of fallback protocols in order of preference
func (ic *IntegratedClient) getFallbackProtocols(failedProtocol protocol.Protocol) []protocol.Protocol {
	// Get the preferred order from protocol engine
	preferredOrder := ic.protocolEngine.GetPreferredOrder()
	
	// Find the position of the failed protocol
	failedIndex := -1
	for i, protocol := range preferredOrder {
		if protocol == failedProtocol {
			failedIndex = i
			break
		}
	}
	
	// If failed protocol not found or it's the last one, return empty list
	if failedIndex == -1 || failedIndex >= len(preferredOrder)-1 {
		return []protocol.Protocol{}
	}
	
	// Return all protocols after the failed one
	return preferredOrder[failedIndex+1:]
}

// tryConnect attempts to connect using a specific protocol
func (ic *IntegratedClient) tryConnect(ctx context.Context, address string, protocol protocol.Protocol) error {
	ctx, cancel := context.WithTimeout(ctx, ic.config.ConnectTimeout)
	defer cancel()

	switch protocol {
	case 0: // QUIC
		return ic.connectQUIC(ctx, address)
	case 1: // HTTP2
		return ic.connectHTTP2(ctx, address)
	case 2: // HTTP1
		return ic.connectHTTP1(ctx, address)
	default:
		return fmt.Errorf("unsupported protocol: %s", protocol)
	}
}

// connectQUIC establishes a QUIC connection
func (ic *IntegratedClient) connectQUIC(ctx context.Context, address string) error {
	quicConfig := &protocol.QUICConfig{
		TLSConfig:        ic.config.TLSConfig,
		KeepAlive:        true,
		KeepAlivePeriod:  30 * time.Second,
		IdleTimeout:      60 * time.Second,
		HandshakeTimeout: 10 * time.Second,
	}

	quicClient := protocol.NewQUICClient(quicConfig)
	if err := quicClient.Connect(ctx, address); err != nil {
		return err
	}

	ic.clients[0] = quicClient // QUIC
	return nil
}

// connectHTTP2 establishes an HTTP/2 connection
func (ic *IntegratedClient) connectHTTP2(ctx context.Context, address string) error {
	http2Config := &protocol.HTTP2Config{
		TLSConfig:       ic.config.TLSConfig,
		Timeout:         ic.config.RequestTimeout,
		KeepAlive:       true,
		KeepAlivePeriod: 30 * time.Second,
		MaxIdleConns:    100,
		IdleConnTimeout: 90 * time.Second,
	}

	http2Client := protocol.NewHTTP2Client(http2Config)
	if err := http2Client.Connect(ctx, address); err != nil {
		return err
	}

	ic.clients[1] = http2Client // HTTP2
	return nil
}

// connectHTTP1 establishes an HTTP/1.1 connection (fallback)
func (ic *IntegratedClient) connectHTTP1(ctx context.Context, address string) error {
	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return err
	}
	
	// Create relay client based on version
	var client *relay.Client
	if ic.version == protocol.ProtocolVersionV1 {
		client = relay.NewClientV1(false, nil)
	} else {
		client = relay.NewClient(false, nil)
		client.SetTenantID(ic.tenantID)
	}
	
	if err := client.Connect(host, port); err != nil {
		return err
	}
	ic.clients[2] = client
	return nil
}

// Send sends data using the current protocol with circuit breaker protection
func (ic *IntegratedClient) Send(data []byte) error {
	return ic.circuitBreaker.Execute(context.Background(), func() error {
		return ic.sendWithCurrentProtocol(data)
	})
}

// sendWithCurrentProtocol sends data using the current protocol
func (ic *IntegratedClient) sendWithCurrentProtocol(data []byte) error {
	ic.mu.RLock()
	defer ic.mu.RUnlock()

	switch ic.currentProtocol {
	case 0: // QUIC
		if client, ok := ic.clients[0].(*protocol.QUICClient); ok {
			err := client.Send(data)
			if err == nil && ic.metrics != nil {
				ic.metrics.IncTunnelBytesToServer("quic_tunnel", int64(len(data)))
			}
			return err
		}
	case 1: // HTTP2
		if client, ok := ic.clients[1].(*protocol.HTTP2Client); ok {
			err := client.Send(data)
			if err == nil && ic.metrics != nil {
				ic.metrics.IncTunnelBytesToServer("http2_tunnel", int64(len(data)))
			}
			return err
		}
	}

	return fmt.Errorf("no client available for protocol: %s", ic.currentProtocol)
}

// Receive receives data using the current protocol
func (ic *IntegratedClient) Receive(buffer []byte) (int, error) {
	ic.mu.RLock()
	defer ic.mu.RUnlock()

	switch ic.currentProtocol {
	case 0: // QUIC
		if client, ok := ic.clients[0].(*protocol.QUICClient); ok {
			n, err := client.Receive(buffer)
			if err == nil && ic.metrics != nil {
				ic.metrics.IncTunnelBytesFromServer("quic_tunnel", int64(n))
			}
			return n, err
		}
	case 1: // HTTP2
		if client, ok := ic.clients[1].(*protocol.HTTP2Client); ok {
			n, err := client.Receive(buffer)
			if err == nil && ic.metrics != nil {
				ic.metrics.IncTunnelBytesFromServer("http2_tunnel", int64(n))
			}
			return n, err
		}
	}

	return 0, fmt.Errorf("no client available for protocol: %s", ic.currentProtocol)
}

// Close closes all connections
func (ic *IntegratedClient) Close() error {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	// Stop health checker
	if ic.healthChecker != nil {
		ic.healthChecker.Stop()
	}

	// Close all clients
	for _, client := range ic.clients {
		if closer, ok := client.(interface{ Close() error }); ok {
			closer.Close()
		}
	}

	if ic.metrics != nil {
		ic.metrics.DecConnections()
	}

	return nil
}

// IsConnected returns true if the client is connected
func (ic *IntegratedClient) IsConnected() bool {
	ic.mu.RLock()
	defer ic.mu.RUnlock()

	switch ic.currentProtocol {
	case 0: // QUIC
		if client, ok := ic.clients[0].(*protocol.QUICClient); ok {
			return client.IsConnected()
		}
	case 1: // HTTP2
		if client, ok := ic.clients[1].(*protocol.HTTP2Client); ok {
			return client.IsConnected()
		}
	case 2: // HTTP1
		if client, ok := ic.clients[2].(*relay.Client); ok {
			return client.IsConnected()
		}
	}

	return false
}

// GetCurrentProtocol returns the current protocol
func (ic *IntegratedClient) GetCurrentProtocol() protocol.Protocol {
	ic.mu.RLock()
	defer ic.mu.RUnlock()
	return ic.currentProtocol
}

// GetStats returns protocol statistics
func (ic *IntegratedClient) GetStats() map[string]interface{} {
	ic.mu.RLock()
	defer ic.mu.RUnlock()

	stats := ic.protocolEngine.GetStats()
	
	// Add client-specific stats
	stats["client"] = map[string]interface{}{
		"version":    ic.version,
		"tenant_id":  ic.tenantID,
		"features":   ic.features,
		"connected":  ic.IsConnected(),
	}

	// Add metrics summary if available
	if ic.metrics != nil {
		stats["metrics"] = ic.metrics.GetMetricsSummary()
	}

	// Add health status if available
	if ic.healthChecker != nil {
		stats["health"] = map[string]interface{}{
			"status":  ic.healthChecker.GetStatus(),
			"healthy": ic.healthChecker.IsHealthy(),
		}
	}

	return stats
}

// SwitchProtocol switches to a new protocol
func (ic *IntegratedClient) SwitchProtocol(newProtocol protocol.Protocol) error {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	if ic.currentProtocol == newProtocol {
		return nil
	}

	oldProtocol := ic.currentProtocol
	
	// Close current connection
	if closer, ok := ic.clients[ic.currentProtocol].(interface{ Close() error }); ok {
		closer.Close()
	}

	ic.currentProtocol = newProtocol

	if ic.metrics != nil {
		ic.metrics.IncProtocolSwitches(oldProtocol.String(), newProtocol.String())
	}

	return nil
}

// Ping sends a ping to test connectivity
func (ic *IntegratedClient) Ping() error {
	ic.mu.RLock()
	defer ic.mu.RUnlock()

	switch ic.currentProtocol {
	case 0: // QUIC
		if client, ok := ic.clients[0].(*protocol.QUICClient); ok {
			return client.Ping()
		}
	case 1: // HTTP2
		if client, ok := ic.clients[1].(*protocol.HTTP2Client); ok {
			return client.Ping()
		}
	}

	return fmt.Errorf("no client available for protocol: %s", ic.currentProtocol)
}

// AutoSwitchProtocol automatically switches to a better protocol if available
func (ic *IntegratedClient) AutoSwitchProtocol() error {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	// Check if we should switch protocols
	if !ic.protocolEngine.ShouldSwitchProtocol(ic.currentProtocol) {
		return nil // No need to switch
	}

	// Get the next best protocol
	nextProtocol := ic.protocolEngine.GetNextProtocol(ic.currentProtocol)
	if nextProtocol == ic.currentProtocol {
		return nil // No better protocol available
	}

	// Try to switch to the better protocol
	return ic.SwitchProtocol(nextProtocol)
}

// GetProtocolRecommendation returns a recommendation for protocol selection
func (ic *IntegratedClient) GetProtocolRecommendation() map[string]interface{} {
	return ic.protocolEngine.GetProtocolRecommendation()
}

// EnableAutoProtocolSwitching enables automatic protocol switching
func (ic *IntegratedClient) EnableAutoProtocolSwitching() {
	ic.protocolEngine.EnableAutoSwitch()
}

// DisableAutoProtocolSwitching disables automatic protocol switching
func (ic *IntegratedClient) DisableAutoProtocolSwitching() {
	ic.protocolEngine.DisableAutoSwitch()
}

// IsAutoProtocolSwitchingEnabled returns true if auto switching is enabled
func (ic *IntegratedClient) IsAutoProtocolSwitchingEnabled() bool {
	return ic.protocolEngine.IsAutoSwitchEnabled()
} 