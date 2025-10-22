#!/bin/bash

# QuantumSpring Test Suite
# This script runs all tests for the QuantumSpring implementation

set -e

echo "=========================================="
echo "QuantumSpring Test Suite"
echo "=========================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_info() {
    echo -e "${YELLOW}➜${NC} $1"
}

# Check if Go is installed
if ! command -v go &> /dev/null; then
    print_error "Go is not installed. Please install Go 1.24 or higher."
    exit 1
fi

print_success "Go is installed: $(go version)"
echo ""

# Step 1: Check if all required files exist
print_info "Step 1: Checking required files..."
files=(
    "internal/persistence/schema.sql"
    "internal/persistence/storage.go"
    "internal/persistence/sqlite.go"
    "internal/persistence/sqlite_test.go"
    "internal/persistence/init.go"
    "internal/usage/persistence_plugin.go"
    "internal/api/handlers/quantumspring/metrics.go"
    "internal/api/handlers/quantumspring/metrics_test.go"
    "internal/api/middleware/basicauth.go"
    "config.quantumspring.yaml"
)

all_files_exist=true
for file in "${files[@]}"; do
    if [ -f "$file" ]; then
        print_success "$file exists"
    else
        print_error "$file is missing!"
        all_files_exist=false
    fi
done

if [ "$all_files_exist" = false ]; then
    print_error "Some required files are missing. Aborting."
    exit 1
fi

echo ""
print_success "All required files exist"
echo ""

# Step 2: Download dependencies
print_info "Step 2: Downloading Go dependencies..."
if go mod download; then
    print_success "Dependencies downloaded"
else
    print_error "Failed to download dependencies"
    exit 1
fi
echo ""

# Step 3: Check if code compiles
print_info "Step 3: Checking if code compiles..."
if go build -o /tmp/cli-proxy-api-test ./cmd/server; then
    print_success "Code compiles successfully"
    rm -f /tmp/cli-proxy-api-test
else
    print_error "Code compilation failed"
    exit 1
fi
echo ""

# Step 4: Run unit tests for persistence layer
print_info "Step 4: Running persistence layer unit tests..."
echo "=========================================="
if go test -v ./internal/persistence/; then
    print_success "Persistence tests passed"
else
    print_error "Persistence tests failed"
    exit 1
fi
echo "=========================================="
echo ""

# Step 5: Run integration tests for QuantumSpring API
print_info "Step 5: Running QuantumSpring API integration tests..."
echo "=========================================="
if go test -v ./internal/api/handlers/quantumspring/; then
    print_success "QuantumSpring API tests passed"
else
    print_error "QuantumSpring API tests failed"
    exit 1
fi
echo "=========================================="
echo ""

# Step 6: Run all tests with coverage
print_info "Step 6: Running all tests with coverage report..."
if go test -cover ./internal/persistence/ ./internal/api/handlers/quantumspring/; then
    print_success "All tests passed with coverage"
else
    print_error "Some tests failed"
    exit 1
fi
echo ""

# Step 7: Generate detailed coverage report
print_info "Step 7: Generating detailed coverage report..."
go test -coverprofile=coverage.out ./internal/persistence/ ./internal/api/handlers/quantumspring/ > /dev/null 2>&1
if [ -f coverage.out ]; then
    coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
    print_success "Total coverage: $coverage"

    # Optionally open HTML coverage report
    read -p "Do you want to open HTML coverage report in browser? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        go tool cover -html=coverage.out
    fi
else
    print_error "Failed to generate coverage report"
fi
echo ""

# Final summary
echo "=========================================="
echo "Test Summary"
echo "=========================================="
print_success "All tests passed!"
print_success "Code compiles successfully"
print_success "Implementation is ready to use"
echo ""
echo "Next steps:"
echo "  1. Copy config: cp config.quantumspring.yaml config.yaml"
echo "  2. Build binary: go build -o cli-proxy-api ./cmd/server"
echo "  3. Run server: ./cli-proxy-api --config config.yaml"
echo "  4. Open dashboard: http://localhost:8317/_qs/metrics/ui"
echo ""
