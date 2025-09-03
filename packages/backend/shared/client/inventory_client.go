package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/melibackend/shared/models"
)

// InventoryClient provides methods to interact with the central inventory API
type InventoryClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewInventoryClient creates a new inventory client
func NewInventoryClient(baseURL, apiKey string) *InventoryClient {
	return &InventoryClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// HealthCheck checks the health of the central inventory API
func (c *InventoryClient) HealthCheck() (*models.HealthResponse, error) {
	url := fmt.Sprintf("%s/health", c.baseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}

	var health models.HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &health, nil
}

// GetProduct retrieves a product from the central inventory API
func (c *InventoryClient) GetProduct(productID string) (*models.Product, error) {
	url := fmt.Sprintf("%s/v1/inventory/%s", c.baseURL, productID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("product not found: %s", productID)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var product models.Product
	if err := json.NewDecoder(resp.Body).Decode(&product); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &product, nil
}

// UpdateInventory sends an inventory update to the central API
func (c *InventoryClient) UpdateInventory(update models.UpdateRequest) (*models.UpdateResponse, error) {
	url := fmt.Sprintf("%s/v1/inventory/updates", c.baseURL)

	jsonData, err := json.Marshal(update)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var updateResp models.UpdateResponse
	if err := json.Unmarshal(body, &updateResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &updateResp, nil
}

// BatchUpdateInventory sends a batch inventory update to the central API
func (c *InventoryClient) BatchUpdateInventory(batchUpdate models.BatchUpdateRequest) (*models.BatchUpdateResponse, error) {
	url := fmt.Sprintf("%s/v1/inventory/updates", c.baseURL)

	jsonData, err := json.Marshal(batchUpdate)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var batchResp models.BatchUpdateResponse
	if err := json.Unmarshal(body, &batchResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &batchResp, nil
}

// GetAllProducts retrieves all products from the central inventory API
func (c *InventoryClient) GetAllProducts() ([]models.Product, error) {
	url := fmt.Sprintf("%s/v1/inventory", c.baseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Read the response body first to debug the format
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Try to parse as the expected format: {"items": [...], "nextCursor": ""}
	var response struct {
		Items      []models.Product `json:"items"`
		NextCursor string           `json:"nextCursor"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		// If that fails, try to parse as a direct array
		var directProducts []models.Product
		if err2 := json.Unmarshal(body, &directProducts); err2 != nil {
			return nil, fmt.Errorf("failed to decode response as object or array: %w (original: %v)", err2, err)
		}
		return directProducts, nil
	}

	return response.Items, nil
}
