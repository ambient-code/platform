FROM registry.access.redhat.com/ubi9/python-311@sha256:d0b35f779ca0ae87deaf17cd1923461904f52d3ef249a53dbd487e02bdabdde6

# Build arguments for metadata
ARG GIT_COMMIT=unknown
ARG GIT_BRANCH=unknown
ARG GIT_REPO=unknown
ARG GIT_VERSION=unknown
ARG BUILD_DATE=unknown
ARG BUILD_USER=unknown

# Add labels to force cache invalidation and provide metadata
LABEL git.commit="${GIT_COMMIT}"
LABEL git.branch="${GIT_BRANCH}"
LABEL git.version="${GIT_VERSION}"
LABEL build.date="${BUILD_DATE}"
LABEL build.user="${BUILD_USER}"

USER 0

# Add GitHub CLI repository and install packages
RUN dnf install -y 'dnf-command(config-manager)' && \
    dnf config-manager --add-repo https://cli.github.com/packages/rpm/gh-cli.repo && \
    dnf install -y gh --repo gh-cli && \
    dnf install -y git jq && \
    dnf clean all

# Install uv (provides uvx for package execution)
RUN pip install --no-cache-dir uv

# Create working directory
WORKDIR /app

# Copy claude-runner package (no separate runner-shell needed)
COPY claude-code-runner /app/claude-runner


# Install runner as a package (pulls dependencies including AG-UI SDK)
RUN pip install --no-cache-dir /app/claude-runner

# Set environment variables
ENV PYTHONUNBUFFERED=1
ENV PYTHONDONTWRITEBYTECODE=1
ENV RUNNER_TYPE=claude
ENV HOME=/app
ENV SHELL=/bin/bash
ENV TERM=xterm-256color
ENV AGUI_PORT=8001

# Build metadata as environment variables
ENV GIT_COMMIT=${GIT_COMMIT}
ENV GIT_BRANCH=${GIT_BRANCH}
ENV GIT_REPO=${GIT_REPO}
ENV GIT_VERSION=${GIT_VERSION}
ENV BUILD_DATE=${BUILD_DATE}
ENV BUILD_USER=${BUILD_USER}

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
CMD ["/bin/bash", "-c", "umask 0022 && cd /app/claude-runner && uvicorn main:app --host 0.0.0.0 --port 8001"]
