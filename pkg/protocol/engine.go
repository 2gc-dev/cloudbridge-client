package protocol

import (
	"context"
	"sync"
	"time"
)

// Protocol represents supported protocols
type Protocol int

const (
	QUIC Protocol = iota
	HTTP2
	HTTP1
)

// Protocol version constants
const (
	ProtocolVersionV1 = "1.0.0"
	ProtocolVersionV2 = "2.0"
)

// Protocol features
const (
	FeatureTLS         = "tls"
	FeatureHeartbeat   = "heartbeat"
	FeatureTunnelInfo  = "tunnel_info"
	FeatureMultiTenant = "multi_tenant"
	FeatureProxy       = "proxy"
	FeatureQUIC        = "quic"
	FeatureMetrics     = "metrics"
	FeatureJWT         = "jwt"
	FeatureTunneling   = "tunneling"
	FeatureHTTP2       = "http2"
)

// GetProtocolQUIC returns QUIC protocol
func GetProtocolQUIC() Protocol {
	return QUIC
}

// GetProtocolHTTP2 returns HTTP2 protocol
func GetProtocolHTTP2() Protocol {
	return HTTP2
}

// GetProtocolHTTP1 returns HTTP1 protocol
func GetProtocolHTTP1() Protocol {
	return HTTP1
}

func (p Protocol) String() string {
	switch p {
	case QUIC:
		return "quic"
	case HTTP2:
		return "http2"
	case HTTP1:
		return "http1"
	default:
		return "unknown"
	}
}

// GetProtocolDescription returns a human-readable description of the protocol
func (p Protocol) GetProtocolDescription() string {
	switch p {
	case QUIC:
		return "QUIC (UDP, RFC 9000) - Fast, 0-RTT, multiplexing"
	case HTTP2:
		return "HTTP/2 (TCP) - Multiplexing, reliable fallback"
	case HTTP1:
		return "HTTP/1.1 (TCP) - Legacy compatibility"
	default:
		return "Unknown protocol"
	}
}

// HelloMessage represents the hello handshake message
type HelloMessage struct {
	Type     string   `json:"type"`
	Version  string   `json:"version"`
	Features []string `json:"features"`
}

// NewHelloMessage creates a new hello message for v2.0
func NewHelloMessage() *HelloMessage {
	return &HelloMessage{
		Type:    "hello",
		Version: ProtocolVersionV2,
		Features: []string{
			FeatureTLS, FeatureHeartbeat, FeatureTunnelInfo,
			FeatureMultiTenant, FeatureProxy, FeatureQUIC, FeatureMetrics,
		},
	}
}

// NewHelloMessageV1 creates a new hello message for v1.0.0 (backward compatibility)
func NewHelloMessageV1() *HelloMessage {
	return &HelloMessage{
		Type:    "hello",
		Version: ProtocolVersionV1,
		Features: []string{
			FeatureTLS, FeatureJWT, FeatureTunneling, FeatureQUIC, FeatureHTTP2,
		},
	}
}

// AuthMessage represents the authentication message
type AuthMessage struct {
	Type      string                 `json:"type"`
	Token     string                 `json:"token"`
	TenantID  string                 `json:"tenant_id,omitempty"`
	Version   string                 `json:"version,omitempty"`
	ClientInfo map[string]interface{} `json:"client_info,omitempty"`
}

// NewAuthMessage creates a new auth message for v2.0
func NewAuthMessage(token, tenantID string) *AuthMessage {
	return &AuthMessage{
		Type:     "auth",
		Token:    token,
		TenantID: tenantID,
	}
}

// NewAuthMessageV1 creates a new auth message for v1.0.0 (backward compatibility)
func NewAuthMessageV1(token string, clientInfo map[string]interface{}) *AuthMessage {
	return &AuthMessage{
		Type:      "auth",
		Token:     token,
		Version:   ProtocolVersionV1,
		ClientInfo: clientInfo,
	}
}

// ProtocolEngine manages protocol selection and switching
type ProtocolEngine struct {
	preferredOrder []Protocol
	switchThreshold float64
	lastSwitch     time.Time
	switchCooldown time.Duration
	stats          map[Protocol]*ProtocolStats
	version        string
	features       []string
	mu             sync.RWMutex
	
	// Enhanced protocol selection
	autoSwitchEnabled bool
	performanceBased  bool
	networkConditions map[Protocol]bool
	lastNetworkCheck  time.Time
	networkCheckInterval time.Duration
}

// ProtocolStats tracks performance metrics for each protocol
type ProtocolStats struct {
	SuccessCount   int64
	FailureCount   int64
	TotalLatency   time.Duration
	LastUsed       time.Time
	IsAvailable    bool
	LastFailure    time.Time
	FailureReason  string
	AverageLatency time.Duration
	ConnectionTime  time.Duration
}

// NewProtocolEngine creates a new protocol engine
func NewProtocolEngine() *ProtocolEngine {
	return &ProtocolEngine{
		preferredOrder: []Protocol{QUIC, HTTP2, HTTP1}, // QUIC first, then HTTP/2, then HTTP/1.1
		switchThreshold: 0.8,
		switchCooldown: 30 * time.Second,
		stats: make(map[Protocol]*ProtocolStats),
		version: ProtocolVersionV2,
		features: []string{
			FeatureTLS, FeatureHeartbeat, FeatureTunnelInfo,
			FeatureMultiTenant, FeatureProxy, FeatureQUIC, FeatureMetrics,
		},
		autoSwitchEnabled: true,
		performanceBased: true,
		networkConditions: make(map[Protocol]bool),
		networkCheckInterval: 60 * time.Second,
	}
}

// NewProtocolEngineV1 creates a new protocol engine for v1.0.0 (backward compatibility)
func NewProtocolEngineV1() *ProtocolEngine {
	return &ProtocolEngine{
		preferredOrder: []Protocol{QUIC, HTTP2, HTTP1},
		switchThreshold: 0.8,
		switchCooldown: 30 * time.Second,
		stats: make(map[Protocol]*ProtocolStats),
		version: ProtocolVersionV1,
		features: []string{
			FeatureTLS, FeatureJWT, FeatureTunneling, FeatureQUIC, FeatureHTTP2,
		},
		autoSwitchEnabled: true,
		performanceBased: true,
		networkConditions: make(map[Protocol]bool),
		networkCheckInterval: 60 * time.Second,
	}
}

// GetVersion returns the protocol version
func (pe *ProtocolEngine) GetVersion() string {
	return pe.version
}

// GetFeatures returns the supported features
func (pe *ProtocolEngine) GetFeatures() []string {
	return pe.features
}

// SetPreferredOrder sets the preferred protocol order
func (pe *ProtocolEngine) SetPreferredOrder(order []Protocol) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.preferredOrder = order
}

// GetPreferredOrder returns the current preferred protocol order
func (pe *ProtocolEngine) GetPreferredOrder() []Protocol {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	
	order := make([]Protocol, len(pe.preferredOrder))
	copy(order, pe.preferredOrder)
	return order
}

// EnableAutoSwitch enables automatic protocol switching
func (pe *ProtocolEngine) EnableAutoSwitch() {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.autoSwitchEnabled = true
}

// DisableAutoSwitch disables automatic protocol switching
func (pe *ProtocolEngine) DisableAutoSwitch() {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.autoSwitchEnabled = false
}

// IsAutoSwitchEnabled returns true if auto switching is enabled
func (pe *ProtocolEngine) IsAutoSwitchEnabled() bool {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	return pe.autoSwitchEnabled
}

// GetBestProtocol returns the best available protocol based on performance and availability
func (pe *ProtocolEngine) GetBestProtocol() Protocol {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	// First, try to find a protocol that's available and performing well
	for _, protocol := range pe.preferredOrder {
		stats := pe.getOrCreateStats(protocol)
		
		// Check if protocol is available
		if !stats.IsAvailable {
			continue
		}
		
		// For protocols with enough data, check performance
		total := stats.SuccessCount + stats.FailureCount
		if total >= 3 {
			failureRate := pe.calculateFailureRate(stats)
			if failureRate <= pe.switchThreshold {
				return protocol
			}
		} else {
			// For protocols with insufficient data, assume they're good
			return protocol
		}
	}

	// If no protocol meets the criteria, return the first available one
	for _, protocol := range pe.preferredOrder {
		stats := pe.getOrCreateStats(protocol)
		if stats.IsAvailable {
			return protocol
		}
	}

	// Final fallback to HTTP/1.1 (most compatible)
	return HTTP1
}

// GetOptimalProtocolForConnection returns the optimal protocol for a new connection
func (pe *ProtocolEngine) GetOptimalProtocolForConnection(ctx context.Context, address string) Protocol {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	// Check network conditions if enough time has passed
	if time.Since(pe.lastNetworkCheck) > pe.networkCheckInterval {
		pe.updateNetworkConditions(ctx, address)
		pe.lastNetworkCheck = time.Now()
	}

	// Start with QUIC (fastest, 0-RTT, multiplexing)
	if pe.isProtocolSuitable(QUIC, address) {
		return QUIC
	}

	// Fallback to HTTP/2 (multiplexing, reliable)
	if pe.isProtocolSuitable(HTTP2, address) {
		return HTTP2
	}

	// Final fallback to HTTP/1.1 (legacy compatibility)
	return HTTP1
}

// isProtocolSuitable checks if a protocol is suitable for the given address
func (pe *ProtocolEngine) isProtocolSuitable(protocol Protocol, address string) bool {
	stats := pe.getOrCreateStats(protocol)
	
	// Check if protocol is marked as available
	if !stats.IsAvailable {
		return false
	}
	
	// Check network conditions
	if !pe.networkConditions[protocol] {
		return false
	}
	
	// Check recent performance
	total := stats.SuccessCount + stats.FailureCount
	if total >= 3 {
		failureRate := pe.calculateFailureRate(stats)
		if failureRate > pe.switchThreshold {
			return false
		}
	}
	
	return true
}

// updateNetworkConditions updates the network conditions for each protocol
func (pe *ProtocolEngine) updateNetworkConditions(ctx context.Context, address string) {
	// This would typically involve network probing
	// For now, we'll assume all protocols are available
	pe.networkConditions[QUIC] = true
	pe.networkConditions[HTTP2] = true
	pe.networkConditions[HTTP1] = true
}

// RecordSuccess records a successful operation for a protocol
func (pe *ProtocolEngine) RecordSuccess(protocol Protocol, latency time.Duration) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	
	stats := pe.getOrCreateStats(protocol)
	stats.SuccessCount++
	stats.TotalLatency += latency
	stats.LastUsed = time.Now()
	stats.IsAvailable = true
	
	// Update average latency
	total := stats.SuccessCount + stats.FailureCount
	if total > 0 {
		stats.AverageLatency = stats.TotalLatency / time.Duration(total)
	}
}

// RecordFailure records a failed operation for a protocol
func (pe *ProtocolEngine) RecordFailure(protocol Protocol, reason string) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	
	stats := pe.getOrCreateStats(protocol)
	stats.FailureCount++
	stats.LastUsed = time.Now()
	stats.LastFailure = time.Now()
	stats.FailureReason = reason
	
	// Mark protocol as unavailable if failure rate is high
	total := stats.SuccessCount + stats.FailureCount
	if total >= 5 {
		failureRate := float64(stats.FailureCount) / float64(total)
		if failureRate > pe.switchThreshold {
			stats.IsAvailable = false
		}
	}
}

// ShouldSwitchProtocol determines if we should switch protocols
func (pe *ProtocolEngine) ShouldSwitchProtocol(current Protocol) bool {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	if !pe.autoSwitchEnabled {
		return false
	}

	if time.Since(pe.lastSwitch) < pe.switchCooldown {
		return false
	}

	currentStats := pe.getOrCreateStats(current)
	total := currentStats.SuccessCount + currentStats.FailureCount
	
	if total < 5 {
		return false
	}

	failureRate := float64(currentStats.FailureCount) / float64(total)
	return failureRate > pe.switchThreshold
}

// GetNextProtocol returns the next protocol to try
func (pe *ProtocolEngine) GetNextProtocol(current Protocol) Protocol {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	for i, protocol := range pe.preferredOrder {
		if protocol == current {
			// Try next protocol in order
			for j := i + 1; j < len(pe.preferredOrder); j++ {
				nextProtocol := pe.preferredOrder[j]
				if stats, exists := pe.stats[nextProtocol]; exists && stats.IsAvailable {
					return nextProtocol
				}
			}
			break
		}
	}
	
	// Fallback to first available protocol
	return pe.GetBestProtocol()
}

// getOrCreateStats gets or creates stats for a protocol
func (pe *ProtocolEngine) getOrCreateStats(protocol Protocol) *ProtocolStats {
	if stats, exists := pe.stats[protocol]; exists {
		return stats
	}
	
	pe.stats[protocol] = &ProtocolStats{
		IsAvailable: true,
		SuccessCount: 0,
		FailureCount: 0,
		TotalLatency: 0,
		AverageLatency: 0,
	}
	return pe.stats[protocol]
}

// GetStats returns protocol statistics
func (pe *ProtocolEngine) GetStats() map[string]interface{} {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	
	result := make(map[string]interface{})
	
	for protocol, stats := range pe.stats {
		protocolName := protocol.String()
		result[protocolName] = map[string]interface{}{
			"success_count":   stats.SuccessCount,
			"failure_count":   stats.FailureCount,
			"total_latency":   stats.TotalLatency.String(),
			"average_latency": stats.AverageLatency.String(),
			"last_used":       stats.LastUsed,
			"is_available":    stats.IsAvailable,
			"failure_rate":    pe.calculateFailureRate(stats),
			"description":     protocol.GetProtocolDescription(),
			"last_failure":    stats.LastFailure,
			"failure_reason":  stats.FailureReason,
		}
	}
	
	return result
}

// calculateFailureRate calculates the failure rate for a protocol
func (pe *ProtocolEngine) calculateFailureRate(stats *ProtocolStats) float64 {
	total := stats.SuccessCount + stats.FailureCount
	if total == 0 {
		return 0
	}
	return float64(stats.FailureCount) / float64(total)
}

// GetProtocolRecommendation returns a recommendation for protocol selection
func (pe *ProtocolEngine) GetProtocolRecommendation() map[string]interface{} {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	
	recommendation := make(map[string]interface{})
	
	for _, protocol := range pe.preferredOrder {
		stats := pe.getOrCreateStats(protocol)
		protocolName := protocol.String()
		
		recommendation[protocolName] = map[string]interface{}{
			"recommended":     protocol == pe.GetBestProtocol(),
			"description":     protocol.GetProtocolDescription(),
			"is_available":    stats.IsAvailable,
			"failure_rate":    pe.calculateFailureRate(stats),
			"average_latency": stats.AverageLatency.String(),
			"priority":        pe.getProtocolPriority(protocol),
		}
	}
	
	return recommendation
}

// getProtocolPriority returns the priority of a protocol
func (pe *ProtocolEngine) getProtocolPriority(protocol Protocol) int {
	for i, p := range pe.preferredOrder {
		if p == protocol {
			return i + 1
		}
	}
	return 999 // Low priority if not in preferred order
}

// ResetStats resets all protocol statistics
func (pe *ProtocolEngine) ResetStats() {
	for _, protocol := range pe.preferredOrder {
		pe.stats[protocol] = &ProtocolStats{
			IsAvailable: true,
			SuccessCount: 0,
			FailureCount: 0,
			TotalLatency: 0,
		}
	}
}

// MarkProtocolAvailable marks a protocol as available
func (pe *ProtocolEngine) MarkProtocolAvailable(protocol Protocol) {
	stats := pe.getOrCreateStats(protocol)
	stats.IsAvailable = true
}

// MarkProtocolUnavailable marks a protocol as unavailable
func (pe *ProtocolEngine) MarkProtocolUnavailable(protocol Protocol) {
	stats := pe.getOrCreateStats(protocol)
	stats.IsAvailable = false
} 