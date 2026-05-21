FROM ghcr.io/github/github-mcp-server:latest AS upstream

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

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

COPY --from=upstream /server/github-mcp-server /usr/local/bin/github-mcp-server
COPY --from=builder /credential-entrypoint /usr/local/bin/credential-entrypoint

RUN microdnf install -y procps-ng shadow-utils && \
    microdnf clean all && \
    useradd -u 1001 -r -g 0 -s /sbin/nologin runner

ENV CREDENTIAL_PROVIDER=github

USER 1001

EXPOSE 8091

ENTRYPOINT ["credential-entrypoint", "github-mcp-server"]
CMD ["http", "--port", "8091"]
