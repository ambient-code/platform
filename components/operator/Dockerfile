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

# Build the application with embedded version info
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w \
    -X main.GitCommit=${GIT_COMMIT} \
    -X main.GitBranch=${GIT_BRANCH} \
    -X main.GitVersion=${GIT_VERSION} \
    -X main.BuildDate=${BUILD_DATE}" \
    -o operator .

# Final stage
FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

# Build arguments (need to redeclare for final stage)
ARG GIT_COMMIT=unknown
ARG GIT_BRANCH=unknown
ARG GIT_REPO=unknown
ARG GIT_VERSION=unknown
ARG BUILD_DATE=unknown
ARG BUILD_USER=unknown

# Add labels to force cache invalidation and provide metadata
LABEL git.commit="${GIT_COMMIT}"
LABEL git.branch="${GIT_BRANCH}"
LABEL git.version="${GIT_VERSION}"
LABEL build.date="${BUILD_DATE}"
LABEL build.user="${BUILD_USER}"

WORKDIR /app

RUN microdnf install -y procps && microdnf clean all

# Copy the binary from builder stage (binary has metadata embedded via ldflags)
COPY --from=builder /app/operator .

# Build metadata as environment variables (fallback, primary source is embedded in binary)
ENV GIT_COMMIT=${GIT_COMMIT}
ENV GIT_BRANCH=${GIT_BRANCH}
ENV GIT_REPO=${GIT_REPO}
ENV GIT_VERSION=${GIT_VERSION}
ENV BUILD_DATE=${BUILD_DATE}
ENV BUILD_USER=${BUILD_USER}

# Set executable permissions and make accessible to any user
RUN chmod +x ./operator && chmod 775 /app

USER 1001

# Command to run the executable
CMD ["./operator"]
