FROM registry.access.redhat.com/ubi10/ubi@sha256:f573194e8e5231f1c9340c497e1f8d9aa9dbb42b2849e60341e34f50eec9477e

USER 0

# Install Python 3.12 (system default on RHEL 10), pip, and build deps
RUN dnf install -y python3 python3-pip python3-devel && \
    dnf clean all

# Install git/jq first, then GitHub CLI (gh RPM requires git)
RUN dnf install -y git jq 'dnf-command(config-manager)' && \
    dnf config-manager --add-repo https://cli.github.com/packages/rpm/gh-cli.repo && \
    dnf install -y gh --repo gh-cli && \
    dnf clean all

# Install Node.js 22 LTS
RUN dnf module enable -y nodejs:22 && \
    dnf install -y nodejs npm && \
    dnf clean all

# Install Go
RUN dnf install -y go-toolset && \
    dnf clean all

# Install uv (provides uvx for package execution) and pre-commit (for repo hooks)
RUN pip3 install --break-system-packages --no-cache-dir uv pre-commit

# Create working directory
WORKDIR /app

# Copy ambient-runner package
COPY ambient-runner /app/ambient-runner

# Install runner as a package (pulls dependencies including AG-UI SDK)
RUN pip3 install --break-system-packages --no-cache-dir '/app/ambient-runner[all]'

# Install Gemini CLI (npm package, Node.js already available)
RUN npm install -g @google/gemini-cli && \
    npm cache clean --force

# Remove build-only dependencies and clean caches
RUN dnf remove -y python3-devel && \
    dnf clean all && \
    rm -rf /var/cache/dnf /var/lib/dnf

# Set environment variables
ENV PYTHONUNBUFFERED=1
ENV PYTHONDONTWRITEBYTECODE=1
ENV RUNNER_TYPE=claude-agent-sdk
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

# Note: .claude directory is mounted as a volume (managed by operator)
# Permissions are handled via fsGroup in pod SecurityContext

# Run as UID 1001 to match content service (fixes permission issues)
USER 1001

# Expose AG-UI server port (8001 to avoid conflict with workspace-mcp OAuth on 8000)
EXPOSE 8001

# Start FastAPI AG-UI server using uvicorn
# The main module is installed as part of the package
CMD ["/bin/bash", "-c", "umask 0022 && cd /app/ambient-runner && uvicorn main:app --host 0.0.0.0 --port 8001"]
