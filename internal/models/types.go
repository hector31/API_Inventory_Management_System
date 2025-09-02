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
	StoreID        string `json:"storeId"`
	ProductID      string `json:"productId"`
	Delta          int    `json:"delta"`
	Version        int    `json:"version"`
	IdempotencyKey string `json:"idempotencyKey"`
}

type UpdateResponse struct {
	ProductID   string `json:"productId"`
	NewQuantity int    `json:"newQuantity"`
	NewVersion  int    `json:"newVersion"`
	Applied     bool   `json:"applied"`
	LastUpdated string `json:"lastUpdated"`
}

type ProductResponse struct {
	ProductID   string `json:"productId"`
	Available   int    `json:"available"`
	Version     int    `json:"version"`
	LastUpdated string `json:"lastUpdated"`
}

type GlobalAvailabilityResponse struct {
	ProductID      string         `json:"productId"`
	TotalAvailable int            `json:"totalAvailable"`
	PerStore       map[string]int `json:"perStore"`
}

type ListResponse struct {
	Items      []ProductResponse `json:"items"`
	NextCursor string            `json:"nextCursor"`
}

type SyncRequest struct {
	StoreID  string        `json:"storeId"`
	Mode     string        `json:"mode"`
	Products []SyncProduct `json:"products"`
}

type SyncProduct struct {
	ID      string `json:"id"`
	Qty     int    `json:"qty"`
	Version int    `json:"version"`
}

type SyncResponse struct {
	Updated int `json:"updated"`
	Created int `json:"created"`
	Skipped int `json:"skipped"`
}

type ReplicationSnapshot struct {
	State      map[string]ProductResponse `json:"state"`
	LastOffset int                        `json:"lastOffset"`
}

type ReplicationChanges struct {
	Events     []ReplicationEvent `json:"events"`
	NextOffset int                `json:"nextOffset"`
	HasMore    bool               `json:"hasMore"`
}

type ReplicationEvent struct {
	Seq        int    `json:"seq"`
	Type       string `json:"type"`
	ProductID  string `json:"productId"`
	StoreID    string `json:"storeId"`
	Delta      int    `json:"delta"`
	NewVersion int    `json:"newVersion"`
	Timestamp  string `json:"ts"`
}
