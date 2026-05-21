FROM registry.access.redhat.com/ubi9/python-312:latest

USER 0
RUN pip install --no-cache-dir mcp-proxy 'workspace-mcp==1.17.1' && \
    chown -R 1001:0 /opt/app-root
USER 1001

EXPOSE 8094

ENTRYPOINT ["mcp-proxy", "--sse-port", "8094", "--"]
CMD ["python", "-m", "workspace_mcp", "--permissions", "gmail:send", "drive:full"]
