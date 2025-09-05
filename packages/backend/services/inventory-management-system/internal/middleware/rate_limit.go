package middleware

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"inventory-management-api/internal/models"
)

// RateLimitType defines the type of rate limiting
type RateLimitType string

const (
	RateLimitTypeIP     RateLimitType = "ip"
	RateLimitTypeGlobal RateLimitType = "global"
	RateLimitTypeBoth   RateLimitType = "both"
)

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled                bool
	Type                   RateLimitType
	RequestsPerMinute      int
	WindowMinutes          int
	AdminRequestsPerMinute int
}

// RateLimitEntry represents a rate limit entry
type RateLimitEntry struct {
	Count     int
	ResetTime time.Time
	mutex     sync.RWMutex
}

// RateLimiter manages rate limiting
type RateLimiter struct {
	config        RateLimitConfig
	ipLimits      map[string]*RateLimitEntry
	globalLimit   *RateLimitEntry
	mutex         sync.RWMutex
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		config:      config,
		ipLimits:    make(map[string]*RateLimitEntry),
		globalLimit: &RateLimitEntry{},
		stopCleanup: make(chan struct{}),
	}

	// Start cleanup goroutine to remove expired entries
	rl.cleanupTicker = time.NewTicker(time.Minute)
	go rl.cleanupExpiredEntries()

	slog.Info("Rate limiter initialized",
		"enabled", config.Enabled,
		"type", config.Type,
		"requests_per_minute", config.RequestsPerMinute,
		"window_minutes", config.WindowMinutes,
		"admin_requests_per_minute", config.AdminRequestsPerMinute)

	return rl
}

// Stop stops the rate limiter and cleanup goroutine
func (rl *RateLimiter) Stop() {
	if rl.cleanupTicker != nil {
		rl.cleanupTicker.Stop()
	}
	close(rl.stopCleanup)
}

// cleanupExpiredEntries removes expired rate limit entries
func (rl *RateLimiter) cleanupExpiredEntries() {
	for {
		select {
		case <-rl.cleanupTicker.C:
			rl.mutex.Lock()
			now := time.Now()
			for ip, entry := range rl.ipLimits {
				entry.mutex.RLock()
				expired := now.After(entry.ResetTime)
				entry.mutex.RUnlock()

				if expired {
					delete(rl.ipLimits, ip)
				}
			}

			// Reset global limit if expired
			rl.globalLimit.mutex.RLock()
			globalExpired := now.After(rl.globalLimit.ResetTime)
			rl.globalLimit.mutex.RUnlock()

			if globalExpired {
				rl.globalLimit.mutex.Lock()
				rl.globalLimit.Count = 0
				rl.globalLimit.ResetTime = time.Time{}
				rl.globalLimit.mutex.Unlock()
			}

			rl.mutex.Unlock()
		case <-rl.stopCleanup:
			return
		}
	}
}

// IsAllowed checks if a request is allowed based on rate limiting rules
func (rl *RateLimiter) IsAllowed(clientIP string, isAdmin bool) (bool, *RateLimitInfo) {
	if !rl.config.Enabled {
		return true, &RateLimitInfo{
			Limit:     -1, // Unlimited
			Remaining: -1,
			ResetTime: time.Time{},
		}
	}

	now := time.Now()
	windowDuration := time.Duration(rl.config.WindowMinutes) * time.Minute

	// Determine the limit based on whether it's an admin request
	limit := rl.config.RequestsPerMinute
	if isAdmin && rl.config.AdminRequestsPerMinute > 0 {
		limit = rl.config.AdminRequestsPerMinute
	}

	var ipAllowed, globalAllowed bool = true, true
	var ipInfo, globalInfo *RateLimitInfo

	// Check IP-based rate limiting
	if rl.config.Type == RateLimitTypeIP || rl.config.Type == RateLimitTypeBoth {
		ipAllowed, ipInfo = rl.checkIPLimit(clientIP, limit, windowDuration, now)
	}

	// Check global rate limiting
	if rl.config.Type == RateLimitTypeGlobal || rl.config.Type == RateLimitTypeBoth {
		globalAllowed, globalInfo = rl.checkGlobalLimit(limit, windowDuration, now)
	}

	// For "both" type, use the most restrictive limit
	if rl.config.Type == RateLimitTypeBoth {
		allowed := ipAllowed && globalAllowed

		// Return the most restrictive info
		info := ipInfo
		if globalInfo != nil && (ipInfo == nil || globalInfo.Remaining < ipInfo.Remaining) {
			info = globalInfo
		}

		return allowed, info
	}

	// Return the appropriate result based on type
	if rl.config.Type == RateLimitTypeIP {
		return ipAllowed, ipInfo
	}

	return globalAllowed, globalInfo
}

// checkIPLimit checks IP-based rate limiting
func (rl *RateLimiter) checkIPLimit(clientIP string, limit int, windowDuration time.Duration, now time.Time) (bool, *RateLimitInfo) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	entry, exists := rl.ipLimits[clientIP]
	if !exists {
		entry = &RateLimitEntry{}
		rl.ipLimits[clientIP] = entry
	}

	entry.mutex.Lock()
	defer entry.mutex.Unlock()

	// Reset if window has expired
	if now.After(entry.ResetTime) {
		entry.Count = 0
		entry.ResetTime = now.Add(windowDuration)
	}

	info := &RateLimitInfo{
		Limit:     limit,
		Remaining: limit - entry.Count - 1, // -1 for current request
		ResetTime: entry.ResetTime,
	}

	if entry.Count >= limit {
		return false, info
	}

	entry.Count++
	info.Remaining = limit - entry.Count
	return true, info
}

// checkGlobalLimit checks global rate limiting
func (rl *RateLimiter) checkGlobalLimit(limit int, windowDuration time.Duration, now time.Time) (bool, *RateLimitInfo) {
	rl.globalLimit.mutex.Lock()
	defer rl.globalLimit.mutex.Unlock()

	// Reset if window has expired
	if now.After(rl.globalLimit.ResetTime) {
		rl.globalLimit.Count = 0
		rl.globalLimit.ResetTime = now.Add(windowDuration)
	}

	info := &RateLimitInfo{
		Limit:     limit,
		Remaining: limit - rl.globalLimit.Count - 1, // -1 for current request
		ResetTime: rl.globalLimit.ResetTime,
	}

	if rl.globalLimit.Count >= limit {
		return false, info
	}

	rl.globalLimit.Count++
	info.Remaining = limit - rl.globalLimit.Count
	return true, info
}

// RateLimitInfo contains rate limit information for response headers
type RateLimitInfo struct {
	Limit     int
	Remaining int
	ResetTime time.Time
}

// RateLimitMiddleware creates a rate limiting middleware using an existing rate limiter
func RateLimitMiddleware(rateLimiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip rate limiting for health check
			if r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}

			clientIP := getClientIP(r)
			isAdmin := strings.HasPrefix(r.URL.Path, "/v1/admin")

			allowed, info := rateLimiter.IsAllowed(clientIP, isAdmin)

			// Set rate limit headers
			setRateLimitHeaders(w, info)

			if !allowed {
				slog.Warn("Rate limit exceeded",
					"client_ip", clientIP,
					"path", r.URL.Path,
					"method", r.Method,
					"is_admin", isAdmin,
					"limit", info.Limit,
					"remaining", info.Remaining,
					"reset_time", info.ResetTime.Format(time.RFC3339))

				writeRateLimitErrorResponse(w, info)
				return
			}

			slog.Debug("Rate limit check passed",
				"client_ip", clientIP,
				"path", r.URL.Path,
				"remaining", info.Remaining)

			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for load balancers/proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		if net.ParseIP(xri) != nil {
			return xri
		}
	}

	// Fall back to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}

// setRateLimitHeaders sets rate limit headers in the response
func setRateLimitHeaders(w http.ResponseWriter, info *RateLimitInfo) {
	if info.Limit >= 0 {
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(info.Limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(info.Remaining))

		if !info.ResetTime.IsZero() {
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(info.ResetTime.Unix(), 10))
		}
	}
}

// writeRateLimitErrorResponse writes a rate limit exceeded error response
func writeRateLimitErrorResponse(w http.ResponseWriter, info *RateLimitInfo) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)

	retryAfter := ""
	if !info.ResetTime.IsZero() {
		retryAfter = fmt.Sprintf("%.0f", time.Until(info.ResetTime).Seconds())
		w.Header().Set("Retry-After", retryAfter)
	}

	errorResp := models.ErrorResponse{
		Code:    "rate_limit_exceeded",
		Message: "Rate limit exceeded. Please try again later.",
		Details: []models.ErrorDetail{
			{
				Field: "rate_limit",
				Issue: fmt.Sprintf("Exceeded %d requests per minute. %d requests remaining.",
					info.Limit, info.Remaining),
			},
			{
				Field: "retry_after",
				Issue: fmt.Sprintf("Retry after %s seconds", retryAfter),
			},
		},
	}

	json.NewEncoder(w).Encode(errorResp)
}
