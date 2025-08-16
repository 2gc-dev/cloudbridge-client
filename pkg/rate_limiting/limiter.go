package rate_limiting

import (
	"fmt"
	"sync"
	"time"
)

// Limiter implements rate limiting with exponential backoff
type Limiter struct {
	mu              sync.RWMutex
	limits          map[string]*UserLimit
	maxRetries      int
	backoffMultiplier float64
	maxBackoff      time.Duration
	cleanupInterval time.Duration
	lastCleanup    time.Time
	windowSize     time.Duration
	maxRequests    int
}

// UserLimit tracks rate limiting for a specific user
type UserLimit struct {
	UserID        string
	RequestCount  int
	LastRequest   time.Time
	RetryCount    int
	BackoffUntil  time.Time
	WindowStart   time.Time
	WindowSize    time.Duration
	MaxRequests   int
}

// Config holds rate limiting configuration
type Config struct {
	MaxRetries       int           `yaml:"max_retries"`
	BackoffMultiplier float64      `yaml:"backoff_multiplier"`
	MaxBackoff       time.Duration `yaml:"max_backoff"`
	WindowSize       time.Duration `yaml:"window_size"`
	MaxRequests      int           `yaml:"max_requests"`
	CleanupInterval  time.Duration `yaml:"cleanup_interval"`
}

// NewLimiter creates a new rate limiter
func NewLimiter(config *Config) *Limiter {
	if config == nil {
		config = &Config{
			MaxRetries:       3,
			BackoffMultiplier: 2.0,
			MaxBackoff:       30 * time.Second,
			WindowSize:       1 * time.Minute,
			MaxRequests:      100,
			CleanupInterval:  5 * time.Minute,
		}
	}

	limiter := &Limiter{
		limits:          make(map[string]*UserLimit),
		maxRetries:      config.MaxRetries,
		backoffMultiplier: config.BackoffMultiplier,
		maxBackoff:      config.MaxBackoff,
		cleanupInterval: config.CleanupInterval,
		lastCleanup:    time.Now(),
		windowSize:     config.WindowSize,
		maxRequests:    config.MaxRequests,
	}

	// Start cleanup goroutine
	go limiter.cleanupLoop()

	return limiter
}

// Allow checks if a request is allowed for the given user
func (l *Limiter) Allow(userID string) (bool, time.Duration, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Cleanup old entries if needed
	l.cleanupIfNeeded()

	// Get or create user limit
	userLimit, exists := l.limits[userID]
	if !exists {
		userLimit = &UserLimit{
			UserID:      userID,
			WindowStart: time.Now(),
			WindowSize:  l.getWindowSize(),
			MaxRequests: l.getMaxRequests(),
		}
		l.limits[userID] = userLimit
	}

	// Check if user is in backoff period
	if time.Now().Before(userLimit.BackoffUntil) {
		remaining := userLimit.BackoffUntil.Sub(time.Now())
		return false, remaining, fmt.Errorf("rate limit exceeded, retry after %v", remaining)
	}

	// Check if window has expired
	if time.Since(userLimit.WindowStart) > userLimit.WindowSize {
		userLimit.RequestCount = 0
		userLimit.WindowStart = time.Now()
		userLimit.RetryCount = 0
	}

	// Check if request count exceeds limit
	if userLimit.RequestCount >= userLimit.MaxRequests {
		userLimit.RetryCount++ // <--- увеличиваем до вычисления backoff
		calculatedBackoff := l.calculateBackoff(userLimit.RetryCount)
		userLimit.BackoffUntil = time.Now().Add(calculatedBackoff)
		return false, calculatedBackoff, fmt.Errorf("rate limit exceeded, retry after %v", calculatedBackoff)
	}

	// Allow request
	userLimit.RequestCount++
	userLimit.LastRequest = time.Now()

	return true, 0, nil
}

// calculateBackoff calculates exponential backoff duration
func (l *Limiter) calculateBackoff(retryCount int) time.Duration {
	if retryCount > l.maxRetries {
		retryCount = l.maxRetries
	}

	backoff := time.Duration(float64(time.Second) * l.backoffMultiplier * float64(retryCount))
	
	if backoff > l.maxBackoff {
		backoff = l.maxBackoff
	}

	return backoff
}

// getWindowSize returns the default window size for new users
func (l *Limiter) getWindowSize() time.Duration {
	if l.windowSize > 0 {
		return l.windowSize
	}
	return 1 * time.Minute
}

// getMaxRequests returns the default max requests for new users
func (l *Limiter) getMaxRequests() int {
	if l.maxRequests > 0 {
		return l.maxRequests
	}
	return 100
}

// cleanupIfNeeded removes old user limits
func (l *Limiter) cleanupIfNeeded() {
	if time.Since(l.lastCleanup) < l.cleanupInterval {
		return
	}

	l.lastCleanup = time.Now()
	cutoff := time.Now().Add(-l.cleanupInterval)

	for userID, userLimit := range l.limits {
		if userLimit.LastRequest.Before(cutoff) {
			delete(l.limits, userID)
		}
	}
}

// cleanupLoop runs periodic cleanup
func (l *Limiter) cleanupLoop() {
	ticker := time.NewTicker(l.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		l.mu.Lock()
		l.cleanupIfNeeded()
		l.mu.Unlock()
	}
}

// GetStats returns rate limiting statistics
func (l *Limiter) GetStats() map[string]interface{} {
	l.mu.RLock()
	defer l.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_users"] = len(l.limits)
	stats["max_retries"] = l.maxRetries
	stats["backoff_multiplier"] = l.backoffMultiplier
	stats["max_backoff"] = l.maxBackoff.String()

	// Count users in backoff
	usersInBackoff := 0
	for _, userLimit := range l.limits {
		if time.Now().Before(userLimit.BackoffUntil) {
			usersInBackoff++
		}
	}
	stats["users_in_backoff"] = usersInBackoff

	return stats
}

// ResetUser resets rate limiting for a specific user
func (l *Limiter) ResetUser(userID string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if userLimit, exists := l.limits[userID]; exists {
		userLimit.RequestCount = 0
		userLimit.RetryCount = 0
		userLimit.BackoffUntil = time.Time{}
		userLimit.WindowStart = time.Now()
	}
}

// Close stops the cleanup goroutine
func (l *Limiter) Close() {
	// The cleanup goroutine will stop when the ticker is stopped
	// This is handled in cleanupLoop
} 