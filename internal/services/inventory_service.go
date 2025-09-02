package services

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"inventory-management-api/internal/models"
)

// InventoryService handles inventory business logic
type InventoryService struct {
	data *InventoryData
}

// InventoryData represents the complete inventory data structure
type InventoryData struct {
	Products map[string]ProductData `json:"products"`
	Metadata MetadataData           `json:"metadata"`
}

// ProductData represents complete product data
type ProductData struct {
	ProductID   string `json:"productId"`
	Name        string `json:"name"`
	Available   int    `json:"available"`
	Version     int    `json:"version"`
	LastUpdated string `json:"lastUpdated"`
}

// MetadataData represents system metadata for replication and caching
type MetadataData struct {
	LastOffset    int    `json:"lastOffset"`    // Last event sequence number for replication
	TotalProducts int    `json:"totalProducts"` // Quick count of total products
	LastUpdated   string `json:"lastUpdated"`   // System-wide last update timestamp
}

// NewInventoryService creates a new inventory service instance
func NewInventoryService() (*InventoryService, error) {
	service := &InventoryService{}
	err := service.loadTestData()
	if err != nil {
		return nil, fmt.Errorf("error loading test data: %w", err)
	}
	return service, nil
}

// loadTestData loads test data from JSON file
func (s *InventoryService) loadTestData() error {
	// Get the test data file path
	dataPath := filepath.Join("data", "inventory_test_data.json")

	slog.Debug("Loading test data", "path", dataPath)

	// Read the file
	data, err := os.ReadFile(dataPath)
	if err != nil {
		slog.Error("Failed to read test data file", "path", dataPath, "error", err)
		return fmt.Errorf("error reading test data file: %w", err)
	}

	// Parse the JSON
	s.data = &InventoryData{}
	err = json.Unmarshal(data, s.data)
	if err != nil {
		slog.Error("Failed to parse test data JSON", "path", dataPath, "error", err)
		return fmt.Errorf("error parsing test data JSON: %w", err)
	}

	slog.Info("Test data loaded successfully",
		"path", dataPath,
		"products_count", len(s.data.Products),
		"last_offset", s.data.Metadata.LastOffset)

	return nil
}

// GetProduct retrieves a product by its ID
func (s *InventoryService) GetProduct(productID string) (*models.ProductResponse, error) {
	slog.Debug("Retrieving product", "product_id", productID)

	// Search for the product in the data
	productData, exists := s.data.Products[productID]
	if !exists {
		slog.Warn("Product not found", "product_id", productID)
		return nil, fmt.Errorf("product not found: %s", productID)
	}

	// Convert to response structure
	response := &models.ProductResponse{
		ProductID:   productData.ProductID,
		Available:   productData.Available,
		Version:     productData.Version,
		LastUpdated: productData.LastUpdated,
	}

	slog.Debug("Product retrieved successfully",
		"product_id", productID,
		"available", response.Available,
		"version", response.Version)

	return response, nil
}

// ListProducts retrieves a list of products with pagination
func (s *InventoryService) ListProducts(cursor string, limit int) (*models.ListResponse, error) {
	// For simplicity, we return all products
	// In a real implementation, we would implement proper pagination

	var items []models.ProductResponse

	for _, productData := range s.data.Products {
		item := models.ProductResponse{
			ProductID:   productData.ProductID,
			Available:   productData.Available,
			Version:     productData.Version,
			LastUpdated: productData.LastUpdated,
		}
		items = append(items, item)

		// Limit the number of items if specified
		if limit > 0 && len(items) >= limit {
			break
		}
	}

	response := &models.ListResponse{
		Items:      items,
		NextCursor: "", // For now, no more pages
	}

	return response, nil
}

// ProductExists checks if a product exists
func (s *InventoryService) ProductExists(productID string) bool {
	_, exists := s.data.Products[productID]
	return exists
}

// GetProductCount returns the total number of products
func (s *InventoryService) GetProductCount() int {
	return len(s.data.Products)
}

// GetSystemMetadata returns system metadata for monitoring and replication
func (s *InventoryService) GetSystemMetadata() MetadataData {
	return s.data.Metadata
}

// GetLastOffset returns the last event offset for replication
func (s *InventoryService) GetLastOffset() int {
	return s.data.Metadata.LastOffset
}
