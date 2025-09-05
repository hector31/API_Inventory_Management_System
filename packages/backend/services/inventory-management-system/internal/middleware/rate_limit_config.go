package middleware

import (
	"log/slog"
	"strconv"
	"strings"
	"time"

	"inventory-management-api/internal/config"
)

// ParseRateLimitConfig parses rate limiting configuration from the config struct
func ParseRateLimitConfig(cfg *config.Config) RateLimitConfig {
	rateLimitConfig := RateLimitConfig{
		Enabled:                parseBool(cfg.RateLimitEnabled, true),
		Type:                   parseRateLimitType(cfg.RateLimitType),
		RequestsPerMinute:      parseInt(cfg.RateLimitRequestsPerMinute, 100),
		WindowMinutes:          parseInt(cfg.RateLimitWindowMinutes, 1),
		AdminRequestsPerMinute: parseInt(cfg.RateLimitAdminRequestsPerMinute, 50),
	}

	// Validate configuration
	if rateLimitConfig.RequestsPerMinute <= 0 {
		slog.Warn("Invalid rate limit requests per minute, using default",
			"configured", cfg.RateLimitRequestsPerMinute, "default", 100)
		rateLimitConfig.RequestsPerMinute = 100
	}

	if rateLimitConfig.WindowMinutes <= 0 {
		slog.Warn("Invalid rate limit window minutes, using default",
			"configured", cfg.RateLimitWindowMinutes, "default", 1)
		rateLimitConfig.WindowMinutes = 1
	}

	if rateLimitConfig.AdminRequestsPerMinute <= 0 {
		slog.Warn("Invalid admin rate limit requests per minute, using default",
			"configured", cfg.RateLimitAdminRequestsPerMinute, "default", 50)
		rateLimitConfig.AdminRequestsPerMinute = 50
	}

	// Log the final configuration
	slog.Info("Rate limiting configuration parsed",
		"enabled", rateLimitConfig.Enabled,
		"type", rateLimitConfig.Type,
		"requests_per_minute", rateLimitConfig.RequestsPerMinute,
		"window_minutes", rateLimitConfig.WindowMinutes,
		"admin_requests_per_minute", rateLimitConfig.AdminRequestsPerMinute)

	return rateLimitConfig
}

// parseBool parses a string to bool with a default value
func parseBool(value string, defaultValue bool) bool {
	if value == "" {
		return defaultValue
	}

	switch strings.ToLower(value) {
	case "true", "1", "yes", "on", "enabled":
		return true
	case "false", "0", "no", "off", "disabled":
		return false
	default:
		slog.Warn("Invalid boolean value, using default",
			"value", value, "default", defaultValue)
		return defaultValue
	}
}

// parseInt parses a string to int with a default value
func parseInt(value string, defaultValue int) int {
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		slog.Warn("Invalid integer value, using default",
			"value", value, "default", defaultValue, "error", err)
		return defaultValue
	}

	return parsed
}

// parseRateLimitType parses the rate limit type with validation
func parseRateLimitType(value string) RateLimitType {
	if value == "" {
		return RateLimitTypeIP // Default
	}

	switch strings.ToLower(value) {
	case "ip":
		return RateLimitTypeIP
	case "global":
		return RateLimitTypeGlobal
	case "both":
		return RateLimitTypeBoth
	default:
		slog.Warn("Invalid rate limit type, using default",
			"value", value, "default", "ip")
		return RateLimitTypeIP
	}
}

// GetRateLimitStats returns current rate limiting statistics
func (rl *RateLimiter) GetRateLimitStats() map[string]interface{} {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	stats := map[string]interface{}{
		"enabled":                   rl.config.Enabled,
		"type":                      string(rl.config.Type),
		"requests_per_minute":       rl.config.RequestsPerMinute,
		"window_minutes":            rl.config.WindowMinutes,
		"admin_requests_per_minute": rl.config.AdminRequestsPerMinute,
		"active_ip_limits":          len(rl.ipLimits),
	}

	// Add global limit stats if applicable
	if rl.config.Type == RateLimitTypeGlobal || rl.config.Type == RateLimitTypeBoth {
		rl.globalLimit.mutex.RLock()
		stats["global_count"] = rl.globalLimit.Count
		stats["global_reset_time"] = rl.globalLimit.ResetTime.Format("2006-01-02T15:04:05Z07:00")
		rl.globalLimit.mutex.RUnlock()
	}

	return stats
}

// ResetRateLimits resets all rate limiting counters (useful for testing)
func (rl *RateLimiter) ResetRateLimits() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	// Clear all IP limits
	rl.ipLimits = make(map[string]*RateLimitEntry)

	// Reset global limit
	rl.globalLimit.mutex.Lock()
	rl.globalLimit.Count = 0
	rl.globalLimit.ResetTime = time.Time{}
	rl.globalLimit.mutex.Unlock()

	slog.Info("Rate limits reset")
}
