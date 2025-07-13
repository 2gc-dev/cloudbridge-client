# Multi-stage build for CloudBridge Client
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X main.Version=$(git describe --tags --always --dirty)" \
    -o cloudbridge-client \
    ./cmd/cloudbridge-client

# Build mock relay for testing
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o mock_relay \
    ./test/mock_relay

# Final stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S cloudbridge && \
    adduser -u 1001 -S cloudbridge -G cloudbridge

# Create necessary directories
RUN mkdir -p /etc/cloudbridge-client /var/log/cloudbridge-client /app && \
    chown -R cloudbridge:cloudbridge /etc/cloudbridge-client /var/log/cloudbridge-client /app

# Copy binaries from builder stage
COPY --from=builder --chown=cloudbridge:cloudbridge /app/cloudbridge-client /app/
COPY --from=builder --chown=cloudbridge:cloudbridge /app/mock_relay /app/

# Copy configuration files
COPY --chown=cloudbridge:cloudbridge config/config.yaml /etc/cloudbridge-client/
COPY --chown=cloudbridge:cloudbridge deploy/cloudbridge-client.service /etc/systemd/system/

# Set working directory
WORKDIR /app

# Switch to non-root user
USER cloudbridge

# Expose ports
EXPOSE 9090 3389

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9090/health || exit 1

# Default command
ENTRYPOINT ["./cloudbridge-client"]

# Default arguments
CMD ["--config", "/etc/cloudbridge-client/config.yaml"] 