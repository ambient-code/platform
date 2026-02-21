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

# Build arguments for metadata (placed after COPY to avoid busting cache)
ARG GIT_COMMIT=unknown
ARG GIT_BRANCH=unknown
ARG GIT_REPO=unknown
ARG GIT_VERSION=unknown
ARG BUILD_DATE=unknown
ARG BUILD_USER=unknown

# Build the application
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o operator .

# Runtime stage
FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/operator .

# Set executable permissions and make accessible to any user
RUN chmod +x ./operator && chmod 775 /app

USER 1001

# Use ENTRYPOINT so that args from K8s are appended, not replaced
ENTRYPOINT ["./operator"]
# Default args (can be overridden by K8s deployment)
CMD []
