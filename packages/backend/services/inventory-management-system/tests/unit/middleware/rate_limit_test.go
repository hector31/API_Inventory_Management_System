package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"inventory-management-api/internal/middleware"
)

func TestRateLimiter_IPBasedLimiting(t *testing.T) {
	config := middleware.RateLimitConfig{
		Enabled:                true,
		Type:                   middleware.RateLimitTypeIP,
		RequestsPerMinute:      3,
		WindowMinutes:          1,
		AdminRequestsPerMinute: 2,
	}

	rateLimiter := middleware.NewRateLimiter(config)
	defer rateLimiter.Stop()

	clientIP := "192.168.1.1"

	// First 3 requests should be allowed
	for i := 0; i < 3; i++ {
		allowed, info := rateLimiter.IsAllowed(clientIP, false)
		if !allowed {
			t.Errorf("Request %d should be allowed", i+1)
		}
		if info.Remaining != 3-i-1 {
			t.Errorf("Expected remaining %d, got %d", 3-i-1, info.Remaining)
		}
	}

	// 4th request should be denied
	allowed, info := rateLimiter.IsAllowed(clientIP, false)
	if allowed {
		t.Error("4th request should be denied")
	}
	if info.Remaining != 0 {
		t.Errorf("Expected remaining 0, got %d", info.Remaining)
	}

	// Different IP should still be allowed
	allowed, _ = rateLimiter.IsAllowed("192.168.1.2", false)
	if !allowed {
		t.Error("Different IP should be allowed")
	}
}

func TestRateLimiter_GlobalLimiting(t *testing.T) {
	config := middleware.RateLimitConfig{
		Enabled:                true,
		Type:                   middleware.RateLimitTypeGlobal,
		RequestsPerMinute:      3,
		WindowMinutes:          1,
		AdminRequestsPerMinute: 2,
	}

	rateLimiter := middleware.NewRateLimiter(config)
	defer rateLimiter.Stop()

	// First 3 requests from different IPs should be allowed
	for i := 0; i < 3; i++ {
		clientIP := "192.168.1." + string(rune(i+1))
		allowed, info := rateLimiter.IsAllowed(clientIP, false)
		if !allowed {
			t.Errorf("Request %d should be allowed", i+1)
		}
		if info.Remaining != 3-i-1 {
			t.Errorf("Expected remaining %d, got %d", 3-i-1, info.Remaining)
		}
	}

	// 4th request from any IP should be denied
	allowed, info := rateLimiter.IsAllowed("192.168.1.4", false)
	if allowed {
		t.Error("4th request should be denied")
	}
	if info.Remaining != 0 {
		t.Errorf("Expected remaining 0, got %d", info.Remaining)
	}
}

func TestRateLimiter_BothLimiting(t *testing.T) {
	config := middleware.RateLimitConfig{
		Enabled:                true,
		Type:                   middleware.RateLimitTypeBoth,
		RequestsPerMinute:      5,
		WindowMinutes:          1,
		AdminRequestsPerMinute: 3,
	}

	rateLimiter := middleware.NewRateLimiter(config)
	defer rateLimiter.Stop()

	clientIP := "192.168.1.1"

	// First 5 requests from same IP should be allowed (IP limit)
	for i := 0; i < 5; i++ {
		allowed, _ := rateLimiter.IsAllowed(clientIP, false)
		if !allowed {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 6th request from same IP should be denied (IP limit exceeded)
	allowed, _ := rateLimiter.IsAllowed(clientIP, false)
	if allowed {
		t.Error("6th request from same IP should be denied")
	}
}

func TestRateLimiter_AdminLimiting(t *testing.T) {
	config := middleware.RateLimitConfig{
		Enabled:                true,
		Type:                   middleware.RateLimitTypeIP,
		RequestsPerMinute:      10,
		WindowMinutes:          1,
		AdminRequestsPerMinute: 2,
	}

	rateLimiter := middleware.NewRateLimiter(config)
	defer rateLimiter.Stop()

	clientIP := "192.168.1.1"

	// First 2 admin requests should be allowed
	for i := 0; i < 2; i++ {
		allowed, info := rateLimiter.IsAllowed(clientIP, true)
		if !allowed {
			t.Errorf("Admin request %d should be allowed", i+1)
		}
		if info.Limit != 2 {
			t.Errorf("Expected admin limit 2, got %d", info.Limit)
		}
	}

	// 3rd admin request should be denied
	allowed, _ := rateLimiter.IsAllowed(clientIP, true)
	if allowed {
		t.Error("3rd admin request should be denied")
	}

	// Regular requests should still use the higher limit
	allowed, info := rateLimiter.IsAllowed("192.168.1.2", false)
	if !allowed {
		t.Error("Regular request should be allowed")
	}
	if info.Limit != 10 {
		t.Errorf("Expected regular limit 10, got %d", info.Limit)
	}
}

func TestRateLimiter_Disabled(t *testing.T) {
	config := middleware.RateLimitConfig{
		Enabled:                false,
		Type:                   middleware.RateLimitTypeIP,
		RequestsPerMinute:      1,
		WindowMinutes:          1,
		AdminRequestsPerMinute: 1,
	}

	rateLimiter := middleware.NewRateLimiter(config)
	defer rateLimiter.Stop()

	// All requests should be allowed when disabled
	for i := 0; i < 10; i++ {
		allowed, info := rateLimiter.IsAllowed("192.168.1.1", false)
		if !allowed {
			t.Errorf("Request %d should be allowed when rate limiting is disabled", i+1)
		}
		if info.Limit != -1 {
			t.Errorf("Expected unlimited (-1), got %d", info.Limit)
		}
	}
}

func TestRateLimitMiddleware_Integration(t *testing.T) {
	config := middleware.RateLimitConfig{
		Enabled:                true,
		Type:                   middleware.RateLimitTypeIP,
		RequestsPerMinute:      2,
		WindowMinutes:          1,
		AdminRequestsPerMinute: 1,
	}

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Create rate limiter and wrap with middleware
	rateLimiter := middleware.NewRateLimiter(config)
	defer rateLimiter.Stop()
	rateLimitedHandler := middleware.RateLimitMiddleware(rateLimiter)(testHandler)

	// Test regular endpoint
	req1 := httptest.NewRequest("GET", "/v1/inventory", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	rr1 := httptest.NewRecorder()
	rateLimitedHandler.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Errorf("First request should succeed, got status %d", rr1.Code)
	}

	// Check rate limit headers
	if rr1.Header().Get("X-RateLimit-Limit") != "2" {
		t.Errorf("Expected X-RateLimit-Limit: 2, got %s", rr1.Header().Get("X-RateLimit-Limit"))
	}
	if rr1.Header().Get("X-RateLimit-Remaining") != "1" {
		t.Errorf("Expected X-RateLimit-Remaining: 1, got %s", rr1.Header().Get("X-RateLimit-Remaining"))
	}

	// Second request should succeed
	req2 := httptest.NewRequest("GET", "/v1/inventory", nil)
	req2.RemoteAddr = "192.168.1.1:12345"
	rr2 := httptest.NewRecorder()
	rateLimitedHandler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Errorf("Second request should succeed, got status %d", rr2.Code)
	}

	// Third request should be rate limited
	req3 := httptest.NewRequest("GET", "/v1/inventory", nil)
	req3.RemoteAddr = "192.168.1.1:12345"
	rr3 := httptest.NewRecorder()
	rateLimitedHandler.ServeHTTP(rr3, req3)

	if rr3.Code != http.StatusTooManyRequests {
		t.Errorf("Third request should be rate limited, got status %d", rr3.Code)
	}

	// Check that Retry-After header is set
	if rr3.Header().Get("Retry-After") == "" {
		t.Error("Retry-After header should be set")
	}
}

func TestRateLimitMiddleware_HealthCheckExemption(t *testing.T) {
	config := middleware.RateLimitConfig{
		Enabled:                true,
		Type:                   middleware.RateLimitTypeIP,
		RequestsPerMinute:      1,
		WindowMinutes:          1,
		AdminRequestsPerMinute: 1,
	}

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Create rate limiter and wrap with middleware
	rateLimiter := middleware.NewRateLimiter(config)
	defer rateLimiter.Stop()
	rateLimitedHandler := middleware.RateLimitMiddleware(rateLimiter)(testHandler)

	// Health check requests should not be rate limited
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()
		rateLimitedHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Health check request %d should succeed, got status %d", i+1, rr.Code)
		}

		// Rate limit headers should not be set for health checks
		if rr.Header().Get("X-RateLimit-Limit") != "" {
			t.Error("Rate limit headers should not be set for health checks")
		}
	}
}

func TestRateLimitMiddleware_AdminEndpoints(t *testing.T) {
	config := middleware.RateLimitConfig{
		Enabled:                true,
		Type:                   middleware.RateLimitTypeIP,
		RequestsPerMinute:      10,
		WindowMinutes:          1,
		AdminRequestsPerMinute: 1,
	}

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Create rate limiter and wrap with middleware
	rateLimiter := middleware.NewRateLimiter(config)
	defer rateLimiter.Stop()
	rateLimitedHandler := middleware.RateLimitMiddleware(rateLimiter)(testHandler)

	// First admin request should succeed
	req1 := httptest.NewRequest("POST", "/v1/admin/products/create", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	rr1 := httptest.NewRecorder()
	rateLimitedHandler.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Errorf("First admin request should succeed, got status %d", rr1.Code)
	}

	// Check that admin limit is applied
	if rr1.Header().Get("X-RateLimit-Limit") != "1" {
		t.Errorf("Expected admin limit 1, got %s", rr1.Header().Get("X-RateLimit-Limit"))
	}

	// Second admin request should be rate limited
	req2 := httptest.NewRequest("POST", "/v1/admin/products/create", nil)
	req2.RemoteAddr = "192.168.1.1:12345"
	rr2 := httptest.NewRecorder()
	rateLimitedHandler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusTooManyRequests {
		t.Errorf("Second admin request should be rate limited, got status %d", rr2.Code)
	}
}
