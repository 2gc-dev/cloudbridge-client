.PHONY: all build clean test install build-all build-russian

BINARY_NAME=cloudbridge-client
VERSION=$(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-X main.Version=${VERSION}"

# Build targets
PLATFORMS=linux/amd64 windows/amd64 darwin/amd64
RUSSIAN_PLATFORMS=astra/amd64 alt/amd64 rosa/amd64 redos/amd64
BUILD_DIR=build

all: clean build

.PHONY: build
build:
	go build -o cloudbridge-client cmd/cloudbridge-client/main.go

.PHONY: build-mock
build-mock:
	go build -o mock_relay test/mock_relay/main.go

.PHONY: build-all
build-all: build build-mock

build-russian:
	@echo "Building for Russian platforms..."
	@mkdir -p ${BUILD_DIR}
	@for platform in ${RUSSIAN_PLATFORMS}; do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		output="${BUILD_DIR}/${BINARY_NAME}-$${os}-$${arch}"; \
		echo "Building for $${os}/$${arch}..."; \
		GOOS=linux GOARCH=$${arch} go build ${LDFLAGS} -o $${output} ./cmd/cloudbridge-client; \
	done

# Test targets
.PHONY: test
test:
	go test -v ./...

.PHONY: test-integration
test-integration:
	go test -v -tags=integration ./test/

.PHONY: test-unit
test-unit:
	go test -v -short ./...

.PHONY: test-coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out

.PHONY: test-benchmark
test-benchmark:
	go test -v -bench=. -benchmem ./test/

# Linting and code quality
.PHONY: lint
lint:
	golangci-lint run

.PHONY: lint-fix
lint-fix:
	golangci-lint run --fix

.PHONY: security-check
security-check:
	gosec ./...
	govulncheck ./...

.PHONY: format
format:
	go fmt ./...
	goimports -w .

# Documentation
.PHONY: docs
docs:
	@echo "Generating documentation..."
	@mkdir -p docs
	godoc -http=:6060 &
	@echo "Documentation available at http://localhost:6060"
	@echo "Press Ctrl+C to stop"

.PHONY: api-docs
api-docs:
	@echo "Generating API documentation..."
	swag init -g cmd/cloudbridge-client/main.go -o docs/swagger

# Development helpers
.PHONY: clean
clean:
	rm -f cloudbridge-client mock_relay
	rm -f coverage.out coverage.html
	go clean -cache

.PHONY: deps
deps:
	go mod download
	go mod tidy

.PHONY: mock-relay
mock-relay: build-mock
	./mock_relay 8084

.PHONY: run-client
run-client: build
	./cloudbridge-client --config config.yaml --token "test-token"

# Docker targets
.PHONY: docker-build
docker-build:
	docker build -t cloudbridge-client .

.PHONY: docker-test
docker-test:
	docker run --rm cloudbridge-client make test

# CI/CD targets
.PHONY: ci-test
ci-test: deps lint security-check test test-coverage

.PHONY: ci-build
ci-build: deps build-all

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build          - Build the main client"
	@echo "  build-mock     - Build the mock relay server"
	@echo "  build-all      - Build both client and mock relay"
	@echo "  test           - Run all tests"
	@echo "  test-integration - Run integration tests"
	@echo "  test-unit      - Run unit tests only"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  test-benchmark - Run benchmark tests"
	@echo "  lint           - Run linter"
	@echo "  lint-fix       - Run linter with auto-fix"
	@echo "  security-check - Run security checks"
	@echo "  format         - Format code"
	@echo "  docs           - Start documentation server"
	@echo "  api-docs       - Generate API documentation"
	@echo "  clean          - Clean build artifacts"
	@echo "  deps           - Download and tidy dependencies"
	@echo "  mock-relay     - Start mock relay server"
	@echo "  run-client     - Run the client"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-test    - Run tests in Docker"
	@echo "  ci-test        - Run CI test suite"
	@echo "  ci-build       - Run CI build" 