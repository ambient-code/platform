FROM quay.io/fedora/fedora:latest

RUN dnf install -y rclone git-core jq && \
    dnf clean all

# Copy scripts
COPY hydrate.sh /usr/local/bin/hydrate.sh
COPY sync.sh /usr/local/bin/sync.sh

# Make scripts executable
RUN chmod +x /usr/local/bin/hydrate.sh /usr/local/bin/sync.sh

# Default to sync.sh (used by sidecar)
ENTRYPOINT ["/usr/local/bin/sync.sh"]
