FROM registry.access.redhat.com/ubi9/python-311@sha256:d0b35f779ca0ae87deaf17cd1923461904f52d3ef249a53dbd487e02bdabdde6

USER 0

# Add GitHub CLI repository and install packages
RUN dnf install -y 'dnf-command(config-manager)' && \
    dnf config-manager --add-repo https://cli.github.com/packages/rpm/gh-cli.repo && \
    dnf install -y gh --repo gh-cli && \
    dnf install -y git jq && \
    dnf clean all


# Install Node.js
# Use UBI AppStream to avoid conflicts with preinstalled nodejs-full-i18n
RUN dnf module reset -y nodejs && \
    dnf module enable -y nodejs:20 && \
    dnf install -y nodejs npm && \
    dnf clean all

# Install Go
# Use Go 1.21 from AppStream (latest LTS version in UBI9)
RUN dnf install -y go-toolset && \
    dnf clean all

# # Install Chromium for Playwright MCP (enables headless browser automation)
# # Required for self-development workflow where runner tests frontend changes locally
# # Using Playwright's bundled Chromium as it's self-contained and works on UBI9
# # Install system deps that Playwright needs on RHEL/UBI
# ENV PLAYWRIGHT_BROWSERS_PATH=/opt/app-root/src/.cache/ms-playwright
# RUN dnf install -y \
#     alsa-lib atk at-spi2-atk cups-libs libdrm libXcomposite libXdamage \
#     libXrandr mesa-libgbm pango nss nspr libxkbcommon && \
#     dnf clean all && \
#     mkdir -p "$PLAYWRIGHT_BROWSERS_PATH" && \
#     npx -y playwright install chromium && \
#     # Create symlink so 'chromium-browser' command works
#     ln -sf /opt/app-root/src/.cache/ms-playwright/chromium-*/chrome-linux/chrome /usr/local/bin/chromium-browser && \
#     # Make Playwright cache writable by non-root user (UID 1001) for lock files and temp dirs
#     chown -R 1001:0 /opt/app-root/src/.cache && \
#     chmod -R g=u /opt/app-root/src/.cache
# MCP ref to add
# "playwright": {
#     "type": "stdio",
#     "command": "npx",
#     "args": [
#       "@playwright/mcp@latest",
#       "--headless",
#       "--browser",
#       "chromium",
#       "--no-sandbox",
#       "--timeout-action",
#       "30000",
#       "--viewport-size",
#       "1280x720",
#       "--block-service-workers"
#     ],
#     "env": {
#       "PLAYWRIGHT_BROWSERS_PATH": "/opt/app-root/src/.cache/ms-playwright",
#       "PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD": "0"
#     }
#   }
# Install uv (provides uvx for package execution) and pre-commit (for repo hooks)
RUN pip install --no-cache-dir uv pre-commit

# Install Python 3.12 for Gerrit MCP server (requires >=3.12, runner uses 3.11)
RUN dnf install -y python3.12 python3.12-pip && \
    dnf clean all

# Install Gerrit MCP server (code review integration for Gerrit-based communities)
# Pinned to specific commit for reproducibility
# Uses python3.12 for the venv since gerrit-mcp-server requires Python >=3.12
RUN set -o pipefail && \
    git clone https://gerrit.googlesource.com/gerrit-mcp-server /opt/gerrit-mcp-server && \
    cd /opt/gerrit-mcp-server && \
    git checkout 5666642afe1a5217e2529225d4bd9c9df6310bd6 && \
    python3.12 -m venv .venv && \
    . .venv/bin/activate && \
    pip install --no-cache-dir --index-url https://pypi.org/simple -r uv-requirements.txt --require-hashes && \
    uv pip compile pyproject.toml --generate-hashes --output-file requirements.txt --extra dev --extra-index-url https://pypi.org/simple && \
    uv pip sync requirements.txt && \
    uv build && \
    WHEEL_FILE=$(find dist -maxdepth 1 -name '*.whl' -print -quit) && \
    WHEEL_HASH=$(sha256sum "${WHEEL_FILE}" | awk '{print $1}') && \
    echo "${WHEEL_FILE} --hash=sha256:${WHEEL_HASH}" > local-requirements.txt && \
    uv pip install -r local-requirements.txt --no-deps --require-hashes && \
    rm -f local-requirements.txt && \
    chown -R 1001:0 /opt/gerrit-mcp-server && \
    chmod -R g=u /opt/gerrit-mcp-server

# Create working directory
WORKDIR /app

# Copy ambient-runner package
COPY ambient-runner /app/ambient-runner

# Install runner as a package (pulls dependencies including AG-UI SDK)
RUN pip install --no-cache-dir '/app/ambient-runner[all]'

# Install Gemini CLI (npm package, Node.js already available)
RUN npm install -g @google/gemini-cli


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
