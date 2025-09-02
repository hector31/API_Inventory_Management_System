# PowerShell Load Testing Script for Inventory Management API
# Tests fine-grained locking implementation with concurrent requests
# Run from: API_Inventory_Management_System/tests/load/

param(
    [string]$ServerUrl = "http://localhost:8081",
    [string]$ApiKey = "demo",
    [int]$ConcurrentClients = 10,
    [int]$TotalRequests = 100
)

Write-Host "🚀 Starting Load Tests for Inventory Management API" -ForegroundColor Green
Write-Host "Server: $ServerUrl"
Write-Host "Concurrent Clients: $ConcurrentClients"
Write-Host "Total Requests per Test: $TotalRequests"
Write-Host "Working Directory: $(Get-Location)"
Write-Host "=========================================="

# Check if hey is installed
if (-not (Get-Command hey -ErrorAction SilentlyContinue)) {
    Write-Host "❌ 'hey' tool is not installed. Please install it first:" -ForegroundColor Red
    Write-Host "   go install github.com/rakyll/hey@latest"
    exit 1
}

# Check if server is running
Write-Host "🔍 Checking if server is running..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "$ServerUrl/health" -Headers @{"X-API-Key" = $ApiKey} -UseBasicParsing -TimeoutSec 5
    Write-Host "✅ Server is running" -ForegroundColor Green
} catch {
    Write-Host "❌ Server is not running on $ServerUrl" -ForegroundColor Red
    Write-Host "Please start the server first: go run ./cmd/server"
    exit 1
}

# Function to fetch current product version
function Get-ProductVersion {
    param($ProductId)
    try {
        $response = Invoke-WebRequest -Uri "$ServerUrl/v1/inventory/$ProductId" -Headers @{"X-API-Key" = $ApiKey} -UseBasicParsing -TimeoutSec 10
        $jsonResponse = $response.Content | ConvertFrom-Json
        return $jsonResponse.version
    } catch {
        Write-Host "❌ Failed to fetch version for $ProductId" -ForegroundColor Red
        return $null
    }
}

# Function to generate dynamic payload
function Generate-DynamicPayload {
    param($ProductId, $Delta, $TestName)

    $version = Get-ProductVersion -ProductId $ProductId
    if ($null -eq $version) {
        Write-Host "❌ Could not get version for $ProductId" -ForegroundColor Red
        return $null
    }

    $randomId = [DateTimeOffset]::Now.ToUnixTimeMilliseconds()
    $idempotencyKey = "$TestName-$ProductId-$randomId"

    $payload = @{
        storeId = "store-1"
        productId = $ProductId
        delta = $Delta
        version = $version
        idempotencyKey = $idempotencyKey
    }

    return ($payload | ConvertTo-Json -Compress)
}

# Function to generate dynamic batch payload
function Generate-BatchPayload {
    param($TestName)

    $randomId = [DateTimeOffset]::Now.ToUnixTimeMilliseconds()

    $sku001Version = Get-ProductVersion -ProductId "SKU-001"
    $sku002Version = Get-ProductVersion -ProductId "SKU-002"
    $sku003Version = Get-ProductVersion -ProductId "SKU-003"

    if ($null -eq $sku001Version -or $null -eq $sku002Version -or $null -eq $sku003Version) {
        Write-Host "❌ Could not get versions for batch test" -ForegroundColor Red
        return $null
    }

    $batchPayload = @{
        storeId = "store-1"
        updates = @(
            @{
                productId = "SKU-001"
                delta = 1
                version = $sku001Version
                idempotencyKey = "$TestName-sku001-$randomId"
            },
            @{
                productId = "SKU-002"
                delta = 1
                version = $sku002Version
                idempotencyKey = "$TestName-sku002-$randomId"
            },
            @{
                productId = "SKU-003"
                delta = 1
                version = $sku003Version
                idempotencyKey = "$TestName-sku003-$randomId"
            }
        )
    }

    return ($batchPayload | ConvertTo-Json -Depth 3 -Compress)
}

Write-Host ""
Write-Host "📋 Test Scenario 1: Different Products (Should Run Concurrently)" -ForegroundColor Cyan
Write-Host "Testing updates to different products - should all succeed"
Write-Host "Expected: All requests succeed, processed by different workers concurrently"

Write-Host "🔄 Fetching current product versions..." -ForegroundColor Yellow
Write-Host "Generating dynamic payload for SKU-001..."
$payload = Generate-DynamicPayload -ProductId "SKU-001" -Delta 1 -TestName "concurrent-test"

if ($null -eq $payload) {
    Write-Host "❌ Failed to generate payload for SKU-001" -ForegroundColor Red
    exit 1
}

$payload | Out-File -FilePath "temp_sku001.json" -Encoding UTF8
Write-Host "✅ Dynamic payload generated for SKU-001" -ForegroundColor Green
Write-Host "Payload content: $payload"

& hey -n $TotalRequests -c $ConcurrentClients `
    -m POST `
    -H "X-API-Key: $ApiKey" `
    -H "Content-Type: application/json" `
    -D temp_sku001.json `
    "$ServerUrl/v1/inventory/updates"

Write-Host ""
Write-Host "📋 Test Scenario 2: Same Product Updates (Should Have Version Conflicts)" -ForegroundColor Cyan
Write-Host "Testing multiple updates to same product with same version"
Write-Host "Expected: Only 1 request succeeds, others get 409 Conflict (version conflicts)"

Write-Host "🔄 Getting current version for SKU-002..." -ForegroundColor Yellow
$sku002Version = Get-ProductVersion -ProductId "SKU-002"

if ($null -eq $sku002Version) {
    Write-Host "❌ Failed to get version for SKU-002" -ForegroundColor Red
    exit 1
}

Write-Host "✅ Current SKU-002 version: $sku002Version" -ForegroundColor Green
Write-Host "🔄 Running concurrent requests with SAME version ($sku002Version) to trigger conflicts..." -ForegroundColor Yellow

# Run multiple concurrent requests to ensure they all use the same version
# but have different idempotency keys to avoid idempotency cache hits
Write-Host "Starting $ConcurrentClients concurrent requests..."

$jobs = @()
for ($i = 1; $i -le $ConcurrentClients; $i++) {
    $job = Start-Job -ScriptBlock {
        param($ServerUrl, $ApiKey, $Version, $RequestId)

        $randomId = [DateTimeOffset]::Now.ToUnixTimeMilliseconds()
        $payload = @{
            storeId = "store-1"
            productId = "SKU-002"
            delta = 1
            version = $Version
            idempotencyKey = "conflict-test-$RequestId-$randomId"
        } | ConvertTo-Json -Compress

        try {
            $response = Invoke-WebRequest -Uri "$ServerUrl/v1/inventory/updates" `
                -Method POST `
                -Headers @{"X-API-Key" = $ApiKey; "Content-Type" = "application/json"} `
                -Body $payload `
                -UseBasicParsing

            return "Request $RequestId`: HTTP $($response.StatusCode) - $($response.Content)"
        } catch {
            return "Request $RequestId`: HTTP $($_.Exception.Response.StatusCode.Value__) - Error: $($_.Exception.Message)"
        }
    } -ArgumentList $ServerUrl, $ApiKey, $sku002Version, $i

    $jobs += $job
}

# Wait for all jobs to complete and show results
Write-Host "Waiting for all concurrent requests to complete..."
$jobs | ForEach-Object {
    $result = Receive-Job -Job $_ -Wait
    Write-Host $result
    Remove-Job -Job $_
}

Write-Host "✅ All concurrent requests completed" -ForegroundColor Green

Write-Host ""
Write-Host "📋 Test Scenario 3: Batch Updates with Mixed Products" -ForegroundColor Cyan
Write-Host "Testing batch updates with multiple products"
Write-Host "Expected: Batch operations succeed, individual products processed concurrently"

Write-Host "🔄 Generating dynamic batch payload with current versions..." -ForegroundColor Yellow
$batchPayload = Generate-BatchPayload -TestName "batch-test"

if ($null -eq $batchPayload) {
    Write-Host "❌ Failed to generate batch payload" -ForegroundColor Red
    exit 1
}

$batchPayload | Out-File -FilePath "temp_batch_mixed.json" -Encoding UTF8
Write-Host "✅ Dynamic batch payload generated" -ForegroundColor Green
Write-Host "Payload content: $batchPayload"

& hey -n 50 -c 5 `
    -m POST `
    -H "X-API-Key: $ApiKey" `
    -H "Content-Type: application/json" `
    -D temp_batch_mixed.json `
    "$ServerUrl/v1/inventory/updates"

Write-Host ""
Write-Host "📋 Test Scenario 4: High Concurrency Mixed Load" -ForegroundColor Cyan
Write-Host "Testing mixed load with different products and high concurrency"
Write-Host "Expected: High throughput, no deadlocks, proper worker distribution"

Write-Host "🔄 Generating dynamic payloads for concurrent tests..." -ForegroundColor Yellow

# Generate dynamic payloads for different products
Write-Host "Generating payload for SKU-001..."
$payload1 = Generate-DynamicPayload -ProductId "SKU-001" -Delta 1 -TestName "concurrent-sku001"
if ($null -eq $payload1) {
    Write-Host "❌ Failed to generate payload for SKU-001" -ForegroundColor Red
    exit 1
}
$payload1 | Out-File -FilePath "temp_concurrent_sku001.json" -Encoding UTF8

Write-Host "Generating payload for SKU-002..."
$payload2 = Generate-DynamicPayload -ProductId "SKU-002" -Delta 1 -TestName "concurrent-sku002"
if ($null -eq $payload2) {
    Write-Host "❌ Failed to generate payload for SKU-002" -ForegroundColor Red
    exit 1
}
$payload2 | Out-File -FilePath "temp_concurrent_sku002.json" -Encoding UTF8

Write-Host "Generating payload for SKU-003..."
$payload3 = Generate-DynamicPayload -ProductId "SKU-003" -Delta 1 -TestName "concurrent-sku003"
if ($null -eq $payload3) {
    Write-Host "❌ Failed to generate payload for SKU-003" -ForegroundColor Red
    exit 1
}
$payload3 | Out-File -FilePath "temp_concurrent_sku003.json" -Encoding UTF8

Write-Host "✅ All concurrent test payloads generated" -ForegroundColor Green
Write-Host "Starting concurrent tests on multiple products..."

# Start background jobs for concurrent testing
$job1 = Start-Job -ScriptBlock {
    param($ServerUrl, $ApiKey)
    & hey -n 50 -c 5 -m POST -H "X-API-Key: $ApiKey" -H "Content-Type: application/json" -D temp_concurrent_sku001.json "$ServerUrl/v1/inventory/updates"
} -ArgumentList $ServerUrl, $ApiKey

$job2 = Start-Job -ScriptBlock {
    param($ServerUrl, $ApiKey)
    & hey -n 50 -c 5 -m POST -H "X-API-Key: $ApiKey" -H "Content-Type: application/json" -D temp_concurrent_sku002.json "$ServerUrl/v1/inventory/updates"
} -ArgumentList $ServerUrl, $ApiKey

$job3 = Start-Job -ScriptBlock {
    param($ServerUrl, $ApiKey)
    & hey -n 50 -c 5 -m POST -H "X-API-Key: $ApiKey" -H "Content-Type: application/json" -D temp_concurrent_sku003.json "$ServerUrl/v1/inventory/updates"
} -ArgumentList $ServerUrl, $ApiKey

# Wait for all jobs to complete
Write-Host "Waiting for concurrent tests to complete..."
Wait-Job $job1, $job2, $job3 | Out-Null

# Get results
Write-Host "Results from concurrent SKU-001 test:"
Receive-Job $job1

Write-Host "Results from concurrent SKU-002 test:"
Receive-Job $job2

Write-Host "Results from concurrent SKU-003 test:"
Receive-Job $job3

# Clean up jobs
Remove-Job $job1, $job2, $job3

Write-Host ""
Write-Host "🧹 Cleaning up temporary files..." -ForegroundColor Yellow
Remove-Item temp_*.json -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "✅ Load testing completed!" -ForegroundColor Green
Write-Host ""
Write-Host "📊 What to look for in the results:" -ForegroundColor Cyan
Write-Host "1. Different products: High success rate (200 OK responses)"
Write-Host "2. Same product: Mix of 200 OK and 409 Conflict responses"
Write-Host "3. No 500 Internal Server Error responses"
Write-Host "4. Server logs should show worker distribution"
Write-Host "5. Response times should be reasonable (< 100ms for most requests)"
Write-Host ""
Write-Host "📝 Check server logs for:" -ForegroundColor Cyan
Write-Host "- Worker distribution across multiple workers"
Write-Host "- Product lock acquisition/release messages"
Write-Host "- Version conflict warnings"
Write-Host "- No deadlock or timeout errors"

Write-Host ""
Write-Host "🔍 Quick verification commands:" -ForegroundColor Yellow
Write-Host "Check current product states:"
Write-Host "  curl -H 'X-API-Key: demo' $ServerUrl/v1/inventory/SKU-001"
Write-Host "  curl -H 'X-API-Key: demo' $ServerUrl/v1/inventory/SKU-002"
Write-Host "  curl -H 'X-API-Key: demo' $ServerUrl/v1/inventory/SKU-003"
