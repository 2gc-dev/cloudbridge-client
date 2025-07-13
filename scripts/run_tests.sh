#!/bin/bash

# CloudBridge Client Test Runner
# This script runs tests with proper environment setup

set -e

echo "🚀 Starting CloudBridge Client tests..."

# Set testing environment
export TESTING=true

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    print_error "go.mod not found. Please run this script from the project root."
    exit 1
fi

# Install dependencies
print_status "Installing dependencies..."
go mod download
go mod tidy

# Run linter
print_status "Running linter..."
if command -v golangci-lint >/dev/null 2>&1; then
    golangci-lint run --timeout=5m
else
    print_warning "golangci-lint not found, skipping linting"
fi

# Run unit tests
print_status "Running unit tests..."
go test -v -short -coverprofile=coverage.out ./...

# Build mock relay
print_status "Building mock relay..."
go build -o mock_relay ./test/mock_relay

# Check service availability
print_status "Checking service availability..."
if command -v nc >/dev/null 2>&1; then
    nc -z localhost 3456 && print_status "Main relay server (3456) is available" || print_warning "Main relay server (3456) is not available"
    nc -z localhost 8082 && print_status "Relay API (8082) is available" || print_warning "Relay API (8082) is not available"
    nc -z localhost 8080 && print_status "Keycloak (8080) is available" || print_warning "Keycloak (8080) is not available"
else
    print_warning "netcat not found, skipping service availability check"
fi

# Run integration tests
print_status "Running integration tests..."
# Start mock relay server
./mock_relay 8085 &
MOCK_RELAY_PID=$!
sleep 3

# Run integration tests
go test -v -tags=integration ./test/

# Stop mock relay
kill $MOCK_RELAY_PID 2>/dev/null || true

# Run benchmarks
print_status "Running benchmarks..."
go test -v -bench=. -benchmem ./pkg/...

# Generate coverage report
print_status "Generating coverage report..."
go tool cover -html=coverage.out -o coverage.html

print_status "✅ All tests completed successfully!"
print_status "📊 Coverage report generated: coverage.html" 