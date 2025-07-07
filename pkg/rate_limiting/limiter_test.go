package rate_limiting

import (
	"testing"
	"time"
)

func TestNewLimiter(t *testing.T) {
	config := &Config{
		MaxRetries:       5,
		BackoffMultiplier: 2.0,
		MaxBackoff:       10 * time.Second,
		WindowSize:       30 * time.Second,
		MaxRequests:      50,
		CleanupInterval:  1 * time.Minute,
	}

	limiter := NewLimiter(config)
	if limiter == nil {
		t.Fatal("Expected limiter to be created")
	}

	if limiter.maxRetries != 5 {
		t.Errorf("Expected maxRetries to be 5, got %d", limiter.maxRetries)
	}

	limiter.Close()
}

func TestAllowWithinLimit(t *testing.T) {
	config := &Config{
		MaxRequests: 10,
		WindowSize:  1 * time.Minute,
		CleanupInterval: 1 * time.Minute,
		BackoffMultiplier: 2.0,
		MaxRetries: 3,
		MaxBackoff: 10 * time.Second,
	}
	limiter := NewLimiter(config)
	defer limiter.Close()

	userID := "test-user"

	// Make requests within limit
	for i := 0; i < 10; i++ {
		allowed, _, err := limiter.Allow(userID)
		if !allowed {
			t.Errorf("Expected request %d to be allowed", i+1)
		}
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	}
}

func TestAllowExceedsLimit(t *testing.T) {
	config := &Config{
		MaxRequests: 5,
		WindowSize:  1 * time.Minute,
		CleanupInterval: 1 * time.Minute,
		BackoffMultiplier: 2.0,
		MaxRetries: 3,
		MaxBackoff: 10 * time.Second,
	}
	limiter := NewLimiter(config)
	defer limiter.Close()

	userID := "test-user"

	// Make requests within limit
	for i := 0; i < 5; i++ {
		allowed, _, err := limiter.Allow(userID)
		if !allowed {
			t.Errorf("Expected request %d to be allowed", i+1)
		}
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	}

	// This request should be denied
	allowed, backoff, _ := limiter.Allow(userID)
	if allowed {
		t.Error("Expected request to be denied")
	}
	if backoff <= 0 {
		t.Error("Expected positive backoff duration")
	}
}

func TestBackoffCalculation(t *testing.T) {
	config := &Config{
		MaxRetries:       3,
		BackoffMultiplier: 2.0,
		MaxBackoff:       10 * time.Second,
		MaxRequests:      1,
		CleanupInterval:  1 * time.Minute,
	}
	limiter := NewLimiter(config)
	defer limiter.Close()

	userID := "test-user"
	// First request should be allowed
	allowed, _, _ := limiter.Allow(userID)
	if !allowed {
		t.Error("Expected first request to be allowed")
	}

	// Second request should trigger backoff
	allowed, backoff, _ := limiter.Allow(userID)
	if allowed {
		t.Error("Expected second request to be denied")
	}

	// Backoff should be exponential
	expectedBackoff := time.Duration(float64(time.Second) * 2.0)
	if backoff != expectedBackoff {
		t.Errorf("Expected backoff %v, got %v", expectedBackoff, backoff)
	}
}

func TestWindowReset(t *testing.T) {
	config := &Config{
		MaxRequests: 5,
		WindowSize:  100 * time.Millisecond, // Short window for testing
		CleanupInterval: 1 * time.Minute,
		BackoffMultiplier: 2.0,
		MaxRetries: 3,
		MaxBackoff: 10 * time.Second,
	}
	limiter := NewLimiter(config)
	defer limiter.Close()

	userID := "test-user"

	// Make requests up to limit
	for i := 0; i < 5; i++ {
		allowed, _, _ := limiter.Allow(userID)
		if !allowed {
			t.Errorf("Expected request %d to be allowed", i+1)
		}
	}

	// Wait for window to reset
	time.Sleep(150 * time.Millisecond)

	// Should be able to make requests again
	allowed, _, _ := limiter.Allow(userID)
	if !allowed {
		t.Error("Expected request to be allowed after window reset")
	}
}

func TestResetUser(t *testing.T) {
	config := &Config{
		MaxRequests: 5,
		WindowSize:  1 * time.Minute,
		CleanupInterval: 1 * time.Minute,
		BackoffMultiplier: 2.0,
		MaxRetries: 3,
		MaxBackoff: 10 * time.Second,
	}
	limiter := NewLimiter(config)
	defer limiter.Close()

	userID := "test-user"

	// Make requests up to limit
	for i := 0; i < 5; i++ {
		limiter.Allow(userID)
	}

	// Reset user
	limiter.ResetUser(userID)

	// Should be able to make requests again
	allowed, _, err := limiter.Allow(userID)
	if !allowed {
		t.Error("Expected request to be allowed after user reset")
	}
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestGetStats(t *testing.T) {
	config := &Config{
		MaxRequests: 5,
		WindowSize:  1 * time.Minute,
		CleanupInterval: 1 * time.Minute,
		BackoffMultiplier: 2.0,
		MaxRetries: 3,
		MaxBackoff: 10 * time.Second,
	}
	limiter := NewLimiter(config)
	defer limiter.Close()

	// Make some requests
	limiter.Allow("user1")
	limiter.Allow("user2")
	
	// Exceed limit for user1 (make 5 more requests to reach limit of 5)
	for i := 0; i < 5; i++ {
		limiter.Allow("user1")
	}

	stats := limiter.GetStats()
	if stats["total_users"] != 2 {
		t.Errorf("Expected 2 users, got %v", stats["total_users"])
	}
	if stats["users_in_backoff"] != 1 {
		t.Errorf("Expected 1 user in backoff, got %v", stats["users_in_backoff"])
	}
}

func TestConcurrentAccess(t *testing.T) {
	config := &Config{
		MaxRequests: 100,
		WindowSize:  1 * time.Minute,
		CleanupInterval: 1 * time.Minute,
		BackoffMultiplier: 2.0,
		MaxRetries: 3,
		MaxBackoff: 10 * time.Second,
	}
	limiter := NewLimiter(config)
	defer limiter.Close()

	userID := "test-user"
	done := make(chan bool, 10)

	// Make concurrent requests
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				limiter.Allow(userID)
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify stats
	stats := limiter.GetStats()
	if stats["total_users"] != 1 {
		t.Errorf("Expected 1 user, got %v", stats["total_users"])
	}
} 