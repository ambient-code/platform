# Build stage
FROM registry.access.redhat.com/ubi9/go-toolset:1.24 AS builder

# Build arguments for metadata
ARG GIT_COMMIT=unknown
ARG GIT_BRANCH=unknown
ARG GIT_REPO=unknown
ARG GIT_VERSION=unknown
ARG BUILD_DATE=unknown
ARG BUILD_USER=unknown

USER 0
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o operator .

# Final stage
FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

WORKDIR /app

RUN microdnf install -y procps && microdnf clean all

# Copy the binary from builder stage
COPY --from=builder /app/operator .

# Set executable permissions and make accessible to any user
RUN chmod +x ./operator && chmod 775 /app

USER 1001

# Use ENTRYPOINT so that args from K8s are appended, not replaced
ENTRYPOINT ["./operator"]
# Default args (can be overridden by K8s deployment)
CMD []
