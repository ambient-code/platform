FROM registry.access.redhat.com/ubi9/go-toolset:1.24 AS builder

USER 0
WORKDIR /build
COPY components/credential-sidecars/entrypoint/ ./entrypoint/
COPY components/ambient-mcp/ ./ambient-mcp/

WORKDIR /build/entrypoint
RUN go mod edit -replace github.com/ambient-code/platform/components/ambient-mcp=../ambient-mcp && \
    go mod tidy && \
    CGO_ENABLED=0 go build -o /credential-entrypoint .

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

USER 0
RUN microdnf install -y procps-ng shadow-utils && \
    microdnf clean all && \
    useradd -u 1001 -r -g 0 -s /sbin/nologin runner && \
    curl -fsSL https://github.com/containers/kubernetes-mcp-server/releases/download/v0.0.62/kubernetes-mcp-server-linux-amd64 \
      -o /usr/local/bin/kubernetes-mcp-server && \
    echo "4e323948bcf1c9b256140669ea56a53b31246fceec71e8cd533617624f4250e5  /usr/local/bin/kubernetes-mcp-server" | sha256sum -c - && \
    chmod +x /usr/local/bin/kubernetes-mcp-server

COPY --from=builder /credential-entrypoint /usr/local/bin/credential-entrypoint

ENV CREDENTIAL_PROVIDER=kubeconfig

USER 1001

EXPOSE 8093

ENTRYPOINT ["credential-entrypoint", "kubernetes-mcp-server"]
CMD ["--port", "8093", "--disable-multi-cluster", "--kubeconfig", "/tmp/.ambient_kubeconfig"]
