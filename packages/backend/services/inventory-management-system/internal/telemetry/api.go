package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// InventoryApiTelemetry provides telemetry for Inventory Management API endpoints
type InventoryApiTelemetry struct {
	meter metric.Meter

	// Request counters
	requestCounter metric.Int64Counter

	// Error counters
	errorCounter metric.Int64Counter

	// Duration histograms
	durationHistogram metric.Float64Histogram

	// Additional metrics for inventory-specific operations
	inventoryUpdateCounter metric.Int64Counter
	eventRetrievalCounter  metric.Int64Counter
	productQueryCounter    metric.Int64Counter
}

// InventoryApiMetrics contains the telemetry data for a request
type InventoryApiMetrics struct {
	Method       string
	Endpoint     string
	StatusCode   int
	Duration     time.Duration
	ErrorMessage string
	// Removed high-cardinality attributes:
	// - ClientIP (can create many unique series)
	// - APIKey (creates series per client)
	// - ProductID (can create thousands of series)
	// Keep only low-cardinality business metrics
	StoreID      string // Keep if store count is manageable
	EventCount   int
	ProductCount int
}

// NewInventoryApiTelemetry creates a new instance of InventoryApiTelemetry
func NewInventoryApiTelemetry() *InventoryApiTelemetry {
	return &InventoryApiTelemetry{}
}

// InitializeTelemetry sets up all the telemetry instruments for the Inventory API
func (t *InventoryApiTelemetry) InitializeTelemetry(ctx context.Context) error {
	slog.Info("Initializing Inventory API telemetry")

	// Get the global meter provider
	t.meter = otel.Meter("inventory-management-api")

	var err error

	// Initialize request counter
	t.requestCounter, err = t.meter.Int64Counter(
		"inventory_api_requests_total",
		metric.WithDescription("Total number of API requests to inventory endpoints"),
		metric.WithUnit("1"),
	)
	if err != nil {
		slog.Error("Failed to create request counter", "error", err)
		return fmt.Errorf("failed to create request counter: %w", err)
	}

	// Initialize error counter
	t.errorCounter, err = t.meter.Int64Counter(
		"inventory_api_errors_total",
		metric.WithDescription("Total number of API errors from inventory endpoints"),
		metric.WithUnit("1"),
	)
	if err != nil {
		slog.Error("Failed to create error counter", "error", err)
		return fmt.Errorf("failed to create error counter: %w", err)
	}

	// Initialize duration histogram
	t.durationHistogram, err = t.meter.Float64Histogram(
		"inventory_api_request_duration_seconds",
		metric.WithDescription("Duration of API requests to inventory endpoints"),
		metric.WithUnit("s"),
	)
	if err != nil {
		slog.Error("Failed to create duration histogram", "error", err)
		return fmt.Errorf("failed to create duration histogram: %w", err)
	}

	// Initialize inventory-specific counters
	t.inventoryUpdateCounter, err = t.meter.Int64Counter(
		"inventory_updates_total",
		metric.WithDescription("Total number of inventory update operations"),
		metric.WithUnit("1"),
	)
	if err != nil {
		slog.Error("Failed to create inventory update counter", "error", err)
		return fmt.Errorf("failed to create inventory update counter: %w", err)
	}

	t.eventRetrievalCounter, err = t.meter.Int64Counter(
		"inventory_events_retrieved_total",
		metric.WithDescription("Total number of inventory events retrieved"),
		metric.WithUnit("1"),
	)
	if err != nil {
		slog.Error("Failed to create event retrieval counter", "error", err)
		return fmt.Errorf("failed to create event retrieval counter: %w", err)
	}

	t.productQueryCounter, err = t.meter.Int64Counter(
		"inventory_products_queried_total",
		metric.WithDescription("Total number of product queries"),
		metric.WithUnit("1"),
	)
	if err != nil {
		slog.Error("Failed to create product query counter", "error", err)
		return fmt.Errorf("failed to create product query counter: %w", err)
	}

	slog.Info("Inventory API telemetry initialized successfully")
	return nil
}

// RegisterRequestReceived records a successful API request
func (t *InventoryApiTelemetry) RegisterRequestReceived(ctx context.Context, metrics InventoryApiMetrics) {
	if t.requestCounter == nil {
		slog.Warn("Request counter not initialized")
		return
	}

	// Low-cardinality attributes only to prevent metric explosion
	attrs := []attribute.KeyValue{
		attribute.String("method", metrics.Method),
		attribute.String("endpoint", metrics.Endpoint),
		attribute.Int("status_code", metrics.StatusCode),
	}

	// Add store_id only if it has manageable cardinality
	if metrics.StoreID != "" {
		attrs = append(attrs, attribute.String("store_id", metrics.StoreID))
	}

	// Record the request
	t.requestCounter.Add(ctx, 1, metric.WithAttributes(attrs...))

	// Record endpoint-specific metrics
	t.recordEndpointSpecificMetrics(ctx, metrics)

	slog.Debug("Recorded successful API request",
		"method", metrics.Method,
		"endpoint", metrics.Endpoint,
		"status_code", metrics.StatusCode,
		"duration_ms", metrics.Duration.Milliseconds(),
	)
}

// RegisterRequestError records a failed API request
func (t *InventoryApiTelemetry) RegisterRequestError(ctx context.Context, metrics InventoryApiMetrics) {
	if t.errorCounter == nil {
		slog.Warn("Error counter not initialized")
		return
	}

	// Low-cardinality attributes only to prevent metric explosion
	attrs := []attribute.KeyValue{
		attribute.String("method", metrics.Method),
		attribute.String("endpoint", metrics.Endpoint),
		attribute.Int("status_code", metrics.StatusCode),
		attribute.String("error_type", categorizeError(metrics.ErrorMessage)),
	}

	// Add store_id only if it has manageable cardinality
	if metrics.StoreID != "" {
		attrs = append(attrs, attribute.String("store_id", metrics.StoreID))
	}

	t.errorCounter.Add(ctx, 1, metric.WithAttributes(attrs...))

	slog.Warn("Recorded API request error",
		"method", metrics.Method,
		"endpoint", metrics.Endpoint,
		"status_code", metrics.StatusCode,
		"error", metrics.ErrorMessage,
	)
}

// RegisterRequestDuration records the duration of an API request
func (t *InventoryApiTelemetry) RegisterRequestDuration(ctx context.Context, metrics InventoryApiMetrics) {
	if t.durationHistogram == nil {
		slog.Warn("Duration histogram not initialized")
		return
	}

	// Low-cardinality attributes only to prevent metric explosion
	attrs := []attribute.KeyValue{
		attribute.String("method", metrics.Method),
		attribute.String("endpoint", metrics.Endpoint),
		attribute.Int("status_code", metrics.StatusCode),
	}

	// Add store_id only if it has manageable cardinality
	if metrics.StoreID != "" {
		attrs = append(attrs, attribute.String("store_id", metrics.StoreID))
	}

	// Record duration in seconds
	durationSeconds := metrics.Duration.Seconds()
	t.durationHistogram.Record(ctx, durationSeconds, metric.WithAttributes(attrs...))

	slog.Debug("Recorded API request duration",
		"method", metrics.Method,
		"endpoint", metrics.Endpoint,
		"duration_seconds", durationSeconds,
	)
}

// recordEndpointSpecificMetrics records metrics specific to each endpoint type
func (t *InventoryApiTelemetry) recordEndpointSpecificMetrics(ctx context.Context, metrics InventoryApiMetrics) {
	switch metrics.Endpoint {
	case "/v1/inventory":
		// Product listing endpoint - use low-cardinality attributes
		if t.productQueryCounter != nil {
			attrs := []attribute.KeyValue{
				attribute.String("operation", "list_products"),
				// Remove product_count as it can vary widely
			}
			t.productQueryCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
		}

	case "/v1/inventory/{productId}":
		// Individual product endpoint - remove product_id to prevent high cardinality
		if t.productQueryCounter != nil {
			attrs := []attribute.KeyValue{
				attribute.String("operation", "get_product"),
			}
			t.productQueryCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
		}

	case "/v1/inventory/updates":
		// Inventory update endpoint - keep store_id if manageable cardinality
		if t.inventoryUpdateCounter != nil {
			attrs := []attribute.KeyValue{
				attribute.String("store_id", metrics.StoreID),
				// Remove product_id to prevent high cardinality
			}
			t.inventoryUpdateCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
		}

	case "/v1/inventory/events":
		// Events endpoint - remove event_count as it can vary widely
		if t.eventRetrievalCounter != nil {
			attrs := []attribute.KeyValue{
				// Remove event_count to prevent high cardinality
			}
			t.eventRetrievalCounter.Add(ctx, int64(metrics.EventCount), metric.WithAttributes(attrs...))
		}
	}
}

// categorizeError groups similar errors to prevent high cardinality
func categorizeError(errorMessage string) string {
	if errorMessage == "" {
		return "unknown"
	}

	// Group common error patterns to keep cardinality low
	switch {
	case strings.Contains(errorMessage, "not found"):
		return "not_found"
	case strings.Contains(errorMessage, "invalid"):
		return "invalid_request"
	case strings.Contains(errorMessage, "unauthorized"):
		return "unauthorized"
	case strings.Contains(errorMessage, "forbidden"):
		return "forbidden"
	case strings.Contains(errorMessage, "timeout"):
		return "timeout"
	case strings.Contains(errorMessage, "internal"):
		return "internal_error"
	case strings.Contains(errorMessage, "bad request"):
		return "bad_request"
	case strings.Contains(errorMessage, "conflict"):
		return "conflict"
	default:
		return "other"
	}
}

// GetEndpointFromPath normalizes the endpoint path for telemetry
func GetEndpointFromPath(path string) string {
	// Normalize paths with parameters to template format
	switch {
	case path == "/v1/inventory":
		return "/v1/inventory"
	case path == "/v1/inventory/updates":
		return "/v1/inventory/updates"
	case path == "/v1/inventory/events":
		return "/v1/inventory/events"
	default:
		// Handle parameterized paths like /v1/inventory/SKU-001
		if len(path) > len("/v1/inventory/") && path[:len("/v1/inventory/")] == "/v1/inventory/" {
			return "/v1/inventory/{productId}"
		}
		return path
	}
}
