FROM registry.access.redhat.com/ubi9/ubi-minimal:9.2-750.1697534106

RUN \
    microdnf install -y \
    util-linux \
    && \
    microdnf clean all

COPY \
    ambient-api-server \
    /usr/local/bin/

EXPOSE 8000

ENTRYPOINT ["/usr/local/bin/ambient-api-server", "serve"]

LABEL name="ambient-api-server" \
      vendor="Ambient" \
      version="0.0.1" \
      summary="Ambient API Server" \
      description="REST API server for the Ambient Code Platform"
