package telemetry

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// TelemetryMiddleware wraps HTTP handlers to automatically collect telemetry
type TelemetryMiddleware struct {
	telemetry *InventoryApiTelemetry
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

// NewTelemetryMiddleware creates a new telemetry middleware
func NewTelemetryMiddleware(telemetry *InventoryApiTelemetry) *TelemetryMiddleware {
	return &TelemetryMiddleware{
		telemetry: telemetry,
	}
}

// Middleware returns the HTTP middleware function
func (tm *TelemetryMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code
		wrapper := &responseWriterWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // Default to 200
		}

		// Extract telemetry data from request
		metrics := tm.extractMetricsFromRequest(r)

		// Call the next handler
		next.ServeHTTP(wrapper, r)

		// Complete the metrics with response data
		metrics.StatusCode = wrapper.statusCode
		metrics.Duration = time.Since(start)

		// Extract additional telemetry data from context
		ctx := r.Context()
		UpdateMetricsFromContext(ctx, &metrics)

		// Record telemetry based on success/failure
		if wrapper.statusCode >= 400 {
			// Error case
			metrics.ErrorMessage = tm.getErrorMessage(wrapper.statusCode)
			tm.telemetry.RegisterRequestError(ctx, metrics)
		} else {
			// Success case
			tm.telemetry.RegisterRequestReceived(ctx, metrics)
		}

		// Always record duration
		tm.telemetry.RegisterRequestDuration(ctx, metrics)
	})
}

// responseWriterWrapper wraps http.ResponseWriter to capture status code
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriterWrapper) Write(data []byte) (int, error) {
	return w.ResponseWriter.Write(data)
}

// extractMetricsFromRequest extracts telemetry data from the HTTP request
func (tm *TelemetryMiddleware) extractMetricsFromRequest(r *http.Request) InventoryApiMetrics {
	// Extract client IP and normalize it for low cardinality
	clientIP := getClientIP(r)
	clientIPType := NormalizeClientIP(clientIP)

	metrics := InventoryApiMetrics{
		Method:       r.Method,
		Endpoint:     GetEndpointFromPath(r.URL.Path),
		ClientIP:     clientIP,     // Raw IP for logging
		ClientIPType: clientIPType, // Normalized for metrics (low cardinality)
		// Removed other high-cardinality attributes:
		// - APIKey (creates series per client)
		// - ProductID extraction (can create thousands of series)
	}

	// No longer extract endpoint-specific high-cardinality data
	// All specific data will be set by handlers through context if needed
	// and only low-cardinality attributes will be used

	return metrics
}

// getClientIP function removed - no longer collecting client IP to prevent high cardinality

// getErrorMessage returns a human-readable error message for the status code
func (tm *TelemetryMiddleware) getErrorMessage(statusCode int) string {
	switch statusCode {
	case http.StatusBadRequest:
		return "Bad Request"
	case http.StatusUnauthorized:
		return "Unauthorized"
	case http.StatusForbidden:
		return "Forbidden"
	case http.StatusNotFound:
		return "Not Found"
	case http.StatusMethodNotAllowed:
		return "Method Not Allowed"
	case http.StatusConflict:
		return "Conflict"
	case http.StatusUnprocessableEntity:
		return "Unprocessable Entity"
	case http.StatusInternalServerError:
		return "Internal Server Error"
	case http.StatusBadGateway:
		return "Bad Gateway"
	case http.StatusServiceUnavailable:
		return "Service Unavailable"
	case http.StatusGatewayTimeout:
		return "Gateway Timeout"
	default:
		return "HTTP Error " + strconv.Itoa(statusCode)
	}
}

// SetProductCount sets the product count for listing endpoints
func SetProductCount(ctx context.Context, count int) context.Context {
	return context.WithValue(ctx, "telemetry_product_count", count)
}

// SetEventCount sets the event count for event endpoints
func SetEventCount(ctx context.Context, count int) context.Context {
	return context.WithValue(ctx, "telemetry_event_count", count)
}

// SetStoreID sets the store ID for update operations
func SetStoreID(ctx context.Context, storeID string) context.Context {
	return context.WithValue(ctx, "telemetry_store_id", storeID)
}

// SetProductID function removed - no longer collecting product ID to prevent high cardinality

// GetProductCount retrieves the product count from context
func GetProductCount(ctx context.Context) int {
	if count, ok := ctx.Value("telemetry_product_count").(int); ok {
		return count
	}
	return 0
}

// GetEventCount retrieves the event count from context
func GetEventCount(ctx context.Context) int {
	if count, ok := ctx.Value("telemetry_event_count").(int); ok {
		return count
	}
	return 0
}

// GetStoreID retrieves the store ID from context
func GetStoreID(ctx context.Context) string {
	if storeID, ok := ctx.Value("telemetry_store_id").(string); ok {
		return storeID
	}
	return ""
}

// GetProductIDFromContext function removed - no longer collecting product ID to prevent high cardinality

// UpdateMetricsFromContext updates metrics with data stored in context
func UpdateMetricsFromContext(ctx context.Context, metrics *InventoryApiMetrics) {
	// Only update low-cardinality attributes to prevent metric explosion
	if storeID := GetStoreID(ctx); storeID != "" {
		metrics.StoreID = storeID
	}
	// Keep business metrics for aggregation but don't use as attributes
	if productCount := GetProductCount(ctx); productCount > 0 {
		metrics.ProductCount = productCount
	}
	if eventCount := GetEventCount(ctx); eventCount > 0 {
		metrics.EventCount = eventCount
	}
	// Removed high-cardinality attributes:
	// - ProductID (can create thousands of unique metric series)
}
