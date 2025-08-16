package circuitbreaker

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/sony/gobreaker"
)

// State represents the circuit breaker state
type State int

const (
	Closed State = iota
	HalfOpen
	Open
)

func (s State) String() string {
	switch s {
	case Closed:
		return "closed"
	case HalfOpen:
		return "half_open"
	case Open:
		return "open"
	default:
		return "unknown"
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	breaker *gobreaker.CircuitBreaker
	mu      sync.RWMutex
	stats   *CircuitBreakerStats
}

// CircuitBreakerStats tracks circuit breaker statistics
type CircuitBreakerStats struct {
	TotalRequests   int64
	SuccessfulRequests int64
	FailedRequests  int64
	LastFailure     time.Time
	LastSuccess     time.Time
	State           State
}

// Config holds circuit breaker configuration
type Config struct {
	Name                   string
	MaxFailures            uint32
	Timeout                time.Duration
	Interval               time.Duration
	ReadyToTrip            func(counts gobreaker.Counts) bool
	OnStateChange          func(name string, from gobreaker.State, to gobreaker.State)
}

// DefaultConfig returns default circuit breaker configuration
func DefaultConfig() *Config {
	return &Config{
		Name:        "default",
		MaxFailures: 5,
		Timeout:     30 * time.Second,
		Interval:    60 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
	}
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config *Config) *CircuitBreaker {
	if config == nil {
		config = DefaultConfig()
	}

	cb := &CircuitBreaker{
		stats: &CircuitBreakerStats{
			State: Closed,
		},
	}

	// Create gobreaker circuit breaker
	cb.breaker = gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        config.Name,
		MaxRequests: 0, // Allow unlimited requests when half-open
		Interval:    config.Interval,
		Timeout:     config.Timeout,
		ReadyToTrip: config.ReadyToTrip,
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			cb.updateState(to)
			if config.OnStateChange != nil {
				config.OnStateChange(name, from, to)
			}
		},
	})

	return cb
}

// Execute runs a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	cb.mu.Lock()
	cb.stats.TotalRequests++
	cb.mu.Unlock()

	_, err := cb.breaker.Execute(func() (interface{}, error) {
		return nil, fn()
	})

	if err != nil {
		cb.mu.Lock()
		cb.stats.FailedRequests++
		cb.stats.LastFailure = time.Now()
		cb.mu.Unlock()
		return err
	}

	cb.mu.Lock()
	cb.stats.SuccessfulRequests++
	cb.stats.LastSuccess = time.Now()
	cb.mu.Unlock()

	return nil
}

// ExecuteWithResult runs a function that returns a result with circuit breaker protection
func ExecuteWithResult[T any](cb *CircuitBreaker, ctx context.Context, fn func() (T, error)) (T, error) {
	cb.mu.Lock()
	cb.stats.TotalRequests++
	cb.mu.Unlock()

	var zero T

	result, err := cb.breaker.Execute(func() (interface{}, error) {
		return fn()
	})

	if err != nil {
		cb.mu.Lock()
		cb.stats.FailedRequests++
		cb.stats.LastFailure = time.Now()
		cb.mu.Unlock()
		return zero, err
	}

	cb.mu.Lock()
	cb.stats.SuccessfulRequests++
	cb.stats.LastSuccess = time.Now()
	cb.mu.Unlock()

	if typedResult, ok := result.(T); ok {
		return typedResult, nil
	}

	return zero, errors.New("type assertion failed")
}

// Ready checks if the circuit breaker is ready to execute
func (cb *CircuitBreaker) Ready() bool {
	return cb.stats.State != Open
}

// State returns the current circuit breaker state
func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.stats.State
}

// GetStats returns circuit breaker statistics
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["name"] = cb.breaker.Name()
	stats["state"] = cb.stats.State.String()
	stats["total_requests"] = cb.stats.TotalRequests
	stats["successful_requests"] = cb.stats.SuccessfulRequests
	stats["failed_requests"] = cb.stats.FailedRequests
	stats["last_failure"] = cb.stats.LastFailure
	stats["last_success"] = cb.stats.LastSuccess
	stats["ready"] = cb.Ready()

	// Calculate success rate
	if cb.stats.TotalRequests > 0 {
		successRate := float64(cb.stats.SuccessfulRequests) / float64(cb.stats.TotalRequests)
		stats["success_rate"] = successRate
	} else {
		stats["success_rate"] = 0.0
	}

	return stats
}

// ForceOpen forces the circuit breaker to open state
func (cb *CircuitBreaker) ForceOpen() {
	cb.updateState(gobreaker.StateOpen)
}

// ForceClose forces the circuit breaker to closed state
func (cb *CircuitBreaker) ForceClose() {
	cb.updateState(gobreaker.StateClosed)
}

// Reset resets the circuit breaker to initial state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.stats = &CircuitBreakerStats{
		State: Closed,
	}
}

// updateState updates the internal state
func (cb *CircuitBreaker) updateState(state gobreaker.State) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch state {
	case gobreaker.StateClosed:
		cb.stats.State = Closed
	case gobreaker.StateHalfOpen:
		cb.stats.State = HalfOpen
	case gobreaker.StateOpen:
		cb.stats.State = Open
	}
}

// IsHealthy returns true if the circuit breaker is healthy
func (cb *CircuitBreaker) IsHealthy() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	// Consider healthy if success rate is above 80% or no requests yet
	if cb.stats.TotalRequests == 0 {
		return true
	}

	successRate := float64(cb.stats.SuccessfulRequests) / float64(cb.stats.TotalRequests)
	return successRate >= 0.8 && cb.stats.State != Open
}

// GetName returns the circuit breaker name
func (cb *CircuitBreaker) GetName() string {
	return cb.breaker.Name()
} 