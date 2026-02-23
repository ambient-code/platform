# Build stage
FROM registry.access.redhat.com/ubi9/go-toolset:1.24 AS builder

USER root

WORKDIR /workspace

# Copy Go modules and source
COPY go.mod go.sum ./
COPY cmd/ cmd/
COPY pkg/ pkg/
COPY plugins/ plugins/
COPY openapi/ openapi/

# Build the binary
RUN go build -ldflags="-s -w" -o ambient-api-server ./cmd/ambient-api-server

# Runtime stage
FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

RUN \
    microdnf install -y \
    util-linux \
    && \
    microdnf clean all

COPY --from=builder /workspace/ambient-api-server /usr/local/bin/

EXPOSE 8000

ENTRYPOINT ["/usr/local/bin/ambient-api-server", "serve"]

LABEL name="ambient-api-server" \
      vendor="Ambient" \
      version="0.0.1" \
      summary="Ambient API Server" \
      description="REST API server for the Ambient Code Platform"
