package models

import "time"

// Product represents an inventory item
type Product struct {
	ProductID   string    `json:"productId"`
	Name        string    `json:"name"`
	Available   int       `json:"available"`
	Version     int       `json:"version"`
	LastUpdated time.Time `json:"lastUpdated"`
	Price       float64   `json:"price"`
}

// UpdateRequest represents a single inventory update request
type UpdateRequest struct {
	StoreID        string `json:"storeId" validate:"required"`
	ProductID      string `json:"productId" validate:"required"`
	Delta          int    `json:"delta" validate:"required"`
	Version        int    `json:"version" validate:"required,min=1"`
	IdempotencyKey string `json:"idempotencyKey" validate:"required"`
}

// BatchUpdateRequest represents a batch of inventory updates
type BatchUpdateRequest struct {
	StoreID string          `json:"storeId" validate:"required"`
	Updates []UpdateRequest `json:"updates" validate:"required,min=1,max=100,dive"`
}

// UpdateResponse represents the response for an inventory update
type UpdateResponse struct {
	ProductID      string `json:"productId"`
	NewQuantity    int    `json:"newQuantity"`
	NewVersion     int    `json:"newVersion"`
	Delta          int    `json:"delta"`
	IdempotencyKey string `json:"idempotencyKey"`
	Applied        bool   `json:"applied"`
}

// BatchUpdateResponse represents the response for a batch update
type BatchUpdateResponse struct {
	StoreID      string           `json:"storeId"`
	Results      []UpdateResponse `json:"results"`
	TotalCount   int              `json:"totalCount"`
	SuccessCount int              `json:"successCount"`
	FailureCount int              `json:"failureCount"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Service   string    `json:"service,omitempty"`
	Version   string    `json:"version,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

// ReplicationResponse represents inventory data for replication
type ReplicationResponse struct {
	Products   []Product `json:"products"`
	LastOffset int       `json:"lastOffset"`
	Snapshot   bool      `json:"snapshot"`
	Timestamp  time.Time `json:"timestamp"`
}

// ServiceConfig represents common service configuration
type ServiceConfig struct {
	Port        int    `json:"port"`
	Environment string `json:"environment"`
	LogLevel    string `json:"logLevel"`
	ServiceName string `json:"serviceName"`
	Version     string `json:"version"`
}

// APIKeyConfig represents API key configuration
type APIKeyConfig struct {
	Keys []string `json:"keys"`
}

// Event represents a change event in the inventory system
type Event struct {
	Offset    int64           `json:"offset"`
	Timestamp string          `json:"timestamp"`
	EventType string          `json:"eventType"`
	ProductID string          `json:"productId"`
	Data      ProductResponse `json:"data"`
	Version   int             `json:"version"`
}

// ProductResponse represents product data in events
type ProductResponse struct {
	ProductID   string  `json:"productId"`
	Name        string  `json:"name"`
	Available   int     `json:"available"`
	Version     int     `json:"version"`
	LastUpdated string  `json:"lastUpdated"`
	Price       float64 `json:"price"`
}

// EventsResponse represents the response for the events endpoint
type EventsResponse struct {
	Events     []Event `json:"events"`
	NextOffset int64   `json:"nextOffset"`
	HasMore    bool    `json:"hasMore"`
	Count      int     `json:"count"`
}

// EventType constants
const (
	EventTypeProductUpdated = "product_updated"
	EventTypeProductCreated = "product_created"
)
