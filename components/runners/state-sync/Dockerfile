FROM alpine:3.21

ARG GIT_COMMIT=unknown

RUN apk add --no-cache rclone git jq bash sqlite

# Copy scripts
COPY hydrate.sh /usr/local/bin/hydrate.sh
COPY sync.sh /usr/local/bin/sync.sh

# Make scripts executable
RUN chmod +x /usr/local/bin/hydrate.sh /usr/local/bin/sync.sh

# Default to sync.sh (used by sidecar)
LABEL org.opencontainers.image.revision=$GIT_COMMIT

ENTRYPOINT ["/usr/local/bin/sync.sh"]
