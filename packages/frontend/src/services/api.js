// API service for communicating with Store S1 backend
const API_BASE_URL = process.env.NODE_ENV === 'production' 
  ? '/api'  // In production, use nginx proxy
  : 'http://localhost:8083/v1/store';  // In development, direct to store API

const API_KEY = 'demo';

// Helper function to make API requests
const apiRequest = async (endpoint, options = {}) => {
  const url = `${API_BASE_URL}${endpoint}`;
  
  const config = {
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': API_KEY,
      ...options.headers,
    },
    ...options,
  };

  try {
    const response = await fetch(url, config);
    
    if (!response.ok) {
      const errorText = await response.text();
      let errorMessage;
      
      try {
        const errorJson = JSON.parse(errorText);
        errorMessage = errorJson.message || errorJson.error || `HTTP ${response.status}`;
      } catch {
        errorMessage = errorText || `HTTP ${response.status}: ${response.statusText}`;
      }
      
      throw new Error(errorMessage);
    }

    const contentType = response.headers.get('content-type');
    if (contentType && contentType.includes('application/json')) {
      return await response.json();
    }
    
    return await response.text();
  } catch (error) {
    console.error(`API request failed: ${url}`, error);
    throw error;
  }
};

// Get all products from store's local cache
export const getProducts = async () => {
  try {
    const data = await apiRequest('/inventory');
    return Array.isArray(data) ? data : [];
  } catch (error) {
    console.error('Failed to fetch products:', error);
    throw new Error(`Failed to load products: ${error.message}`);
  }
};

// Get a specific product by ID
export const getProduct = async (productId) => {
  try {
    return await apiRequest(`/inventory/${productId}`);
  } catch (error) {
    console.error(`Failed to fetch product ${productId}:`, error);
    throw new Error(`Failed to load product: ${error.message}`);
  }
};

// Update inventory (purchase items)
export const updateInventory = async (productId, delta, idempotencyKey = null) => {
  try {
    const body = {
      product_id: productId,
      delta: delta,
      idempotency_key: idempotencyKey || `frontend-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`
    };

    return await apiRequest('/inventory', {
      method: 'PUT',
      body: JSON.stringify(body),
    });
  } catch (error) {
    console.error(`Failed to update inventory for ${productId}:`, error);
    throw new Error(`Failed to update inventory: ${error.message}`);
  }
};

// Get sync status
export const getSyncStatus = async () => {
  try {
    return await apiRequest('/sync/status');
  } catch (error) {
    console.error('Failed to fetch sync status:', error);
    throw new Error(`Failed to get sync status: ${error.message}`);
  }
};

// Force sync
export const forceSync = async () => {
  try {
    return await apiRequest('/sync/force', {
      method: 'POST',
    });
  } catch (error) {
    console.error('Failed to force sync:', error);
    throw new Error(`Failed to force sync: ${error.message}`);
  }
};

// Get cache stats
export const getCacheStats = async () => {
  try {
    return await apiRequest('/cache/stats');
  } catch (error) {
    console.error('Failed to fetch cache stats:', error);
    throw new Error(`Failed to get cache stats: ${error.message}`);
  }
};

// Health check
export const healthCheck = async () => {
  try {
    const response = await fetch(
      process.env.NODE_ENV === 'production' 
        ? '/api/../health'  // Go up one level from /api to reach /health
        : 'http://localhost:8083/health',
      {
        headers: {
          'X-API-Key': API_KEY,
        },
      }
    );
    
    if (!response.ok) {
      throw new Error(`Health check failed: ${response.status}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Health check failed:', error);
    throw new Error(`Health check failed: ${error.message}`);
  }
};
