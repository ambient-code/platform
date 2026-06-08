FROM ghcr.io/github/github-mcp-server:latest AS upstream

FROM registry.access.redhat.com/ubi9/go-toolset:1.24 AS builder

USER 0
WORKDIR /build
COPY credential-sidecars/entrypoint/ ./entrypoint/
COPY ambient-mcp/ ./ambient-mcp/

WORKDIR /build/entrypoint
RUN go mod edit -replace github.com/ambient-code/platform/components/ambient-mcp=../ambient-mcp && \
    go mod tidy && \
    CGO_ENABLED=0 go build -o /credential-entrypoint .

FROM registry.access.redhat.com/ubi9/python-312:latest

USER 0
RUN pip install --no-cache-dir mcp-proxy && \
    chown -R 1001:0 /opt/app-root

COPY --from=upstream /server/github-mcp-server /opt/app-root/bin/github-mcp-server
COPY --from=builder /credential-entrypoint /opt/app-root/bin/credential-entrypoint

ENV CREDENTIAL_PROVIDER=github

USER 1001

EXPOSE 8091

ENTRYPOINT ["credential-entrypoint", "mcp-proxy", "--pass-environment", "--port", "8091", "--"]
CMD ["github-mcp-server", "stdio"]
