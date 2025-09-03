package services

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"inventory-management-api/internal/cache"
	"inventory-management-api/internal/config"
	"inventory-management-api/internal/models"
)

// InventoryService handles inventory business logic
type InventoryService struct {
	data                  *InventoryData
	globalMutex           sync.RWMutex // Only for global operations like file saves
	productLockManager    *ProductLockManager
	updateQueue           chan *UpdateRequest
	idempotencyCache      *cache.TTLCache
	dataFilePath          string
	enableJSONPersistence bool
	workerCount           int
	queueBufferSize       int
	stopWorkers           chan bool
	workersWaitGroup      sync.WaitGroup
}

// UpdateRequest represents an internal update request for queue processing
type UpdateRequest struct {
	ProductID      string
	Delta          int
	Version        int
	IdempotencyKey string
	StoreID        string
	ResponseChan   chan *UpdateResult
}

// UpdateResult represents the result of an update operation
type UpdateResult struct {
	Success     bool
	NewQuantity int
	NewVersion  int
	Error       error
	Applied     bool
	LastUpdated string
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
func NewInventoryService(cfg *config.Config) (*InventoryService, error) {
	// Parse cache TTL
	cacheTTL, err := time.ParseDuration(cfg.IdempotencyCacheTTL)
	if err != nil {
		slog.Warn("Invalid cache TTL, using default", "provided", cfg.IdempotencyCacheTTL, "error", err)
		cacheTTL = 2 * time.Minute
	}

	// Parse cleanup interval
	cleanupInterval, err := time.ParseDuration(cfg.IdempotencyCacheCleanupInterval)
	if err != nil {
		slog.Warn("Invalid cleanup interval, using default", "provided", cfg.IdempotencyCacheCleanupInterval, "error", err)
		cleanupInterval = 30 * time.Second
	}

	// Parse JSON persistence setting
	enablePersistence, err := strconv.ParseBool(cfg.EnableJSONPersistence)
	if err != nil {
		slog.Warn("Invalid JSON persistence setting, using default", "provided", cfg.EnableJSONPersistence, "error", err)
		enablePersistence = true
	}

	// Parse worker count
	workerCount, err := strconv.Atoi(cfg.InventoryWorkerCount)
	if err != nil || workerCount < 1 {
		slog.Warn("Invalid worker count, using default", "provided", cfg.InventoryWorkerCount, "error", err)
		workerCount = 1
	}

	// Parse queue buffer size
	queueBufferSize, err := strconv.Atoi(cfg.InventoryQueueBufferSize)
	if err != nil || queueBufferSize < 1 {
		slog.Warn("Invalid queue buffer size, using default", "provided", cfg.InventoryQueueBufferSize, "error", err)
		queueBufferSize = 100
	}

	service := &InventoryService{
		updateQueue:           make(chan *UpdateRequest, queueBufferSize),
		idempotencyCache:      cache.NewTTLCache(cacheTTL, cleanupInterval),
		productLockManager:    NewProductLockManager(),
		dataFilePath:          cfg.DataPath,
		enableJSONPersistence: enablePersistence,
		workerCount:           workerCount,
		queueBufferSize:       queueBufferSize,
		stopWorkers:           make(chan bool),
	}

	err = service.loadTestData()
	if err != nil {
		return nil, fmt.Errorf("error loading test data: %w", err)
	}

	// Start the worker pool
	service.startWorkerPool()

	slog.Info("Inventory service initialized with queue processing",
		"worker_count", workerCount,
		"queue_buffer_size", queueBufferSize,
		"cache_ttl", cacheTTL.String(),
		"cleanup_interval", cleanupInterval.String(),
		"json_persistence", enablePersistence)

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

// GetProduct retrieves a product by its ID using product-level read lock
func (s *InventoryService) GetProduct(productID string) (*models.ProductResponse, error) {
	slog.Debug("Retrieving product", "product_id", productID)

	var response *models.ProductResponse
	var err error

	// Use product-level read lock for concurrent access
	s.productLockManager.WithProductReadLock(productID, func() {
		// Search for the product in the data
		productData, exists := s.data.Products[productID]
		if !exists {
			slog.Warn("Product not found", "product_id", productID)
			err = fmt.Errorf("product not found: %s", productID)
			return
		}

		// Convert to response structure
		response = &models.ProductResponse{
			ProductID:   productData.ProductID,
			Available:   productData.Available,
			Version:     productData.Version,
			LastUpdated: productData.LastUpdated,
		}

		slog.Debug("Product retrieved successfully",
			"product_id", productID,
			"available", response.Available,
			"version", response.Version)
	})

	return response, err
}

// ListProducts retrieves a list of products with pagination
func (s *InventoryService) ListProducts(cursor string, limit int) (*models.ListResponse, error) {
	// Use global read lock for multi-product operations
	s.globalMutex.RLock()
	defer s.globalMutex.RUnlock()

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

// startWorkerPool starts the configured number of worker goroutines
func (s *InventoryService) startWorkerPool() {
	slog.Info("Starting inventory update worker pool", "worker_count", s.workerCount)

	for i := 0; i < s.workerCount; i++ {
		s.workersWaitGroup.Add(1)
		go s.processUpdateWorker(i + 1)
	}
}

// processUpdateWorker processes inventory updates from the queue
func (s *InventoryService) processUpdateWorker(workerID int) {
	defer s.workersWaitGroup.Done()

	slog.Debug("Starting inventory update worker", "worker_id", workerID)

	for {
		select {
		case updateReq := <-s.updateQueue:
			result := s.processUpdateInternal(updateReq)

			// Send result back through response channel
			select {
			case updateReq.ResponseChan <- result:
				// Successfully sent response
				slog.Debug("Update processed by worker",
					"worker_id", workerID,
					"product_id", updateReq.ProductID,
					"applied", result.Applied)
			case <-time.After(5 * time.Second):
				slog.Error("Timeout sending update result",
					"worker_id", workerID,
					"product_id", updateReq.ProductID,
					"idempotency_key", updateReq.IdempotencyKey)
			}
		case <-s.stopWorkers:
			slog.Debug("Stopping inventory update worker", "worker_id", workerID)
			return
		}
	}
}

// processUpdateInternal handles the actual update logic with OCC and idempotency
func (s *InventoryService) processUpdateInternal(req *UpdateRequest) *UpdateResult {
	slog.Debug("Processing update request",
		"product_id", req.ProductID,
		"delta", req.Delta,
		"version", req.Version,
		"idempotency_key", req.IdempotencyKey)

	// Check idempotency first using TTL cache (no locking needed for cache check)
	if cachedResult, exists := s.idempotencyCache.Get(req.IdempotencyKey); exists {
		if result, ok := cachedResult.(*UpdateResult); ok {
			slog.Info("Idempotent request detected, returning cached result",
				"idempotency_key", req.IdempotencyKey,
				"product_id", req.ProductID)
			return result
		}
	}

	var result *UpdateResult

	// Use product-level write lock for OCC-compliant update
	s.productLockManager.WithProductWriteLock(req.ProductID, func() {
		// Get current product data
		productData, exists := s.data.Products[req.ProductID]
		if !exists {
			result = &UpdateResult{
				Success: false,
				Error:   fmt.Errorf("product not found: %s", req.ProductID),
				Applied: false,
			}
			s.cacheIdempotencyResult(req.IdempotencyKey, result)
			return
		}

		// Check version for OCC
		if productData.Version != req.Version {
			result = &UpdateResult{
				Success: false,
				Error:   fmt.Errorf("version conflict: expected %d, got %d", productData.Version, req.Version),
				Applied: false,
			}
			s.cacheIdempotencyResult(req.IdempotencyKey, result)

			slog.Warn("Version conflict detected",
				"product_id", req.ProductID,
				"expected_version", productData.Version,
				"provided_version", req.Version,
				"idempotency_key", req.IdempotencyKey)

			return
		}

		// Calculate new quantity
		newQuantity := productData.Available + req.Delta
		if newQuantity < 0 {
			result = &UpdateResult{
				Success: false,
				Error:   fmt.Errorf("insufficient inventory: current %d, delta %d", productData.Available, req.Delta),
				Applied: false,
			}
			s.cacheIdempotencyResult(req.IdempotencyKey, result)
			return
		}

		// Apply the update
		newVersion := productData.Version + 1
		lastUpdated := time.Now().UTC().Format(time.RFC3339)

		productData.Available = newQuantity
		productData.Version = newVersion
		productData.LastUpdated = lastUpdated
		s.data.Products[req.ProductID] = productData

		// Update global metadata (requires brief global lock)
		s.globalMutex.Lock()
		s.data.Metadata.LastOffset++
		s.data.Metadata.LastUpdated = lastUpdated
		s.globalMutex.Unlock()

		result = &UpdateResult{
			Success:     true,
			NewQuantity: newQuantity,
			NewVersion:  newVersion,
			Applied:     true,
			LastUpdated: lastUpdated,
		}

		// Cache the result for idempotency
		s.cacheIdempotencyResult(req.IdempotencyKey, result)

		slog.Info("Inventory update applied successfully",
			"product_id", req.ProductID,
			"old_quantity", productData.Available-req.Delta,
			"new_quantity", newQuantity,
			"old_version", req.Version,
			"new_version", newVersion,
			"delta", req.Delta,
			"idempotency_key", req.IdempotencyKey)
	})

	// Persist changes to JSON file if enabled (outside of product lock)
	if result.Success {
		if saveErr := s.saveDataToFile(); saveErr != nil {
			slog.Error("Failed to persist inventory data to file",
				"error", saveErr,
				"product_id", req.ProductID)
			// Note: We don't fail the update operation if file save fails
			// The in-memory state is still consistent
		}
	}

	return result
}

// cacheIdempotencyResult stores the result for future idempotent requests
func (s *InventoryService) cacheIdempotencyResult(key string, result *UpdateResult) {
	s.idempotencyCache.Set(key, result)
}

// saveDataToFile persists the current inventory data to the JSON file
func (s *InventoryService) saveDataToFile() error {
	if !s.enableJSONPersistence {
		slog.Debug("JSON persistence disabled, skipping file save")
		return nil
	}

	slog.Debug("Saving inventory data to file", "path", s.dataFilePath)

	// Use global read lock to get consistent snapshot of data
	s.globalMutex.RLock()

	// Marshal the data to JSON with indentation for readability
	jsonData, err := json.MarshalIndent(s.data, "", "  ")
	productsCount := len(s.data.Products)
	lastOffset := s.data.Metadata.LastOffset

	s.globalMutex.RUnlock()

	if err != nil {
		return fmt.Errorf("error marshaling inventory data: %w", err)
	}

	// Write to file atomically by writing to a temp file first
	tempFilePath := s.dataFilePath + ".tmp"
	err = os.WriteFile(tempFilePath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("error writing temp file: %w", err)
	}

	// Atomically replace the original file
	err = os.Rename(tempFilePath, s.dataFilePath)
	if err != nil {
		// Clean up temp file if rename fails
		os.Remove(tempFilePath)
		return fmt.Errorf("error replacing original file: %w", err)
	}

	slog.Info("Inventory data saved to file successfully",
		"path", s.dataFilePath,
		"products_count", productsCount,
		"last_offset", lastOffset)

	return nil
}

// GetCacheStats returns statistics about the idempotency cache
func (s *InventoryService) GetCacheStats() map[string]interface{} {
	return s.idempotencyCache.GetStats()
}

// GetLockStats returns statistics about the product lock manager
func (s *InventoryService) GetLockStats() map[string]interface{} {
	return s.productLockManager.GetLockStats()
}

// Stop gracefully shuts down the inventory service
func (s *InventoryService) Stop() {
	slog.Info("Stopping inventory service", "worker_count", s.workerCount)

	// Signal all workers to stop
	close(s.stopWorkers)

	// Wait for all workers to finish processing current requests
	s.workersWaitGroup.Wait()

	// Close the update queue
	close(s.updateQueue)

	// Stop the idempotency cache
	if s.idempotencyCache != nil {
		s.idempotencyCache.Stop()
	}

	slog.Info("Inventory service stopped successfully")
}

// UpdateInventory submits an inventory update request to the queue and waits for the result
func (s *InventoryService) UpdateInventory(productID string, delta, version int, idempotencyKey, storeID string) (*UpdateResult, error) {
	// Create response channel
	responseChan := make(chan *UpdateResult, 1)

	// Create update request
	updateReq := &UpdateRequest{
		ProductID:      productID,
		Delta:          delta,
		Version:        version,
		IdempotencyKey: idempotencyKey,
		StoreID:        storeID,
		ResponseChan:   responseChan,
	}

	slog.Debug("Submitting update to queue",
		"product_id", productID,
		"delta", delta,
		"version", version,
		"idempotency_key", idempotencyKey)

	// Submit to queue
	select {
	case s.updateQueue <- updateReq:
		// Successfully queued
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("timeout submitting update to queue")
	}

	// Wait for result
	select {
	case result := <-responseChan:
		return result, nil
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("timeout waiting for update result")
	}
}
