FROM alpine:3.19

# Install rclone, git, and utilities
RUN apk add --no-cache \
    rclone \
    git \
    bash \
    curl \
    jq \
    ca-certificates

# Copy scripts
COPY hydrate.sh /usr/local/bin/hydrate.sh
COPY sync.sh /usr/local/bin/sync.sh

# Make scripts executable
RUN chmod +x /usr/local/bin/hydrate.sh /usr/local/bin/sync.sh

# Default to sync.sh (used by sidecar)
ENTRYPOINT ["/usr/local/bin/sync.sh"]

