FROM ghcr.io/github/github-mcp-server:latest AS upstream

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

COPY --from=upstream /github-mcp-server /usr/local/bin/github-mcp-server

RUN microdnf install -y procps-ng shadow-utils && \
    microdnf clean all && \
    useradd -u 1001 -r -g 0 -s /sbin/nologin runner

USER 1001

EXPOSE 8091

ENTRYPOINT ["github-mcp-server"]
CMD ["http", "--port", "8091"]
