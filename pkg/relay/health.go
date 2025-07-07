package relay

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// HealthStatus представляет текущее состояние сервера
type HealthStatus struct {
	Status            string    `json:"status"`
	Version          string    `json:"version"`
	Uptime           string    `json:"uptime"`
	ConnectionsTotal float64   `json:"connections_total"`
	ActiveTunnels    float64   `json:"active_tunnels"`
	ErrorsTotal      float64   `json:"errors_total"`
	MissedHeartbeats float64   `json:"missed_heartbeats"`
	LastUpdate       time.Time `json:"last_update"`
}

var (
	healthStatus = HealthStatus{
		Status:  "unknown",
		Version: "1.0.11",
	}
	startTime = time.Now()
)

// UpdateHealthStatus обновляет статус здоровья
func UpdateHealthStatus(status string) {
	healthStatus.Status = status
	healthStatus.LastUpdate = time.Now()
	healthStatus.Uptime = time.Since(startTime).String()
}

// GetHealthStatus возвращает текущий статус здоровья
func GetHealthStatus() HealthStatus {
	// Обновляем метрики
	healthStatus.ConnectionsTotal = getMetricValue(connectionsTotal)
	healthStatus.ActiveTunnels = getMetricValue(activeTunnels)
	healthStatus.ErrorsTotal = getMetricValue(errorsTotal)
	healthStatus.MissedHeartbeats = getMetricValue(missedHeartbeats)
	healthStatus.LastUpdate = time.Now()
	healthStatus.Uptime = time.Since(startTime).String()
	return healthStatus
}

func getMetricValue(metric prometheus.Collector) float64 {
	// Handle different metric types
	switch m := metric.(type) {
	case *prometheus.CounterVec:
		// For CounterVec, we need to get all metrics and sum them
		ch := make(chan prometheus.Metric, 100)
		go func() {
			m.Collect(ch)
			close(ch)
		}()
		var sum float64
		for metric := range ch {
			var dtoMetric dto.Metric
			if err := metric.Write(&dtoMetric); err == nil && dtoMetric.Counter != nil {
				sum += dtoMetric.Counter.GetValue()
			}
		}
		return sum
	default:
		// Try the original approach for other types
		var dtoMetric dto.Metric
		if err := metric.(prometheus.Metric).Write(&dtoMetric); err != nil {
			return 0
		}
		if dtoMetric.Counter != nil {
			return dtoMetric.Counter.GetValue()
		}
		if dtoMetric.Gauge != nil {
			return dtoMetric.Gauge.GetValue()
		}
		return 0
	}
}

// HealthCheckHandler обрабатывает запросы к /health
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	status := GetHealthStatus()
	
	w.Header().Set("Content-Type", "application/json")
	if status.Status != "ok" {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	
	if err := json.NewEncoder(w).Encode(status); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
} 