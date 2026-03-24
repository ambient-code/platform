FROM registry.access.redhat.com/ubi9/go-toolset:1.24 AS builder

WORKDIR /build

USER 0

COPY ambient-api-server/ ./ambient-api-server/
COPY ambient-sdk/ ./ambient-sdk/
COPY ambient-control-plane/ ./ambient-control-plane/

WORKDIR /build/ambient-control-plane

RUN go mod download && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" \
    -o ambient-control-plane ./cmd/ambient-control-plane

FROM registry.access.redhat.com/ubi9/ubi-minimal:9.7

RUN microdnf install -y util-linux && microdnf clean all

WORKDIR /app

COPY --from=builder /build/ambient-control-plane/ambient-control-plane .

RUN chmod +x ./ambient-control-plane && chmod 775 /app

USER 1001

LABEL name="ambient-control-plane" \
      vendor="Ambient" \
      version="0.0.1" \
      summary="Ambient Control Plane" \
      description="Kubernetes reconciler for the Ambient Code Platform"

ENTRYPOINT ["./ambient-control-plane"]
