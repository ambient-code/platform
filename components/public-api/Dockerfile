# Build stage
FROM registry.access.redhat.com/ubi9/go-toolset:1.25 AS builder

WORKDIR /app

USER 0

# Copy go mod files first for caching
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o public-api .

# Runtime stage
FROM registry.access.redhat.com/ubi9/ubi-minimal:9.7

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/public-api .

# Set executable permissions
RUN chmod +x ./public-api && chmod 775 /app

USER 1001

# Expose port
EXPOSE 8081

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -sf http://localhost:8081/health || exit 1

# Run the binary
ENTRYPOINT ["./public-api"]
