FROM registry.access.redhat.com/ubi9/python-311@sha256:d0b35f779ca0ae87deaf17cd1923461904f52d3ef249a53dbd487e02bdabdde6

USER 0

# Add GitHub CLI repository and install packages
RUN dnf install -y 'dnf-command(config-manager)' && \
    dnf config-manager --add-repo https://cli.github.com/packages/rpm/gh-cli.repo && \
    dnf install -y gh --repo gh-cli && \
    dnf install -y git jq && \
    dnf clean all

# Create working directory
WORKDIR /app

# Copy and install runner-shell package (expects build context at components/runners)
COPY runner-shell /app/runner-shell
RUN cd /app/runner-shell && pip install --no-cache-dir .

# Copy and install mcp-ambient-server package
COPY mcp-ambient-server /app/mcp-ambient-server
RUN cd /app/mcp-ambient-server && pip install --no-cache-dir .

# Copy claude-runner specific files
COPY claude-code-runner /app/claude-runner

# Install runner wrapper as a package (pulls dependencies like claude-agent-sdk)
RUN pip install --no-cache-dir /app/claude-runner[observability] \
    && pip install --no-cache-dir aiofiles

# Set environment variables
ENV PYTHONUNBUFFERED=1
ENV PYTHONDONTWRITEBYTECODE=1
ENV RUNNER_TYPE=claude
ENV HOME=/app
ENV SHELL=/bin/bash
ENV TERM=xterm-256color

# Set umask to make files readable by other containers (fixes content service access)
# 0022 creates files as rw-r--r-- (644) instead of default rw------- (600)
RUN echo "umask 0022" >> /etc/profile && \
    echo "umask 0022" >> /root/.bashrc

# OpenShift compatibility
RUN chmod -R g=u /app && chmod -R g=u /usr/local && chmod g=u /etc/passwd

# Run as UID 1001 to match content service (fixes permission issues)
USER 1001

# Default command - run via runner-shell
CMD ["/bin/bash", "-c", "umask 0022 && python /app/claude-runner/wrapper.py"]
