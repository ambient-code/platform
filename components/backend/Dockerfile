# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install git for fetching dependencies
RUN apk add --no-cache git

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Copy the source code
COPY . .

# Build the application
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o main .

# Runtime stage
FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

RUN microdnf install -y git && microdnf clean all

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Default agents directory
ENV AGENTS_DIR=/app/agents

# Set executable permissions and make accessible to any user
RUN chmod +x ./main && chmod 775 /app

USER 1001

# Expose port
EXPOSE 8080

# Command to run the executable
CMD ["./main"]
