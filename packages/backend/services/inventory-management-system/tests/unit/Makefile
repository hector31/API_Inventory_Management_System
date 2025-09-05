# Makefile for Inventory Management System Unit Tests
# Provides convenient commands for running tests with various configurations

.PHONY: help test test-verbose test-race test-coverage test-bench test-all clean setup

# Default target
help: ## Show this help message
	@echo "Inventory Management System - Unit Test Commands"
	@echo "================================================"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "Examples:"
	@echo "  make test              # Run basic unit tests"
	@echo "  make test-coverage     # Run tests with coverage report"
	@echo "  make test-all          # Run comprehensive test suite"
	@echo "  make clean             # Clean test artifacts"

# Basic test commands
test: ## Run basic unit tests
	@echo "🧪 Running basic unit tests..."
	go test -v ./...

test-verbose: ## Run tests with verbose output
	@echo "🧪 Running tests with verbose output..."
	go test -v -count=1 ./...

test-race: ## Run tests with race detection
	@echo "🧪 Running tests with race detection..."
	go test -race -v ./...

test-short: ## Run only short tests (skip long-running tests)
	@echo "🧪 Running short tests..."
	go test -short -v ./...

# Coverage commands
test-coverage: ## Run tests with coverage report
	@echo "🧪 Running tests with coverage..."
	@mkdir -p ../../coverage
	go test -coverprofile=../../coverage/coverage.out -covermode=atomic ./...
	go tool cover -html=../../coverage/coverage.out -o ../../coverage/coverage.html
	@echo "📊 Coverage report generated: ../../coverage/coverage.html"
	go tool cover -func=../../coverage/coverage.out | tail -1

test-coverage-func: ## Show function-level coverage
	@echo "🧪 Generating function-level coverage..."
	@mkdir -p ../../coverage
	go test -coverprofile=../../coverage/coverage.out -covermode=atomic ./...
	go tool cover -func=../../coverage/coverage.out

# Benchmark commands
test-bench: ## Run benchmark tests
	@echo "🧪 Running benchmark tests..."
	@mkdir -p ../../test-reports
	go test -bench=. -benchmem -run=^$$ ./... | tee ../../test-reports/benchmarks.txt

test-bench-cpu: ## Run benchmarks with CPU profiling
	@echo "🧪 Running benchmarks with CPU profiling..."
	@mkdir -p ../../coverage
	go test -bench=. -cpuprofile=../../coverage/cpu.prof -run=^$$ ./...
	@echo "⚡ CPU profile saved to: ../../coverage/cpu.prof"

test-bench-mem: ## Run benchmarks with memory profiling
	@echo "🧪 Running benchmarks with memory profiling..."
	@mkdir -p ../../coverage
	go test -bench=. -memprofile=../../coverage/mem.prof -run=^$$ ./...
	@echo "🧠 Memory profile saved to: ../../coverage/mem.prof"

# Package-specific tests
test-handlers: ## Run handler tests only
	@echo "🧪 Running handler tests..."
	go test -v ./handlers/...

test-services: ## Run service tests only
	@echo "🧪 Running service tests..."
	go test -v ./services/...

test-cache: ## Run cache tests only
	@echo "🧪 Running cache tests..."
	go test -v ./cache/...

test-models: ## Run model tests only
	@echo "🧪 Running model tests..."
	go test -v ./models/...

test-testutils: ## Run test utility tests
	@echo "🧪 Running test utility tests..."
	go test -v ./testutils/...

# Comprehensive test suites
test-all: ## Run comprehensive test suite with all configurations
	@echo "🧪 Running comprehensive test suite..."
	@chmod +x run_tests.sh
	./run_tests.sh

test-ci: ## Run tests suitable for CI/CD (no interactive features)
	@echo "🧪 Running CI/CD test suite..."
	go test -v -race -coverprofile=../../coverage/coverage.out -covermode=atomic ./...
	go tool cover -func=../../coverage/coverage.out

# Quality and analysis
test-vet: ## Run go vet on test files
	@echo "🔍 Running go vet on tests..."
	go vet ./...

test-lint: ## Run golint on test files (requires golint)
	@echo "🔍 Running golint on tests..."
	@command -v golint >/dev/null 2>&1 || { echo "golint not installed. Run: go install golang.org/x/lint/golint@latest"; exit 1; }
	golint ./...

test-fmt: ## Check if test files are properly formatted
	@echo "🔍 Checking test file formatting..."
	@test -z "$$(gofmt -l .)" || (echo "Files need formatting:" && gofmt -l . && exit 1)

test-mod: ## Verify module dependencies
	@echo "🔍 Verifying module dependencies..."
	go mod verify
	go mod tidy

# Performance and profiling
profile-cpu: ## Analyze CPU profile (requires cpu.prof)
	@echo "⚡ Analyzing CPU profile..."
	@test -f ../../coverage/cpu.prof || { echo "CPU profile not found. Run 'make test-bench-cpu' first."; exit 1; }
	go tool pprof ../../coverage/cpu.prof

profile-mem: ## Analyze memory profile (requires mem.prof)
	@echo "🧠 Analyzing memory profile..."
	@test -f ../../coverage/mem.prof || { echo "Memory profile not found. Run 'make test-bench-mem' first."; exit 1; }
	go tool pprof ../../coverage/mem.prof

# Utility commands
clean: ## Clean test artifacts and reports
	@echo "🧹 Cleaning test artifacts..."
	rm -rf ../../coverage/
	rm -rf ../../test-reports/
	go clean -testcache

setup: ## Set up test environment and dependencies
	@echo "🔧 Setting up test environment..."
	go mod download
	go install golang.org/x/tools/cmd/cover@latest
	@mkdir -p ../../coverage
	@mkdir -p ../../test-reports
	@chmod +x run_tests.sh

# Development helpers
test-watch: ## Watch for changes and run tests (requires entr)
	@echo "👀 Watching for changes..."
	@command -v entr >/dev/null 2>&1 || { echo "entr not installed. Install with your package manager."; exit 1; }
	find . -name "*.go" | entr -c make test

test-debug: ## Run tests with debugging information
	@echo "🐛 Running tests with debugging..."
	go test -v -count=1 -timeout=30m ./...

test-timeout: ## Run tests with custom timeout
	@echo "⏰ Running tests with 5 minute timeout..."
	go test -v -timeout=5m ./...

# Documentation
test-docs: ## Generate test documentation
	@echo "📚 Generating test documentation..."
	@mkdir -p ../../test-reports
	go doc -all ./... > ../../test-reports/test_docs.txt
	@echo "📄 Test documentation saved to: ../../test-reports/test_docs.txt"

# Statistics
test-stats: ## Show test statistics
	@echo "📊 Test Statistics:"
	@echo "=================="
	@echo "Total test files: $$(find . -name "*_test.go" | wc -l)"
	@echo "Total test functions: $$(grep -r "^func Test" . | wc -l)"
	@echo "Total benchmark functions: $$(grep -r "^func Benchmark" . | wc -l)"
	@echo "Lines of test code: $$(find . -name "*_test.go" -exec wc -l {} + | tail -1)"
	@echo ""
	@echo "Test files by package:"
	@find . -name "*_test.go" | sed 's|/[^/]*$$||' | sort | uniq -c

# Quick commands for common workflows
quick: test-short ## Quick test run (alias for test-short)

full: test-all ## Full test suite (alias for test-all)

cover: test-coverage ## Coverage report (alias for test-coverage)

bench: test-bench ## Benchmark tests (alias for test-bench)

# Help for specific commands
help-coverage: ## Show coverage command help
	@echo "Coverage Commands:"
	@echo "=================="
	@echo "make test-coverage      - Generate HTML coverage report"
	@echo "make test-coverage-func - Show function-level coverage"
	@echo "make cover             - Alias for test-coverage"
	@echo ""
	@echo "Coverage files are saved to: ../../coverage/"

help-bench: ## Show benchmark command help
	@echo "Benchmark Commands:"
	@echo "==================="
	@echo "make test-bench      - Run all benchmarks"
	@echo "make test-bench-cpu  - Run benchmarks with CPU profiling"
	@echo "make test-bench-mem  - Run benchmarks with memory profiling"
	@echo "make profile-cpu     - Analyze CPU profile"
	@echo "make profile-mem     - Analyze memory profile"
	@echo ""
	@echo "Profile files are saved to: ../../coverage/"

help-packages: ## Show package-specific command help
	@echo "Package-Specific Commands:"
	@echo "=========================="
	@echo "make test-handlers  - Test handlers package only"
	@echo "make test-services  - Test services package only"
	@echo "make test-cache     - Test cache package only"
	@echo "make test-models    - Test models package only"
	@echo "make test-testutils - Test utilities package only"
