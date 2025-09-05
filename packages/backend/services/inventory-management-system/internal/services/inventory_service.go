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
	"inventory-management-api/internal/events"
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
	eventQueue            *events.EventQueue
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
	Success      bool
	NewQuantity  int
	NewVersion   int
	ErrorType    string
	ErrorMessage string
	Applied      bool
	LastUpdated  string
}

// InventoryData represents the complete inventory data structure
type InventoryData struct {
	Products map[string]ProductData `json:"products"`
	Metadata MetadataData           `json:"metadata"`
}

// ProductData represents complete product data
type ProductData struct {
	ProductID   string  `json:"productId"`
	Name        string  `json:"name"`
	Available   int     `json:"available"`
	Version     int     `json:"version"`
	LastUpdated string  `json:"lastUpdated"`
	Price       float64 `json:"price"`
}

// MetadataData represents system metadata for replication and caching
type MetadataData struct {
	LastOffset    int    `json:"lastOffset"`    // Last event sequence number for replication
	TotalProducts int    `json:"totalProducts"` // Quick count of total products
	LastUpdated   string `json:"lastUpdated"`   // System-wide last update timestamp
}

const (
	// Error types
	ErrTypeProductNotFound       = "product_not_found"
	ErrTypeVersionConflict       = "version_conflict"
	ErrTypeInvalidRequest        = "invalid_request"
	ErrTypeInvalidDelta          = "invalid_delta"
	ErrTypeInsufficientInventory = "insufficient_inventory"
	ErrTypeTimeout               = "timeout"
	ErrTypeInternalError         = "internal_error"
	ErrTypeUnknown               = "unknown_error"
	ErrTypeInvalidIdempotencyKey = "invalid_idempotency_key"
	ErrTypeMissingProductID      = "missing_product_id"
	ErrTypeNotFound              = "not_found"
	ErrTypeValidation            = "validation_error"
)

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
			Name:        productData.Name,
			Available:   productData.Available,
			Version:     productData.Version,
			LastUpdated: productData.LastUpdated,
			Price:       productData.Price,
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
			Name:        productData.Name,
			Available:   productData.Available,
			Version:     productData.Version,
			LastUpdated: productData.LastUpdated,
			Price:       productData.Price,
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
			// Process update with timeout protection
			resultChan := make(chan *UpdateResult, 1)
			go func() {
				result := s.processUpdateInternal(updateReq)
				resultChan <- result
			}()

			var result *UpdateResult
			select {
			case result = <-resultChan:
				// Update completed successfully
			case <-time.After(15 * time.Second):
				// Update processing timed out
				slog.Error("Update processing timed out",
					"worker_id", workerID,
					"product_id", updateReq.ProductID,
					"idempotency_key", updateReq.IdempotencyKey)
				result = &UpdateResult{
					Success:      false,
					ErrorType:    ErrTypeTimeout,
					ErrorMessage: "update processing timed out",
					Applied:      false,
					NewQuantity:  0,
					NewVersion:   0,
					LastUpdated:  "",
				}
			}

			// Send result back through response channel with timeout
			select {
			case updateReq.ResponseChan <- result:
				// Successfully sent response
				slog.Debug("Update processed by worker",
					"worker_id", workerID,
					"product_id", updateReq.ProductID,
					"applied", result.Applied)
			case <-time.After(2 * time.Second):
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

// SetEventQueue sets the event queue for publishing events and synchronizes metadata
func (s *InventoryService) SetEventQueue(eventQueue *events.EventQueue) {
	s.eventQueue = eventQueue

	// Synchronize metadata with event queue's current offset for perfect consistency
	s.globalMutex.Lock()
	currentOffset := eventQueue.GetCurrentOffset()

	slog.Debug("SetEventQueue synchronization check",
		"db_offset", s.data.Metadata.LastOffset,
		"queue_offset", currentOffset)

	// If the database metadata has a higher offset than the event queue,
	// it means the event queue was reset but the database wasn't.
	// In this case, we should use the database's offset as the starting point.
	if s.data.Metadata.LastOffset > int(currentOffset) {
		slog.Warn("Database metadata offset is higher than event queue offset",
			"db_offset", s.data.Metadata.LastOffset,
			"queue_offset", currentOffset,
			"action", "keeping_database_offset")
		if int(currentOffset) == 0 {
			slog.Info("Event queue offset is 0, resetting database offset for consistency")
			// Reset database offset directly since we already have the mutex
			s.resetDatabaseOffsetInternal("file_not_exist")
		}
	} else {
		// Update database metadata to match event queue offset
		s.data.Metadata.LastOffset = int(currentOffset)
		slog.Info("Synchronized database metadata with event queue offset",
			"offset", currentOffset)
	}
	s.globalMutex.Unlock()
}

// ResetDatabaseOffset resets the database offset to 0 to maintain consistency with event queue reset
func (s *InventoryService) ResetDatabaseOffset() {
	s.ResetDatabaseOffsetWithReason("event_queue_reset")
}

// ResetDatabaseOffsetWithReason resets the database offset to 0 with a specific reason
func (s *InventoryService) ResetDatabaseOffsetWithReason(reason string) {
	// Use a goroutine to avoid potential deadlock when called from within a locked context
	go func() {
		s.globalMutex.Lock()
		defer s.globalMutex.Unlock()
		s.resetDatabaseOffsetInternal(reason)
	}()
}

// resetDatabaseOffsetInternal performs the actual reset without acquiring mutex (internal use only)
func (s *InventoryService) resetDatabaseOffsetInternal(reason string) {
	oldOffset := s.data.Metadata.LastOffset
	s.data.Metadata.LastOffset = 0
	s.data.Metadata.LastUpdated = time.Now().UTC().Format(time.RFC3339)

	var logMessage string
	switch reason {
	case "file_not_exist":
		logMessage = "Database offset reset to maintain consistency - events file doesn't exist, starting fresh"
	case "file_load_failure":
		logMessage = "Database offset reset to maintain consistency - events file corrupted or invalid"
	default:
		logMessage = "Database offset reset to maintain consistency with event queue"
	}

	slog.Info(logMessage,
		"old_offset", oldOffset,
		"new_offset", 0,
		"reason", reason)

	// Persist the reset offset to file if persistence is enabled
	if err := s.saveDataToFileInternal(); err != nil {
		slog.Error("Failed to persist database offset reset to file", "error", err, "reason", reason)
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
				Success:      false,
				ErrorMessage: fmt.Sprintf("product not found: %s", req.ProductID),
				ErrorType:    ErrTypeProductNotFound,
				Applied:      false,
				NewQuantity:  0,
				NewVersion:   0,
				LastUpdated:  "",
			}
			s.cacheIdempotencyResult(req.IdempotencyKey, result)
			return
		}

		// Check version for OCC
		if productData.Version != req.Version {
			result = &UpdateResult{
				Success:      false,
				ErrorMessage: fmt.Sprintf("version conflict: expected %d, got %d", productData.Version, req.Version),
				ErrorType:    ErrTypeVersionConflict,
				Applied:      false,
				NewQuantity:  productData.Available, // Return current quantity
				NewVersion:   productData.Version,   // Return current version
				LastUpdated:  productData.LastUpdated,
			}
			s.cacheIdempotencyResult(req.IdempotencyKey, result)

			slog.Warn("Version conflict detected",
				"product_id", req.ProductID,
				"expected_version", productData.Version,
				"provided_version", req.Version,
				"idempotency_key", req.IdempotencyKey)

			return
		}

		// stores only negative quantities
		if req.Delta > 0 {
			result = &UpdateResult{
				Success:      false,
				ErrorMessage: fmt.Sprintf("invalid delta: %d - only negative quantities are allowed", req.Delta),
				ErrorType:    ErrTypeInvalidRequest,
				Applied:      false,
			}
			s.cacheIdempotencyResult(req.IdempotencyKey, result)
			return
		}

		// Calculate new quantity
		newQuantity := productData.Available + req.Delta
		if newQuantity < 0 {
			result = &UpdateResult{
				Success:      false,
				ErrorMessage: fmt.Sprintf("insufficient inventory: current %d, delta %d", productData.Available, req.Delta),
				ErrorType:    ErrTypeInsufficientInventory,
				Applied:      false,
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

	// Publish event if update was successful and event queue is available
	// Do this asynchronously to prevent blocking the response
	if result.Success && s.eventQueue != nil {
		go func() {
			// Get the complete product data to include in the event
			var productName string
			var productPrice float64
			s.productLockManager.WithProductReadLock(req.ProductID, func() {
				if productData, exists := s.data.Products[req.ProductID]; exists {
					productName = productData.Name
					productPrice = productData.Price
				}
			})

			// Create event data with complete product information
			eventData := models.ProductResponse{
				ProductID:   req.ProductID,
				Name:        productName,
				Available:   result.NewQuantity,
				Version:     result.NewVersion,
				LastUpdated: result.LastUpdated,
				Price:       productPrice,
			}

			s.eventQueue.PublishEvent(
				models.EventTypeProductUpdated,
				req.ProductID,
				eventData,
				result.NewVersion,
			)

			// Update metadata with current event offset for snapshot synchronization
			s.globalMutex.Lock()
			currentOffset := s.eventQueue.GetCurrentOffset()
			s.data.Metadata.LastOffset = int(currentOffset)
			s.data.Metadata.LastUpdated = result.LastUpdated
			s.globalMutex.Unlock()

			slog.Debug("Event published for inventory update",
				"product_id", req.ProductID,
				"product_name", productName,
				"event_type", models.EventTypeProductUpdated,
				"new_version", result.NewVersion,
				"new_quantity", result.NewQuantity,
				"current_offset", currentOffset)
		}()
	}

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
	slog.Debug("saveDataToFile called", "enableJSONPersistence", s.enableJSONPersistence)
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

// saveDataToFileInternal saves data to file without acquiring mutex (internal use only)
func (s *InventoryService) saveDataToFileInternal() error {
	if !s.enableJSONPersistence {
		slog.Debug("JSON persistence disabled, skipping file save")
		return nil
	}

	slog.Debug("Saving inventory data to file (internal)", "path", s.dataFilePath)

	// Marshal the data to JSON with indentation for readability
	// Note: No mutex needed here as this is called from within a locked context
	jsonData, err := json.MarshalIndent(s.data, "", "  ")
	productsCount := len(s.data.Products)
	lastOffset := s.data.Metadata.LastOffset

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

	slog.Info("Inventory data saved to file successfully (internal)",
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

	// Wait for result with extended timeout to account for file I/O
	select {
	case result := <-responseChan:
		return result, nil
	case <-time.After(20 * time.Second):
		slog.Error("Timeout waiting for update result",
			"product_id", productID,
			"idempotency_key", idempotencyKey,
			"timeout", "20s")
		return nil, fmt.Errorf("timeout waiting for update result after 20 seconds")
	}
}

// AdminSetProducts performs admin-level product updates with OCC
func (s *InventoryService) AdminSetProducts(products []models.AdminProductUpdate) (*models.AdminSetResponse, error) {
	slog.Info("Processing admin set request", "product_count", len(products))

	results := make([]models.AdminProductResult, 0, len(products))
	successCount := 0
	failureCount := 0

	for _, productUpdate := range products {
		result := s.processAdminProductUpdate(productUpdate)
		results = append(results, result)

		if result.Success {
			successCount++
		} else {
			failureCount++
		}
	}

	// Persist changes to JSON file if any update was successful
	if successCount > 0 {
		slog.Debug("Attempting to persist admin set changes to file", "successful_updates", successCount)
		if saveErr := s.saveDataToFile(); saveErr != nil {
			slog.Error("Failed to persist inventory data to file after admin set",
				"error", saveErr,
				"successful_updates", successCount)
			// Note: We don't fail the set operation if file save fails
			// The in-memory state is still consistent
		} else {
			slog.Debug("Successfully persisted admin set changes to file", "successful_updates", successCount)
		}
	}

	response := &models.AdminSetResponse{
		Results: results,
		Summary: models.AdminSetSummary{
			TotalRequests:     len(products),
			SuccessfulUpdates: successCount,
			FailedUpdates:     failureCount,
		},
	}

	slog.Info("Admin set request completed",
		"total", len(products),
		"successful", successCount,
		"failed", failureCount)

	return response, nil
}

// processAdminProductUpdate handles a single admin product update with OCC
func (s *InventoryService) processAdminProductUpdate(update models.AdminProductUpdate) models.AdminProductResult {
	slog.Debug("Processing admin product update", "product_id", update.ProductID)

	// Use product-level locking for OCC
	var result models.AdminProductResult
	var productData ProductData
	var exists bool

	s.productLockManager.WithProductWriteLock(update.ProductID, func() {
		// Check if product exists
		productData, exists = s.data.Products[update.ProductID]
		if !exists {
			result = models.AdminProductResult{
				ProductID:    update.ProductID,
				Success:      false,
				ErrorType:    ErrTypeNotFound,
				ErrorMessage: "Product not found",
			}
			return
		}

		// Create updated product data
		updatedProduct := productData // Copy existing data
		hasChanges := false

		// Apply partial updates
		if update.Name != nil {
			updatedProduct.Name = *update.Name
			hasChanges = true
		}
		if update.Available != nil {
			updatedProduct.Available = *update.Available
			hasChanges = true
		}
		if update.Price != nil {
			updatedProduct.Price = *update.Price
			hasChanges = true
		}

		if !hasChanges {
			result = models.AdminProductResult{
				ProductID:    update.ProductID,
				Success:      false,
				ErrorType:    ErrTypeValidation,
				ErrorMessage: "No fields to update",
			}
			return
		}

		// Update version and timestamp (OCC)
		updatedProduct.Version++
		updatedProduct.LastUpdated = time.Now().Format(time.RFC3339)

		// Apply the update
		s.data.Products[update.ProductID] = updatedProduct

		result = models.AdminProductResult{
			ProductID:   update.ProductID,
			Success:     true,
			NewVersion:  updatedProduct.Version,
			LastUpdated: updatedProduct.LastUpdated,
		}

		slog.Debug("Admin product update successful",
			"product_id", update.ProductID,
			"new_version", updatedProduct.Version,
			"name_updated", update.Name != nil,
			"available_updated", update.Available != nil,
			"price_updated", update.Price != nil)
	})

	// Publish event if update was successful
	if result.Success && s.eventQueue != nil {
		go func() {
			// Get the complete updated product data
			s.productLockManager.WithProductReadLock(update.ProductID, func() {
				if updatedProductData, exists := s.data.Products[update.ProductID]; exists {
					eventData := models.ProductResponse{
						ProductID:   update.ProductID,
						Name:        updatedProductData.Name,
						Available:   updatedProductData.Available,
						Version:     updatedProductData.Version,
						LastUpdated: updatedProductData.LastUpdated,
						Price:       updatedProductData.Price,
					}

					s.eventQueue.PublishEvent(
						models.EventTypeProductUpdated,
						update.ProductID,
						eventData,
						updatedProductData.Version,
					)

					// Update metadata with current event offset
					s.globalMutex.Lock()
					currentOffset := s.eventQueue.GetCurrentOffset()
					s.data.Metadata.LastOffset = int(currentOffset)
					s.data.Metadata.LastUpdated = updatedProductData.LastUpdated
					s.globalMutex.Unlock()

					slog.Debug("Event published for admin product update",
						"product_id", update.ProductID,
						"event_type", models.EventTypeProductUpdated,
						"new_version", updatedProductData.Version,
						"current_offset", currentOffset)
				}
			})
		}()
	}

	return result
}

// AdminCreateProducts performs admin-level product creation with OCC
func (s *InventoryService) AdminCreateProducts(products []models.AdminProductCreate) (*models.AdminCreateResponse, error) {
	slog.Info("Processing admin create request", "product_count", len(products))

	results := make([]models.AdminProductResult, 0, len(products))
	successCount := 0
	failureCount := 0

	for _, productCreate := range products {
		result := s.processAdminProductCreate(productCreate)
		results = append(results, result)

		if result.Success {
			successCount++
		} else {
			failureCount++
		}
	}

	// Persist changes to JSON file if any creation was successful
	if successCount > 0 {
		slog.Debug("Attempting to persist admin create changes to file", "successful_creations", successCount)
		if saveErr := s.saveDataToFile(); saveErr != nil {
			slog.Error("Failed to persist inventory data to file after admin create",
				"error", saveErr,
				"successful_creations", successCount)
			// Note: We don't fail the create operation if file save fails
			// The in-memory state is still consistent
		} else {
			slog.Debug("Successfully persisted admin create changes to file", "successful_creations", successCount)
		}
	}

	response := &models.AdminCreateResponse{
		Results: results,
		Summary: models.AdminCreateSummary{
			TotalRequests:       len(products),
			SuccessfulCreations: successCount,
			FailedCreations:     failureCount,
		},
	}

	slog.Info("Admin create request completed",
		"total", len(products),
		"successful", successCount,
		"failed", failureCount)

	return response, nil
}

// processAdminProductCreate handles a single admin product creation with OCC
func (s *InventoryService) processAdminProductCreate(create models.AdminProductCreate) models.AdminProductResult {
	slog.Debug("Processing admin product creation", "product_id", create.ProductID)

	// Use product-level locking for OCC
	var result models.AdminProductResult

	s.productLockManager.WithProductWriteLock(create.ProductID, func() {
		// Check if product already exists
		if _, exists := s.data.Products[create.ProductID]; exists {
			result = models.AdminProductResult{
				ProductID:    create.ProductID,
				Success:      false,
				ErrorType:    ErrTypeValidation,
				ErrorMessage: "Product already exists",
			}
			return
		}

		// Create new product data
		newProduct := ProductData{
			ProductID:   create.ProductID,
			Name:        create.Name,
			Available:   create.Available,
			Price:       create.Price,
			Version:     1, // Start with version 1
			LastUpdated: time.Now().Format(time.RFC3339),
		}

		// Add the product
		s.data.Products[create.ProductID] = newProduct

		// Update metadata
		s.data.Metadata.TotalProducts++
		s.data.Metadata.LastUpdated = newProduct.LastUpdated

		result = models.AdminProductResult{
			ProductID:   create.ProductID,
			Success:     true,
			NewVersion:  newProduct.Version,
			LastUpdated: newProduct.LastUpdated,
		}

		slog.Debug("Admin product creation successful",
			"product_id", create.ProductID,
			"name", create.Name,
			"available", create.Available,
			"price", create.Price)
	})

	// Publish event if creation was successful
	if result.Success && s.eventQueue != nil {
		go func() {
			// Get the complete created product data
			s.productLockManager.WithProductReadLock(create.ProductID, func() {
				if createdProductData, exists := s.data.Products[create.ProductID]; exists {
					eventData := models.ProductResponse{
						ProductID:   create.ProductID,
						Name:        createdProductData.Name,
						Available:   createdProductData.Available,
						Version:     createdProductData.Version,
						LastUpdated: createdProductData.LastUpdated,
						Price:       createdProductData.Price,
					}

					s.eventQueue.PublishEvent(
						models.EventTypeProductCreated,
						create.ProductID,
						eventData,
						createdProductData.Version,
					)

					// Update metadata with current event offset
					s.globalMutex.Lock()
					currentOffset := s.eventQueue.GetCurrentOffset()
					s.data.Metadata.LastOffset = int(currentOffset)
					s.data.Metadata.LastUpdated = createdProductData.LastUpdated
					s.globalMutex.Unlock()

					slog.Debug("Event published for admin product creation",
						"product_id", create.ProductID,
						"event_type", models.EventTypeProductCreated,
						"version", createdProductData.Version,
						"current_offset", currentOffset)
				}
			})
		}()
	}

	return result
}

// AdminDeleteProducts performs admin-level product deletion with OCC
func (s *InventoryService) AdminDeleteProducts(productIDs []string) (*models.AdminDeleteResponse, error) {
	slog.Info("Processing admin delete request", "product_count", len(productIDs))

	results := make([]models.AdminProductResult, 0, len(productIDs))
	successCount := 0
	failureCount := 0

	for _, productID := range productIDs {
		result := s.processAdminProductDelete(productID)
		results = append(results, result)

		if result.Success {
			successCount++
		} else {
			failureCount++
		}
	}

	// Persist changes to JSON file if any deletion was successful
	if successCount > 0 {
		slog.Debug("Attempting to persist admin delete changes to file", "successful_deletions", successCount)
		if saveErr := s.saveDataToFile(); saveErr != nil {
			slog.Error("Failed to persist inventory data to file after admin delete",
				"error", saveErr,
				"successful_deletions", successCount)
			// Note: We don't fail the delete operation if file save fails
			// The in-memory state is still consistent
		} else {
			slog.Debug("Successfully persisted admin delete changes to file", "successful_deletions", successCount)
		}
	}

	response := &models.AdminDeleteResponse{
		Results: results,
		Summary: models.AdminDeleteSummary{
			TotalRequests:       len(productIDs),
			SuccessfulDeletions: successCount,
			FailedDeletions:     failureCount,
		},
	}

	slog.Info("Admin delete request completed",
		"total", len(productIDs),
		"successful", successCount,
		"failed", failureCount)

	return response, nil
}

// processAdminProductDelete handles a single admin product deletion with OCC
func (s *InventoryService) processAdminProductDelete(productID string) models.AdminProductResult {
	slog.Debug("Processing admin product deletion", "product_id", productID)

	// Use product-level locking for OCC
	var result models.AdminProductResult
	var deletedProduct ProductData
	var existed bool

	s.productLockManager.WithProductWriteLock(productID, func() {
		// Check if product exists and get its data before deletion
		deletedProduct, existed = s.data.Products[productID]
		if !existed {
			result = models.AdminProductResult{
				ProductID:    productID,
				Success:      false,
				ErrorType:    ErrTypeNotFound,
				ErrorMessage: "Product not found",
			}
			return
		}

		// Delete the product
		delete(s.data.Products, productID)

		// Update metadata
		s.data.Metadata.TotalProducts--
		s.data.Metadata.LastUpdated = time.Now().Format(time.RFC3339)

		result = models.AdminProductResult{
			ProductID:   productID,
			Success:     true,
			NewVersion:  deletedProduct.Version + 1, // Increment version for deletion event
			LastUpdated: s.data.Metadata.LastUpdated,
		}

		slog.Debug("Admin product deletion successful",
			"product_id", productID,
			"name", deletedProduct.Name)
	})

	// Publish event if deletion was successful
	if result.Success && s.eventQueue != nil {
		go func() {
			// Create event data with the deleted product information
			eventData := models.ProductResponse{
				ProductID:   productID,
				Name:        deletedProduct.Name,
				Available:   deletedProduct.Available,
				Version:     result.NewVersion,
				LastUpdated: result.LastUpdated,
				Price:       deletedProduct.Price,
			}

			s.eventQueue.PublishEvent(
				models.EventTypeProductDeleted,
				productID,
				eventData,
				result.NewVersion,
			)

			// Update metadata with current event offset
			s.globalMutex.Lock()
			currentOffset := s.eventQueue.GetCurrentOffset()
			s.data.Metadata.LastOffset = int(currentOffset)
			s.data.Metadata.LastUpdated = result.LastUpdated
			s.globalMutex.Unlock()

			slog.Debug("Event published for admin product deletion",
				"product_id", productID,
				"event_type", models.EventTypeProductDeleted,
				"version", result.NewVersion,
				"current_offset", currentOffset)
		}()
	}

	return result
}
