FROM registry.access.redhat.com/ubi10/ubi@sha256:f573194e8e5231f1c9340c497e1f8d9aa9dbb42b2849e60341e34f50eec9477e

ARG GIT_COMMIT=unknown

USER 0

# --- Pinned tool versions (bumped by runner-tool-versions workflow) ---
ARG GH_VERSION=2.74.0
ARG GLAB_VERSION=1.52.0
ARG UV_VERSION=0.7.8
ARG PRE_COMMIT_VERSION=4.2.0
ARG GEMINI_CLI_VERSION=0.1.17

# Install system packages: Python 3.12, git, jq, Node.js, Go, unzip (needed by CodeRabbit install script)
RUN dnf install -y python3 python3-pip python3-devel \
    git jq nodejs npm go-toolset unzip && \
    dnf clean all

# Install GitHub CLI and GitLab CLI (binary downloads, pinned)
RUN ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/') && \
    curl -fsSL "https://github.com/cli/cli/releases/download/v${GH_VERSION}/gh_${GH_VERSION}_linux_${ARCH}.tar.gz" \
    | tar -xz -C /usr/local/bin --strip-components=2 "gh_${GH_VERSION}_linux_${ARCH}/bin/gh" && \
    curl -fsSL "https://gitlab.com/gitlab-org/cli/-/releases/v${GLAB_VERSION}/downloads/glab_${GLAB_VERSION}_linux_${ARCH}.tar.gz" \
    | tar -xz -C /usr/local/bin --strip-components=1 bin/glab && \
    chmod +x /usr/local/bin/gh /usr/local/bin/glab

# Install uv (provides uvx for package execution) and pre-commit (for repo hooks)
RUN pip3 install --break-system-packages --no-cache-dir uv==${UV_VERSION} pre-commit==${PRE_COMMIT_VERSION}

# Create working directory
WORKDIR /app

# Copy ambient-runner package
COPY . /app/ambient-runner

# Install runner as a package, then remove build-only deps in same layer
RUN pip3 install --break-system-packages --no-cache-dir '/app/ambient-runner[all]' && \
    dnf remove -y python3-devel && \
    dnf clean all && \
    rm -rf /var/cache/dnf /var/lib/dnf

# Install Gemini CLI (npm package, Node.js already available)
RUN npm install -g @google/gemini-cli@${GEMINI_CLI_VERSION} && \
    npm cache clean --force

# Install CodeRabbit CLI (official install script, binary for current arch)
RUN curl -fsSL https://cli.coderabbit.ai/install.sh | CODERABBIT_INSTALL_DIR=/usr/local/bin sh

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
LABEL org.opencontainers.image.revision=$GIT_COMMIT

CMD ["/bin/bash", "-c", "umask 0022 && cd /app/ambient-runner && uvicorn main:app --host 0.0.0.0 --port 8001"]
