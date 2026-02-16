FROM registry.access.redhat.com/ubi9/ubi-minimal:9.2-750.1697534106

RUN \
    microdnf install -y \
    util-linux \
    && \
    microdnf clean all

COPY \
    ambient-control-plane \
    /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/ambient-control-plane"]

LABEL name="ambient-control-plane" \
      vendor="Ambient" \
      version="0.0.1" \
      summary="Ambient Control Plane" \
      description="Kubernetes reconciler for the Ambient Code Platform"
