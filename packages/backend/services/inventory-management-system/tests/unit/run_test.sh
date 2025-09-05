#!/bin/bash

# Simple test runner to debug test execution issues
echo "🧪 Simple Test Runner for Inventory Management System"
echo "====================================================="

# Change to the project root
cd "$(dirname "$0")/../.."
echo "📁 Working directory: $(pwd)"

# Check Go version
echo "🔍 Checking Go version..."
go version

# Check if go.mod exists
echo "📋 Checking go.mod..."
if [ -f "go.mod" ]; then
    echo "✅ go.mod found"
    head -5 go.mod
else
    echo "❌ go.mod not found"
    exit 1
fi

# Run go mod tidy
echo "🔧 Running go mod tidy..."
go mod tidy

# Test 1: Check if we can list test files
echo "📝 Listing test files..."
find ./tests/unit -name "*_test.go" -type f

# Test 2: Try to run a simple test
echo "🧪 Running models tests..."
go test -v ./tests/unit/models 2>&1

echo "🧪 Running cache tests..."
go test -v ./tests/unit/cache 2>&1

echo "🧪 Running testutils tests..."
go test -v ./tests/unit/testutils/... 2>&1

echo "✅ Test runner completed"
