package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics represents client metrics compatible with Relay Server v2.0
type Metrics struct {
	mu sync.RWMutex

	// Connection metrics
	connectionsTotal      prometheus.Counter
	rejectedConnections   prometheus.Counter
	connectionErrors      *prometheus.CounterVec
	activeConnections     prometheus.Gauge
	connectionDuration    prometheus.Histogram

	// Protocol metrics
	protocolLatency       *prometheus.HistogramVec
	protocolErrors        *prometheus.CounterVec
	protocolSwitches      *prometheus.CounterVec
	protocolSuccess       *prometheus.CounterVec

	// Tunnel metrics
	tunnelCreations       prometheus.Counter
	tunnelClosures        prometheus.Counter
	tunnelDuration        prometheus.Histogram
	tunnelBytesFromServer *prometheus.CounterVec
	tunnelBytesToServer   *prometheus.CounterVec
	tunnelErrors          *prometheus.CounterVec
	tunnelStatus          *prometheus.GaugeVec

	// Authentication metrics
	authAttempts          prometheus.Counter
	authFailures          prometheus.Counter
	authDuration          prometheus.Histogram

	// Heartbeat metrics
	heartbeatsTotal       prometheus.Counter
	heartbeatErrors       prometheus.Counter
	heartbeatLatency      prometheus.Histogram

	// Tenant metrics (for multi-tenancy)
	tenantConnections     *prometheus.GaugeVec
	tenantTunnels         *prometheus.GaugeVec
	tenantBandwidth       *prometheus.CounterVec
	tenantErrors          *prometheus.CounterVec

	// Client info metrics
	clientVersion         prometheus.Gauge
	clientUptime          prometheus.Gauge
	clientMemoryUsage     prometheus.Gauge

	// Local counters for current values
	activeConnectionsCount int64
	activeTunnelsCount     int64
	startTime              time.Time
}

// NewMetrics creates new client metrics
func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		startTime: time.Now(),
		connectionsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "client_connections_total",
			Help: "Total number of connections",
		}),
		rejectedConnections: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "client_rejected_connections_total",
			Help: "Total number of rejected connections",
		}),
		connectionErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "client_connection_errors_total",
			Help: "Total number of connection errors by type",
		}, []string{"error_type"}),
		activeConnections: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "client_active_connections",
			Help: "Number of active connections",
		}),
		connectionDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "client_connection_duration_seconds",
			Help:    "Connection duration in seconds",
			Buckets: []float64{0.1, 0.5, 1.0, 5.0, 10.0, 30.0, 60.0},
		}),
		protocolLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "client_protocol_latency_seconds",
			Help:    "Protocol latency in seconds",
			Buckets: prometheus.DefBuckets,
		}, []string{"protocol"}),
		protocolErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "client_protocol_errors_total",
			Help: "Total number of protocol errors",
		}, []string{"protocol"}),
		protocolSwitches: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "client_protocol_switches_total",
			Help: "Total number of protocol switches",
		}, []string{"from", "to"}),
		protocolSuccess: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "client_protocol_success_total",
			Help: "Total number of successful protocol operations",
		}, []string{"protocol"}),
		tunnelCreations: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "client_tunnel_creations_total",
			Help: "Total number of tunnel creations",
		}),
		tunnelClosures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "client_tunnel_closures_total",
			Help: "Total number of tunnel closures",
		}),
		tunnelDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "client_tunnel_duration_seconds",
			Help:    "Tunnel duration in seconds",
			Buckets: []float64{1.0, 5.0, 10.0, 30.0, 60.0, 300.0, 600.0, 3600.0},
		}),
		tunnelBytesFromServer: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "client_tunnel_bytes_from_server_total",
			Help: "Total bytes received from server",
		}, []string{"tunnel_id"}),
		tunnelBytesToServer: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "client_tunnel_bytes_to_server_total",
			Help: "Total bytes sent to server",
		}, []string{"tunnel_id"}),
		tunnelErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "client_tunnel_errors_total",
			Help: "Total number of tunnel errors",
		}, []string{"tunnel_id", "error_type"}),
		tunnelStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "client_tunnel_status",
			Help: "Tunnel status (1=active, 0=inactive)",
		}, []string{"tunnel_id"}),
		authAttempts: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "client_auth_attempts_total",
			Help: "Total number of authentication attempts",
		}),
		authFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "client_auth_failures_total",
			Help: "Total number of authentication failures",
		}),
		authDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "client_auth_duration_seconds",
			Help:    "Authentication duration in seconds",
			Buckets: prometheus.DefBuckets,
		}),
		heartbeatsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "client_heartbeats_total",
			Help: "Total number of heartbeats",
		}),
		heartbeatErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "client_heartbeat_errors_total",
			Help: "Total number of heartbeat errors",
		}),
		heartbeatLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "client_heartbeat_latency_seconds",
			Help:    "Heartbeat latency in seconds",
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1.0},
		}),
		tenantConnections: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "client_tenant_connections",
			Help: "Number of connections per tenant",
		}, []string{"tenant_id"}),
		tenantTunnels: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "client_tenant_tunnels",
			Help: "Number of tunnels per tenant",
		}, []string{"tenant_id"}),
		tenantBandwidth: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "client_tenant_bandwidth_bytes_total",
			Help: "Total bandwidth usage per tenant",
		}, []string{"tenant_id"}),
		tenantErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "client_tenant_errors_total",
			Help: "Total errors per tenant",
		}, []string{"tenant_id"}),
		clientVersion: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "client_version_info",
			Help: "Client version information",
		}),
		clientUptime: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "client_uptime_seconds",
			Help: "Client uptime in seconds",
		}),
		clientMemoryUsage: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "client_memory_usage_bytes",
			Help: "Client memory usage in bytes",
		}),
	}

	// Register all metrics
	reg.MustRegister(
		m.connectionsTotal,
		m.rejectedConnections,
		m.connectionErrors,
		m.activeConnections,
		m.connectionDuration,
		m.protocolLatency,
		m.protocolErrors,
		m.protocolSwitches,
		m.protocolSuccess,
		m.tunnelCreations,
		m.tunnelClosures,
		m.tunnelDuration,
		m.tunnelBytesFromServer,
		m.tunnelBytesToServer,
		m.tunnelErrors,
		m.tunnelStatus,
		m.authAttempts,
		m.authFailures,
		m.authDuration,
		m.heartbeatsTotal,
		m.heartbeatErrors,
		m.heartbeatLatency,
		m.tenantConnections,
		m.tenantTunnels,
		m.tenantBandwidth,
		m.tenantErrors,
		m.clientVersion,
		m.clientUptime,
		m.clientMemoryUsage,
	)

	return m
}

// Connection metrics
func (m *Metrics) IncConnections() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connectionsTotal.Inc()
	m.activeConnectionsCount++
	m.activeConnections.Set(float64(m.activeConnectionsCount))
}

func (m *Metrics) DecConnections() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeConnectionsCount--
	if m.activeConnectionsCount < 0 {
		m.activeConnectionsCount = 0
	}
	m.activeConnections.Set(float64(m.activeConnectionsCount))
}

func (m *Metrics) IncRejectedConnections() {
	m.rejectedConnections.Inc()
}

func (m *Metrics) IncConnectionErrors(errorType string) {
	m.connectionErrors.WithLabelValues(errorType).Inc()
}

func (m *Metrics) ObserveConnectionDuration(duration time.Duration) {
	m.connectionDuration.Observe(duration.Seconds())
}

// Protocol metrics
func (m *Metrics) ObserveProtocolLatency(protocol string, duration time.Duration) {
	m.protocolLatency.WithLabelValues(protocol).Observe(duration.Seconds())
}

func (m *Metrics) IncProtocolErrors(protocol string) {
	m.protocolErrors.WithLabelValues(protocol).Inc()
}

func (m *Metrics) IncProtocolSwitches(from, to string) {
	m.protocolSwitches.WithLabelValues(from, to).Inc()
}

func (m *Metrics) IncProtocolSuccess(protocol string) {
	m.protocolSuccess.WithLabelValues(protocol).Inc()
}

// Tunnel metrics
func (m *Metrics) IncTunnelCreations() {
	m.tunnelCreations.Inc()
}

func (m *Metrics) IncTunnelClosures() {
	m.tunnelClosures.Inc()
}

func (m *Metrics) ObserveTunnelDuration(duration time.Duration) {
	m.tunnelDuration.Observe(duration.Seconds())
}

func (m *Metrics) IncTunnelBytesFromServer(tunnelID string, bytes int64) {
	m.tunnelBytesFromServer.WithLabelValues(tunnelID).Add(float64(bytes))
}

func (m *Metrics) IncTunnelBytesToServer(tunnelID string, bytes int64) {
	m.tunnelBytesToServer.WithLabelValues(tunnelID).Add(float64(bytes))
}

func (m *Metrics) IncTunnelErrors(tunnelID, errorType string) {
	m.tunnelErrors.WithLabelValues(tunnelID, errorType).Inc()
}

func (m *Metrics) SetTunnelStatus(tunnelID string, active bool) {
	status := 0.0
	if active {
		status = 1.0
	}
	m.tunnelStatus.WithLabelValues(tunnelID).Set(status)
}

// Authentication metrics
func (m *Metrics) IncAuthAttempts() {
	m.authAttempts.Inc()
}

func (m *Metrics) IncAuthFailures() {
	m.authFailures.Inc()
}

func (m *Metrics) ObserveAuthDuration(duration time.Duration) {
	m.authDuration.Observe(duration.Seconds())
}

// Heartbeat metrics
func (m *Metrics) IncHeartbeats() {
	m.heartbeatsTotal.Inc()
}

func (m *Metrics) IncHeartbeatErrors() {
	m.heartbeatErrors.Inc()
}

func (m *Metrics) ObserveHeartbeatLatency(duration time.Duration) {
	m.heartbeatLatency.Observe(duration.Seconds())
}

// Tenant metrics
func (m *Metrics) SetTenantConnections(tenantID string, count int) {
	m.tenantConnections.WithLabelValues(tenantID).Set(float64(count))
}

func (m *Metrics) SetTenantTunnels(tenantID string, count int) {
	m.tenantTunnels.WithLabelValues(tenantID).Set(float64(count))
}

func (m *Metrics) IncTenantBandwidth(tenantID string, bytes int64) {
	m.tenantBandwidth.WithLabelValues(tenantID).Add(float64(bytes))
}

func (m *Metrics) IncTenantErrors(tenantID string) {
	m.tenantErrors.WithLabelValues(tenantID).Inc()
}

// Client info metrics
func (m *Metrics) SetClientVersion(version string) {
	// For simplicity, we'll use a hash of the version string
	// In a real implementation, you might want to use a different approach
	versionHash := float64(len(version))
	m.clientVersion.Set(versionHash)
}

func (m *Metrics) UpdateClientUptime() {
	uptime := time.Since(m.startTime).Seconds()
	m.clientUptime.Set(uptime)
}

func (m *Metrics) SetClientMemoryUsage(bytes int64) {
	m.clientMemoryUsage.Set(float64(bytes))
}

// GetActiveConnections returns the current number of active connections
func (m *Metrics) GetActiveConnections() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activeConnectionsCount
}

// GetActiveTunnels returns the current number of active tunnels
func (m *Metrics) GetActiveTunnels() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activeTunnelsCount
}

// SetActiveTunnels sets the number of active tunnels
func (m *Metrics) SetActiveTunnels(count int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeTunnelsCount = count
}

// GetMetricsSummary returns a summary of all metrics
func (m *Metrics) GetMetricsSummary() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"connections": map[string]interface{}{
			"total":     m.activeConnectionsCount,
			"uptime":    time.Since(m.startTime).String(),
		},
		"tunnels": map[string]interface{}{
			"active": m.activeTunnelsCount,
		},
		"protocols": map[string]interface{}{
			"supported": []string{"quic", "http2", "http1"},
		},
	}
} 