FROM registry.access.redhat.com/ubi9/python-312:latest

USER 0
RUN pip install --no-cache-dir mcp-proxy mcp-atlassian && \
    chown -R 1001:0 /opt/app-root
USER 1001

EXPOSE 8092

ENTRYPOINT ["mcp-proxy", "--sse-port", "8092", "--"]
CMD ["python", "-m", "mcp_atlassian"]
