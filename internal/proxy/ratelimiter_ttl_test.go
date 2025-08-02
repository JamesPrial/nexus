package proxy

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// Test TTL cleanup for per-client rate limiter
func TestPerClientRateLimiterWithTTL(t *testing.T) {
	logger := &mockLogger{}
	// Create limiter with 200ms TTL for fast testing
	limiter := NewPerClientRateLimiterWithTTL(1, 1, 200*time.Millisecond, logger)

	// Start cleanup routine
	stopChan := make(chan struct{})
	go limiter.StartCleanup(100*time.Millisecond, stopChan)
	defer close(stopChan)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := limiter.Middleware(next)

	// Make request from client1
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.Header.Set("Authorization", "Bearer client1")
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	// Verify client1 is tracked
	if !limiter.HasClient("Bearer client1") {
		t.Error("Client1 should be tracked after request")
	}

	// Wait for TTL to expire
	time.Sleep(300 * time.Millisecond)

	// Verify client1 was cleaned up
	if limiter.HasClient("Bearer client1") {
		t.Error("Client1 should have been cleaned up after TTL")
	}
}

// Test that active clients are not cleaned up
func TestPerClientRateLimiterWithTTL_ActiveClients(t *testing.T) {
	logger := &mockLogger{}
	// Create limiter with 300ms TTL
	limiter := NewPerClientRateLimiterWithTTL(10, 10, 300*time.Millisecond, logger)

	// Start cleanup routine
	stopChan := make(chan struct{})
	go limiter.StartCleanup(100*time.Millisecond, stopChan)
	defer close(stopChan)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := limiter.Middleware(next)

	// Make periodic requests to keep client active
	done := make(chan struct{})
	go func() {
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer active-client")
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			time.Sleep(100 * time.Millisecond)
		}
		close(done)
	}()

	// Wait for requests to complete
	<-done

	// Verify client is still tracked (was active recently)
	if !limiter.HasClient("Bearer active-client") {
		t.Error("Active client should not have been cleaned up")
	}

	// Wait for TTL after last request
	time.Sleep(400 * time.Millisecond)

	// Now it should be cleaned up
	if limiter.HasClient("Bearer active-client") {
		t.Error("Client should have been cleaned up after becoming inactive")
	}
}

// Test concurrent access during cleanup
func TestPerClientRateLimiterWithTTL_Concurrent(t *testing.T) {
	logger := &mockLogger{}
	limiter := NewPerClientRateLimiterWithTTL(100, 100, 100*time.Millisecond, logger)

	// Start cleanup routine
	stopChan := make(chan struct{})
	go limiter.StartCleanup(50*time.Millisecond, stopChan)
	defer close(stopChan)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := limiter.Middleware(next)

	// Launch concurrent requests from different clients
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Authorization", "Bearer client"+string(rune('0'+clientID)))
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)
				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	// No panic means concurrent access is safe
}

// Test memory cleanup effectiveness
func TestPerClientRateLimiterWithTTL_MemoryCleanup(t *testing.T) {
	logger := &mockLogger{}
	limiter := NewPerClientRateLimiterWithTTL(1000, 1000, 100*time.Millisecond, logger)

	// Start cleanup routine
	stopChan := make(chan struct{})
	go limiter.StartCleanup(50*time.Millisecond, stopChan)
	defer close(stopChan)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := limiter.Middleware(next)

	// Create many different clients
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer client%d", i))
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}

	// Check initial count
	initialCount := limiter.ClientCount()
	if initialCount != 100 {
		t.Errorf("Expected 100 clients, got %d", initialCount)
	}

	// Wait for cleanup
	time.Sleep(200 * time.Millisecond)

	// All clients should be cleaned up
	finalCount := limiter.ClientCount()
	if finalCount != 0 {
		t.Errorf("Expected 0 clients after cleanup, got %d", finalCount)
	}

	// Check that cleanup was logged
	foundCleanupLog := false
	totalCleaned := 0
	for _, log := range logger.logs {
		if log.level == "debug" && log.message == "Cleaned up inactive rate limiter entries" {
			foundCleanupLog = true
			if count, ok := log.fields["cleaned_count"].(int); ok {
				totalCleaned += count
			}
		}
	}
	if !foundCleanupLog {
		t.Error("Expected cleanup log entry")
	}
	// Allow for multiple cleanup batches
	if totalCleaned < 90 || totalCleaned > 100 {
		t.Errorf("Expected cleanup count around 100, got %d", totalCleaned)
	}
}

// Test stopping cleanup routine
func TestPerClientRateLimiterWithTTL_StopCleanup(t *testing.T) {
	logger := &mockLogger{}
	limiter := NewPerClientRateLimiterWithTTL(1, 1, 100*time.Millisecond, logger)

	// Start cleanup routine
	stopChan := make(chan struct{})
	cleanupDone := make(chan struct{})
	go func() {
		limiter.StartCleanup(50*time.Millisecond, stopChan)
		close(cleanupDone)
	}()

	// Let it run briefly
	time.Sleep(100 * time.Millisecond)

	// Stop cleanup
	close(stopChan)

	// Wait for cleanup to stop
	select {
	case <-cleanupDone:
		// Good, cleanup stopped
	case <-time.After(200 * time.Millisecond):
		t.Error("Cleanup routine did not stop in time")
	}
}

// Test token limiter with TTL
func TestTokenLimiterWithTTL(t *testing.T) {
	logger := &mockLogger{}
	tokenCounter := &DefaultTokenCounter{}
	// Create limiter with 200ms TTL
	limiter := NewTokenLimiterWithTTL(60, 10, tokenCounter, 200*time.Millisecond, logger)

	// Start cleanup routine
	stopChan := make(chan struct{})
	go limiter.StartCleanup(100*time.Millisecond, stopChan)
	defer close(stopChan)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := limiter.Middleware(next)

	// Make request from client1
	body := `{"messages": [{"role": "user", "content": "Hi"}]}`
	req1 := httptest.NewRequest("POST", "/test", strings.NewReader(body))
	req1.Header.Set("Authorization", "client1-key")
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	// Verify client1 is tracked
	if !limiter.HasClient("client1-key") {
		t.Error("Client1 should be tracked after request")
	}

	// Wait for TTL to expire
	time.Sleep(300 * time.Millisecond)

	// Verify client1 was cleaned up
	if limiter.HasClient("client1-key") {
		t.Error("Client1 should have been cleaned up after TTL")
	}
}

// Benchmark cleanup performance
func BenchmarkPerClientRateLimiterWithTTL_Cleanup(b *testing.B) {
	logger := &mockLogger{}
	limiter := NewPerClientRateLimiterWithTTL(1000, 1000, 100*time.Millisecond, logger)

	// Pre-populate with clients
	for i := 0; i < 1000; i++ {
		limiter.getOrCreateLimiter("client" + string(rune(i)))
	}

	// Set all last access times to past
	limiter.mu.Lock()
	now := time.Now()
	for key := range limiter.lastAccess {
		limiter.lastAccess[key] = now.Add(-200 * time.Millisecond)
	}
	limiter.mu.Unlock()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.cleanup()
	}
}