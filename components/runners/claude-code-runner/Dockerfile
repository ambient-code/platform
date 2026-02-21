# Builder stage - install Python dependencies
FROM registry.access.redhat.com/ubi9/python-312 AS builder

USER 0
WORKDIR /build

COPY claude-code-runner /build/claude-runner

RUN pip install --no-cache-dir --prefix=/install '/build/claude-runner[all]' uv

# Runtime stage
FROM registry.access.redhat.com/ubi9/python-312

USER 0

# Install all system dependencies in a single layer to minimize image size
RUN dnf install -y 'dnf-command(config-manager)' && \
    dnf config-manager --add-repo https://cli.github.com/packages/rpm/gh-cli.repo && \
    dnf install -y gh --repo gh-cli git && \
    dnf module reset -y nodejs && \
    dnf module enable -y nodejs:20 && \
    dnf install -y nodejs npm && \
    dnf clean all && \
    rm -rf /var/cache/dnf && \
    curl -Lo /usr/local/bin/jq https://github.com/jqlang/jq/releases/download/jq-1.7.1/jq-linux-$(uname -m | sed 's/aarch64/arm64/;s/x86_64/amd64/') && \
    chmod +x /usr/local/bin/jq

# Copy installed Python packages from builder
COPY --from=builder /install /usr/local

WORKDIR /app

# Copy claude-runner package
COPY claude-code-runner /app/claude-runner

# Set environment variables
ENV PYTHONUNBUFFERED=1
ENV PYTHONDONTWRITEBYTECODE=1
ENV RUNNER_TYPE=claude
ENV HOME=/app
ENV SHELL=/bin/bash
ENV TERM=xterm-256color
ENV AGUI_PORT=8001

# Set umask to make files readable by other containers (fixes content service access)
# 0022 creates files as rw-r--r-- (644) instead of default rw------- (600)
RUN echo "umask 0022" >> /etc/profile && \
    echo "umask 0022" >> /root/.bashrc

# OpenShift compatibility
RUN chmod -R g=u /app && chmod -R g=u /usr/local && chmod g=u /etc/passwd

# Run as UID 1001 to match content service (fixes permission issues)
USER 1001

# Expose AG-UI server port (8001 to avoid conflict with workspace-mcp OAuth on 8000)
EXPOSE 8001

# Start FastAPI AG-UI server using uvicorn
CMD ["/bin/bash", "-c", "umask 0022 && cd /app/claude-runner && uvicorn main:app --host 0.0.0.0 --port 8001"]
