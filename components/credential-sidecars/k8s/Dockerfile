FROM registry.access.redhat.com/ubi9/go-toolset:1.24 AS builder

USER 0
WORKDIR /build
COPY components/credential-sidecars/entrypoint/*.go ./entrypoint/
COPY components/ambient-mcp/ ./ambient-mcp/

WORKDIR /build/entrypoint
RUN go mod init github.com/ambient-code/platform/components/credential-sidecars/entrypoint && \
    go mod edit -require github.com/ambient-code/platform/components/ambient-mcp@v0.0.0 && \
    go mod edit -replace github.com/ambient-code/platform/components/ambient-mcp=../ambient-mcp && \
    go mod tidy && \
    CGO_ENABLED=0 go build -o /credential-entrypoint .

FROM registry.access.redhat.com/ubi9/python-312:latest

USER 0
RUN pip install --no-cache-dir mcp-proxy kubernetes-mcp-server && \
    chown -R 1001:0 /opt/app-root

COPY --from=builder /credential-entrypoint /opt/app-root/bin/credential-entrypoint

ENV CREDENTIAL_PROVIDER=kubeconfig

USER 1001

EXPOSE 8093

ENTRYPOINT ["credential-entrypoint", "mcp-proxy", "--sse-port", "8093", "--"]
CMD ["python", "-m", "kubernetes_mcp_server", "--kubeconfig", "/tmp/.ambient_kubeconfig", "--disable-multi-cluster"]
