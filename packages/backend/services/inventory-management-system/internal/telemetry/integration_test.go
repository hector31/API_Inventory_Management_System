package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// TestTelemetryIntegration tests the complete telemetry integration
func TestTelemetryIntegration(t *testing.T) {
	// Initialize telemetry
	apiTelemetry := NewInventoryApiTelemetry()
	ctx := context.Background()

	if err := apiTelemetry.InitializeTelemetry(ctx); err != nil {
		t.Fatalf("Failed to initialize telemetry: %v", err)
	}

	// Create middleware
	middleware := NewTelemetryMiddleware(apiTelemetry)

	// Create a test handler that sets telemetry context data
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Set telemetry context data based on endpoint
		switch r.URL.Path {
		case "/v1/inventory":
			ctx = SetProductCount(ctx, 5)
		case "/v1/inventory/SKU-001":
			// No longer setting product ID to prevent high cardinality
		case "/v1/inventory/updates":
			ctx = SetStoreID(ctx, "store-1")
			// No longer setting product ID to prevent high cardinality
		case "/v1/inventory/events":
			ctx = SetEventCount(ctx, 3)
		}

		// Simulate some processing time
		time.Sleep(10 * time.Millisecond)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	})

	// Create router with middleware
	router := mux.NewRouter()
	router.Use(middleware.Middleware)
	router.HandleFunc("/v1/inventory", testHandler).Methods("GET")
	router.HandleFunc("/v1/inventory/{productId}", testHandler).Methods("GET")
	router.HandleFunc("/v1/inventory/updates", testHandler).Methods("POST")
	router.HandleFunc("/v1/inventory/events", testHandler).Methods("GET")

	// Test cases
	testCases := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "List Products",
			method:         "GET",
			path:           "/v1/inventory",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Get Product",
			method:         "GET",
			path:           "/v1/inventory/SKU-001",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Update Inventory",
			method:         "POST",
			path:           "/v1/inventory/updates",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Get Events",
			method:         "GET",
			path:           "/v1/inventory/events",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create request
			var req *http.Request
			if tc.method == "POST" {
				req = httptest.NewRequest(tc.method, tc.path, strings.NewReader(`{"test": "data"}`))
			} else {
				req = httptest.NewRequest(tc.method, tc.path, nil)
			}

			// Add headers
			req.Header.Set("X-API-Key", "test-api-key-12345")
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			rr := httptest.NewRecorder()

			// Execute request
			router.ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, rr.Code)
			}

			// Verify response
			if rr.Body.String() == "" {
				t.Error("Expected response body, got empty")
			}
		})
	}
}

// TestTelemetryContextHelpers tests the context helper functions
func TestTelemetryContextHelpers(t *testing.T) {
	ctx := context.Background()

	// Test SetProductCount and GetProductCount
	ctx = SetProductCount(ctx, 10)
	if count := GetProductCount(ctx); count != 10 {
		t.Errorf("Expected product count 10, got %d", count)
	}

	// Test SetEventCount and GetEventCount
	ctx = SetEventCount(ctx, 5)
	if count := GetEventCount(ctx); count != 5 {
		t.Errorf("Expected event count 5, got %d", count)
	}

	// Test SetStoreID and GetStoreID
	ctx = SetStoreID(ctx, "store-123")
	if storeID := GetStoreID(ctx); storeID != "store-123" {
		t.Errorf("Expected store ID 'store-123', got '%s'", storeID)
	}

	// Product ID functions removed to prevent high cardinality
}

// TestUpdateMetricsFromContext tests the UpdateMetricsFromContext function
func TestUpdateMetricsFromContext(t *testing.T) {
	ctx := context.Background()

	// Set context data
	ctx = SetProductCount(ctx, 15)
	ctx = SetEventCount(ctx, 8)
	ctx = SetStoreID(ctx, "store-456")
	// Product ID no longer set to prevent high cardinality

	// Create metrics
	metrics := InventoryApiMetrics{
		Method:   "GET",
		Endpoint: "/v1/inventory",
	}

	// Update metrics from context
	UpdateMetricsFromContext(ctx, &metrics)

	// Verify metrics were updated
	if metrics.ProductCount != 15 {
		t.Errorf("Expected product count 15, got %d", metrics.ProductCount)
	}
	if metrics.EventCount != 8 {
		t.Errorf("Expected event count 8, got %d", metrics.EventCount)
	}
	if metrics.StoreID != "store-456" {
		t.Errorf("Expected store ID 'store-456', got '%s'", metrics.StoreID)
	}
	// Product ID test removed to prevent high cardinality
}

// TestGetEndpointFromPath tests the endpoint normalization function
func TestGetEndpointFromPath(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"/v1/inventory", "/v1/inventory"},
		{"/v1/inventory/updates", "/v1/inventory/updates"},
		{"/v1/inventory/events", "/v1/inventory/events"},
		{"/v1/inventory/SKU-001", "/v1/inventory/{productId}"},
		{"/v1/inventory/PROD-123", "/v1/inventory/{productId}"},
		{"/v1/inventory/some-long-product-id", "/v1/inventory/{productId}"},
		{"/unknown/path", "/unknown/path"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := GetEndpointFromPath(tc.input)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// TestMaskAPIKey removed - API key masking no longer used to prevent high cardinality
