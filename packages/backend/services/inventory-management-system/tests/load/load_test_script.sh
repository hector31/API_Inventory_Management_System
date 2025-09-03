#!/bin/bash

# Load Testing Script for Inventory Management API
# Tests fine-grained locking implementation with concurrent requests
# Run from: API_Inventory_Management_System/tests/load/

SERVER_URL="http://localhost:8081"
API_KEY="demo"
CONCURRENT_CLIENTS=10
TOTAL_REQUESTS=100

echo "🚀 Starting Load Tests for Inventory Management API"
echo "Server: $SERVER_URL"
echo "Concurrent Clients: $CONCURRENT_CLIENTS"
echo "Total Requests per Test: $TOTAL_REQUESTS"
echo "Working Directory: $(pwd)"
echo "=========================================="

# Check if hey is installed
if ! command -v hey &> /dev/null; then
    echo "❌ 'hey' tool is not installed. Please install it first:"
    echo "   go install github.com/rakyll/hey@latest"
    exit 1
fi

# Check if server is running
echo "🔍 Checking if server is running..."
if ! curl -s -H "X-API-Key: $API_KEY" "$SERVER_URL/health" > /dev/null; then
    echo "❌ Server is not running on $SERVER_URL"
    echo "Please start the server first: go run ./cmd/server"
    exit 1
fi
echo "✅ Server is running"

# Function to fetch current product version
get_product_version() {
    local product_id=$1
    local response=$(curl -s -H "X-API-Key: $API_KEY" "$SERVER_URL/v1/inventory/$product_id")
    if [ $? -eq 0 ] && [ -n "$response" ]; then
        echo "$response" | grep -o '"version":[0-9]*' | cut -d':' -f2
    else
        echo "❌ Failed to fetch version for $product_id" >&2
        return 1
    fi
}

# Function to generate dynamic payload
generate_dynamic_payload() {
    local product_id=$1
    local delta=$2
    local test_name=$3
    local version=$(get_product_version "$product_id")

    if [ -z "$version" ]; then
        echo "❌ Could not get version for $product_id" >&2
        return 1
    fi

    local random_id=$(date +%s%N | cut -b1-13)
    local idempotency_key="${test_name}-${product_id}-${random_id}"

    echo "{\"storeId\":\"store-1\",\"productId\":\"$product_id\",\"delta\":$delta,\"version\":$version,\"idempotencyKey\":\"$idempotency_key\"}"
}

# Function to generate conflict test payload with placeholder for unique idempotency keys
generate_conflict_payload_template() {
    local product_id=$1
    local delta=$2
    local test_name=$3
    local version=$(get_product_version "$product_id")

    if [ -z "$version" ]; then
        echo "❌ Could not get version for $product_id" >&2
        return 1
    fi

    # Use placeholder that will be replaced by hey with unique values
    echo "{\"storeId\":\"store-1\",\"productId\":\"$product_id\",\"delta\":$delta,\"version\":$version,\"idempotencyKey\":\"$test_name-$product_id-{{.RequestNumber}}\"}"
}

# Function to generate dynamic batch payload
generate_batch_payload() {
    local test_name=$1
    local random_id=$(date +%s%N | cut -b1-13)

    local sku001_version=$(get_product_version "SKU-001")
    local sku002_version=$(get_product_version "SKU-002")
    local sku003_version=$(get_product_version "SKU-003")

    if [ -z "$sku001_version" ] || [ -z "$sku002_version" ] || [ -z "$sku003_version" ]; then
        echo "❌ Could not get versions for batch test" >&2
        return 1
    fi

    cat << EOF
{
  "storeId": "store-1",
  "updates": [
    {
      "productId": "SKU-001",
      "delta": 1,
      "version": $sku001_version,
      "idempotencyKey": "${test_name}-sku001-${random_id}"
    },
    {
      "productId": "SKU-002",
      "delta": 1,
      "version": $sku002_version,
      "idempotencyKey": "${test_name}-sku002-${random_id}"
    },
    {
      "productId": "SKU-003",
      "delta": 1,
      "version": $sku003_version,
      "idempotencyKey": "${test_name}-sku003-${random_id}"
    }
  ]
}
EOF
}

echo ""
echo "📋 Test Scenario 1: Different Products (Should Run Concurrently)"
echo "Testing updates to different products - should all succeed"
echo "Expected: All requests succeed, processed by different workers concurrently"

echo "🔄 Fetching current product versions..."
echo "Generating dynamic payload for SKU-001..."
generate_dynamic_payload "SKU-001" 1 "concurrent-test" > temp_sku001.json

if [ $? -ne 0 ]; then
    echo "❌ Failed to generate payload for SKU-001"
    exit 1
fi

echo "✅ Dynamic payload generated for SKU-001"
echo "Payload content: $(cat temp_sku001.json)"

hey -n $TOTAL_REQUESTS -c $CONCURRENT_CLIENTS \
    -m POST \
    -H "X-API-Key: $API_KEY" \
    -H "Content-Type: application/json" \
    -D temp_sku001.json \
    "$SERVER_URL/v1/inventory/updates"

echo ""
echo "📋 Test Scenario 2: Same Product Updates (Should Have Version Conflicts)"
echo "Testing multiple updates to same product with same version"
echo "Expected: Only 1 request succeeds, others get 409 Conflict (version conflicts)"

echo "🔄 Getting current version for SKU-002..."
SKU002_VERSION=$(get_product_version "SKU-002")

if [ -z "$SKU002_VERSION" ]; then
    echo "❌ Failed to get version for SKU-002"
    exit 1
fi

echo "✅ Current SKU-002 version: $SKU002_VERSION"
echo "🔄 Running concurrent requests with SAME version ($SKU002_VERSION) to trigger conflicts..."

# Run multiple concurrent requests in background to ensure they all use the same version
# but have different idempotency keys to avoid idempotency cache hits
echo "Starting $CONCURRENT_CLIENTS concurrent requests..."

for i in $(seq 1 $CONCURRENT_CLIENTS); do
    {
        local_random=$(date +%s%N | tail -c 6)
        curl -s -X POST \
            -H "X-API-Key: $API_KEY" \
            -H "Content-Type: application/json" \
            -d "{\"storeId\":\"store-1\",\"productId\":\"SKU-002\",\"delta\":1,\"version\":$SKU002_VERSION,\"idempotencyKey\":\"conflict-test-$i-$local_random\"}" \
            "$SERVER_URL/v1/inventory/updates" \
            -w "Request $i: HTTP %{http_code} - %{json}\n"
    } &
done

# Wait for all background requests to complete
wait

echo "✅ All concurrent requests completed"

echo ""
echo "📋 Test Scenario 3: Batch Updates with Mixed Products"
echo "Testing batch updates with multiple products"
echo "Expected: Batch operations succeed, individual products processed concurrently"

echo "🔄 Generating dynamic batch payload with current versions..."
generate_batch_payload "batch-test" > temp_batch_mixed.json

if [ $? -ne 0 ]; then
    echo "❌ Failed to generate batch payload"
    exit 1
fi

echo "✅ Dynamic batch payload generated"
echo "Payload content: $(cat temp_batch_mixed.json)"

hey -n 50 -c 5 \
    -m POST \
    -H "X-API-Key: $API_KEY" \
    -H "Content-Type: application/json" \
    -D temp_batch_mixed.json \
    "$SERVER_URL/v1/inventory/updates"

echo ""
echo "📋 Test Scenario 4: High Concurrency Mixed Load"
echo "Testing mixed load with different products and high concurrency"
echo "Expected: High throughput, no deadlocks, proper worker distribution"

echo "🔄 Generating dynamic payloads for concurrent tests..."

# Generate dynamic payloads for different products
echo "Generating payload for SKU-001..."
generate_dynamic_payload "SKU-001" 1 "concurrent-sku001" > temp_concurrent_sku001.json
if [ $? -ne 0 ]; then
    echo "❌ Failed to generate payload for SKU-001"
    exit 1
fi

echo "Generating payload for SKU-002..."
generate_dynamic_payload "SKU-002" 1 "concurrent-sku002" > temp_concurrent_sku002.json
if [ $? -ne 0 ]; then
    echo "❌ Failed to generate payload for SKU-002"
    exit 1
fi

echo "Generating payload for SKU-003..."
generate_dynamic_payload "SKU-003" 1 "concurrent-sku003" > temp_concurrent_sku003.json
if [ $? -ne 0 ]; then
    echo "❌ Failed to generate payload for SKU-003"
    exit 1
fi

echo "✅ All concurrent test payloads generated"
echo "Starting concurrent tests on multiple products..."

# Run concurrent tests on different products simultaneously
hey -n 50 -c 5 -m POST -H "X-API-Key: $API_KEY" -H "Content-Type: application/json" -D temp_concurrent_sku001.json "$SERVER_URL/v1/inventory/updates" &
hey -n 50 -c 5 -m POST -H "X-API-Key: $API_KEY" -H "Content-Type: application/json" -D temp_concurrent_sku002.json "$SERVER_URL/v1/inventory/updates" &
hey -n 50 -c 5 -m POST -H "X-API-Key: $API_KEY" -H "Content-Type: application/json" -D temp_concurrent_sku003.json "$SERVER_URL/v1/inventory/updates" &

# Wait for all background processes to complete
wait

echo ""
echo "🧹 Cleaning up temporary files..."
rm -f temp_*.json

echo ""
echo "✅ Load testing completed!"
echo ""
echo "📊 What to look for in the results:"
echo "1. Different products: High success rate (200 OK responses)"
echo "2. Same product: Mix of 200 OK and 409 Conflict responses"
echo "3. No 500 Internal Server Error responses"
echo "4. Server logs should show worker distribution"
echo "5. Response times should be reasonable (< 100ms for most requests)"
echo ""
echo "📝 Check server logs for:"
echo "- Worker distribution across multiple workers"
echo "- Product lock acquisition/release messages"
echo "- Version conflict warnings"
echo "- No deadlock or timeout errors"
