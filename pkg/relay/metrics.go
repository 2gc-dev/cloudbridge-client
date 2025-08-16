package relay

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Connection metrics
	connectionsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "relay_connections_total",
		Help: "Total number of connections",
	})

	connectionDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "relay_connection_duration_seconds",
		Help:    "Connection duration in seconds",
		Buckets: prometheus.DefBuckets,
	})

	// Error metrics
	errorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "relay_errors_total",
		Help: "Total number of errors by code",
	}, []string{"code"})

	// Tunnel metrics
	activeTunnels = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "relay_active_tunnels",
		Help: "Number of active tunnels",
	})

	// Heartbeat metrics
	heartbeatLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "relay_heartbeat_latency_seconds",
		Help:    "Heartbeat latency in seconds",
		Buckets: prometheus.DefBuckets,
	})

	missedHeartbeats = promauto.NewCounter(prometheus.CounterOpts{
		Name: "relay_missed_heartbeats_total",
		Help: "Total number of missed heartbeats",
	})
)

// RecordConnection records a new connection
func RecordConnection(duration float64) {
	connectionsTotal.Inc()
	connectionDuration.Observe(duration)
}

// RecordError records an error
func RecordError(code string) {
	errorsTotal.WithLabelValues(code).Inc()
}

// SetActiveTunnels sets the number of active tunnels
func SetActiveTunnels(count int) {
	activeTunnels.Set(float64(count))
}

// RecordHeartbeat records a heartbeat
func RecordHeartbeat(latency float64) {
	heartbeatLatency.Observe(latency)
}

// RecordMissedHeartbeat records a missed heartbeat
func RecordMissedHeartbeat() {
	missedHeartbeats.Inc()
} 