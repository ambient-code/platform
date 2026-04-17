FROM registry.access.redhat.com/ubi9/go-toolset:1.25 AS builder

USER root

WORKDIR /workspace

COPY ambient-api-server/ ambient-api-server/
COPY ambient-sdk/go-sdk/ ambient-sdk/go-sdk/
COPY ambient-control-plane/go.mod ambient-control-plane/go.sum ambient-control-plane/

WORKDIR /workspace/ambient-control-plane
RUN go mod download

COPY ambient-control-plane/cmd/ cmd/
COPY ambient-control-plane/internal/ internal/

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o ambient-control-plane ./cmd/ambient-control-plane

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

WORKDIR /app

COPY --from=builder /workspace/ambient-control-plane/ambient-control-plane /usr/local/bin/

RUN chmod +x /usr/local/bin/ambient-control-plane

USER 1001

ENTRYPOINT ["/usr/local/bin/ambient-control-plane"]

LABEL name="ambient-control-plane" \
      vendor="Ambient" \
      version="0.0.1" \
      summary="Ambient Control Plane" \
      description="Kubernetes reconciler for the Ambient Code Platform"
