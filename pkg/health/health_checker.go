package health

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// HealthStatus represents health check status
type HealthStatus string

const (
	Healthy   HealthStatus = "healthy"
	Unhealthy HealthStatus = "unhealthy"
	Degraded  HealthStatus = "degraded"
	Unknown   HealthStatus = "unknown"
)

// HealthCheck represents a health check
type HealthCheck struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Status      HealthStatus           `json:"status"`
	LastCheck   time.Time              `json:"last_check"`
	LastError   error                  `json:"last_error,omitempty"`
	Duration    time.Duration          `json:"duration"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// HealthCheckerFunc is a function that performs a health check
type HealthCheckerFunc func(ctx context.Context) (*HealthCheck, error)

// HealthChecker manages health checks
type HealthChecker struct {
	checks       map[string]HealthCheckerFunc
	interval     time.Duration
	timeout      time.Duration
	lastResults  map[string]*HealthCheck
	stopChan     chan struct{}
	isRunning    bool
	mu           sync.RWMutex
}

// Config holds health checker configuration
type Config struct {
	Interval time.Duration `json:"interval"`
	Timeout  time.Duration `json:"timeout"`
}

// DefaultConfig returns default health checker configuration
func DefaultConfig() *Config {
	return &Config{
		Interval: 30 * time.Second,
		Timeout:  10 * time.Second,
	}
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(config *Config) *HealthChecker {
	if config == nil {
		config = DefaultConfig()
	}
	
	return &HealthChecker{
		checks:      make(map[string]HealthCheckerFunc),
		interval:    config.Interval,
		timeout:     config.Timeout,
		lastResults: make(map[string]*HealthCheck),
		stopChan:    make(chan struct{}),
	}
}

// AddCheck adds a health check
func (hc *HealthChecker) AddCheck(name string, checker HealthCheckerFunc) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	
	hc.checks[name] = checker
}

// RemoveCheck removes a health check
func (hc *HealthChecker) RemoveCheck(name string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	
	delete(hc.checks, name)
	delete(hc.lastResults, name)
}

// Start starts the health checker
func (hc *HealthChecker) Start() {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	
	if hc.isRunning {
		return
	}
	
	hc.isRunning = true
	hc.stopChan = make(chan struct{})
	
	go hc.run()
}

// Stop stops the health checker
func (hc *HealthChecker) Stop() {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	
	if !hc.isRunning {
		return
	}
	
	close(hc.stopChan)
	hc.isRunning = false
}

// run runs the health checker loop
func (hc *HealthChecker) run() {
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()
	
	// Run initial check
	hc.runChecks()
	
	for {
		select {
		case <-ticker.C:
			hc.runChecks()
		case <-hc.stopChan:
			return
		}
	}
}

// runChecks runs all health checks
func (hc *HealthChecker) runChecks() {
	hc.mu.RLock()
	checks := make(map[string]HealthCheckerFunc)
	for k, v := range hc.checks {
		checks[k] = v
	}
	hc.mu.RUnlock()
	
	var wg sync.WaitGroup
	results := make(chan struct {
		name   string
		result *HealthCheck
		err    error
	}, len(checks))
	
	for name, checker := range checks {
		wg.Add(1)
		go func(name string, checker HealthCheckerFunc) {
			defer wg.Done()
			
			ctx, cancel := context.WithTimeout(context.Background(), hc.timeout)
			defer cancel()
			
			start := time.Now()
			result, err := checker(ctx)
			duration := time.Since(start)
			
			if result == nil {
				result = &HealthCheck{
					Name:      name,
					Status:    Unknown,
					LastCheck: time.Now(),
					Duration:  duration,
				}
			}
			
			if err != nil {
				result.Status = Unhealthy
				result.LastError = err
			}
			
			result.LastCheck = time.Now()
			result.Duration = duration
			
			results <- struct {
				name   string
				result *HealthCheck
				err    error
			}{name, result, err}
		}(name, checker)
	}
	
	go func() {
		wg.Wait()
		close(results)
	}()
	
	hc.mu.Lock()
	for result := range results {
		hc.lastResults[result.name] = result.result
	}
	hc.mu.Unlock()
}

// GetStatus returns the overall health status
func (hc *HealthChecker) GetStatus() HealthStatus {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	
	if len(hc.lastResults) == 0 {
		return Unknown
	}
	
	unhealthyCount := 0
	degradedCount := 0
	
	for _, result := range hc.lastResults {
		switch result.Status {
		case Unhealthy:
			unhealthyCount++
		case Degraded:
			degradedCount++
		}
	}
	
	if unhealthyCount > 0 {
		return Unhealthy
	}
	
	if degradedCount > 0 {
		return Degraded
	}
	
	return Healthy
}

// GetResults returns all health check results
func (hc *HealthChecker) GetResults() map[string]*HealthCheck {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	
	results := make(map[string]*HealthCheck)
	for k, v := range hc.lastResults {
		results[k] = v
	}
	return results
}

// GetResult returns a specific health check result
func (hc *HealthChecker) GetResult(name string) (*HealthCheck, bool) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	
	result, exists := hc.lastResults[name]
	return result, exists
}

// IsHealthy returns true if all checks are healthy
func (hc *HealthChecker) IsHealthy() bool {
	return hc.GetStatus() == Healthy
}

// RunCheck runs a specific health check
func (hc *HealthChecker) RunCheck(name string) (*HealthCheck, error) {
	hc.mu.RLock()
	checker, exists := hc.checks[name]
	hc.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("health check %s not found", name)
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), hc.timeout)
	defer cancel()
	
	start := time.Now()
	result, err := checker(ctx)
	duration := time.Since(start)
	
	if result == nil {
		result = &HealthCheck{
			Name:      name,
			Status:    Unknown,
			LastCheck: time.Now(),
			Duration:  duration,
		}
	}
	
	if err != nil {
		result.Status = Unhealthy
		result.LastError = err
	}
	
	result.LastCheck = time.Now()
	result.Duration = duration
	
	hc.mu.Lock()
	hc.lastResults[name] = result
	hc.mu.Unlock()
	
	return result, err
}

// HTTPHealthCheck creates an HTTP health check
func HTTPHealthCheck(name, url string) HealthCheckerFunc {
	return func(ctx context.Context) (*HealthCheck, error) {
		client := &http.Client{
			Timeout: 5 * time.Second,
		}
		
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}
		
		resp, err := client.Do(req)
		if err != nil {
			return &HealthCheck{
				Name:        name,
				Description: fmt.Sprintf("HTTP health check for %s", url),
				Status:      Unhealthy,
				LastError:   err,
			}, err
		}
		defer resp.Body.Close()
		
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return &HealthCheck{
				Name:        name,
				Description: fmt.Sprintf("HTTP health check for %s", url),
				Status:      Healthy,
				Metadata: map[string]interface{}{
					"status_code": resp.StatusCode,
					"url":         url,
				},
			}, nil
		}
		
		return &HealthCheck{
			Name:        name,
			Description: fmt.Sprintf("HTTP health check for %s", url),
			Status:      Unhealthy,
			LastError:   fmt.Errorf("unexpected status code: %d", resp.StatusCode),
			Metadata: map[string]interface{}{
				"status_code": resp.StatusCode,
				"url":         url,
			},
		}, nil
	}
}

// PingHealthCheck creates a ping health check
func PingHealthCheck(name, host string) HealthCheckerFunc {
	return func(ctx context.Context) (*HealthCheck, error) {
		conn, err := net.DialTimeout("tcp", host, 5*time.Second)
		if err != nil {
			return &HealthCheck{
				Name:        name,
				Description: fmt.Sprintf("Ping health check for %s", host),
				Status:      Unhealthy,
				LastError:   err,
			}, err
		}
		defer conn.Close()
		
		return &HealthCheck{
			Name:        name,
			Description: fmt.Sprintf("Ping health check for %s", host),
			Status:      Healthy,
			Metadata: map[string]interface{}{
				"host": host,
			},
		}, nil
	}
}

// CustomHealthCheck creates a custom health check
func CustomHealthCheck(name, description string, fn func(ctx context.Context) error) HealthCheckerFunc {
	return func(ctx context.Context) (*HealthCheck, error) {
		err := fn(ctx)
		if err != nil {
			return &HealthCheck{
				Name:        name,
				Description: description,
				Status:      Unhealthy,
				LastError:   err,
			}, err
		}
		
		return &HealthCheck{
			Name:        name,
			Description: description,
			Status:      Healthy,
		}, nil
	}
}

// ConnectionHealthCheck creates a connection health check
func ConnectionHealthCheck(name, host string, port int) HealthCheckerFunc {
	return func(ctx context.Context) (*HealthCheck, error) {
		address := fmt.Sprintf("%s:%d", host, port)
		conn, err := net.DialTimeout("tcp", address, 5*time.Second)
		if err != nil {
			return &HealthCheck{
				Name:        name,
				Description: fmt.Sprintf("Connection health check for %s", address),
				Status:      Unhealthy,
				LastError:   err,
			}, err
		}
		defer conn.Close()
		
		return &HealthCheck{
			Name:        name,
			Description: fmt.Sprintf("Connection health check for %s", address),
			Status:      Healthy,
			Metadata: map[string]interface{}{
				"host": host,
				"port": port,
			},
		}, nil
	}
} 