package models

// ErrorResponse represents the standard error response format
type ErrorResponse struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Details []ErrorDetail `json:"details,omitempty"`
}

type ErrorDetail struct {
	Field string `json:"field"`
	Issue string `json:"issue"`
}

// Request/Response types for inventory operations
type UpdateRequest struct {
	// Single product update fields
	StoreID        string `json:"storeId,omitempty"`
	ProductID      string `json:"productId,omitempty"`
	Delta          int    `json:"delta,omitempty"`
	Version        int    `json:"version,omitempty"`
	IdempotencyKey string `json:"idempotencyKey,omitempty"`

	// Batch update fields
	Updates []ProductUpdate `json:"updates,omitempty"`
}

// ProductUpdate represents a single product update in a batch operation
type ProductUpdate struct {
	ProductID      string `json:"productId"`
	Delta          int    `json:"delta"`
	Version        int    `json:"version"`
	IdempotencyKey string `json:"idempotencyKey"`
}

type UpdateResponse struct {
	// Single product response fields
	ProductID   string `json:"productId,omitempty"`
	NewQuantity int    `json:"newQuantity,omitempty"`
	NewVersion  int    `json:"newVersion,omitempty"`
	Applied     bool   `json:"applied,omitempty"`
	LastUpdated string `json:"lastUpdated,omitempty"`

	// Batch response fields
	Results      []ProductUpdateResult `json:"results,omitempty"`
	Summary      *BatchSummary         `json:"summary,omitempty"`
	ErrorType    string                `json:"errorType,omitempty"`
	ErrorMessage string                `json:"errorMessage,omitempty"`
}

// ProductUpdateResult represents the result of a single product update in a batch
type ProductUpdateResult struct {
	ProductID    string `json:"productId"`
	NewQuantity  int    `json:"newQuantity"`
	NewVersion   int    `json:"newVersion"`
	Applied      bool   `json:"applied"`
	LastUpdated  string `json:"lastUpdated"`
	ErrorType    string `json:"errorType,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// BatchSummary provides summary statistics for batch operations
type BatchSummary struct {
	Total     int `json:"total"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
}

type ProductResponse struct {
	ProductID   string  `json:"productId"`
	Name        string  `json:"name"`
	Available   int     `json:"available"`
	Version     int     `json:"version"`
	LastUpdated string  `json:"lastUpdated"`
	Price       float64 `json:"price"`
}

type ListResponse struct {
	Items      []ProductResponse `json:"items"`
	NextCursor string            `json:"nextCursor"`
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

// Admin SET endpoint models
type AdminSetRequest struct {
	Products []AdminProductUpdate `json:"products"`
}

type AdminProductUpdate struct {
	ProductID string   `json:"productId"`
	Name      *string  `json:"name,omitempty"`      // Pointer for optional field
	Available *int     `json:"available,omitempty"` // Pointer for optional field
	Price     *float64 `json:"price,omitempty"`     // Pointer for optional field
}

type AdminSetResponse struct {
	Results []AdminProductResult `json:"results"`
	Summary AdminSetSummary      `json:"summary"`
}

type AdminProductResult struct {
	ProductID    string `json:"productId"`
	Success      bool   `json:"success"`
	NewVersion   int    `json:"newVersion,omitempty"`
	LastUpdated  string `json:"lastUpdated,omitempty"`
	ErrorType    string `json:"errorType,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

type AdminSetSummary struct {
	TotalRequests     int `json:"totalRequests"`
	SuccessfulUpdates int `json:"successfulUpdates"`
	FailedUpdates     int `json:"failedUpdates"`
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
