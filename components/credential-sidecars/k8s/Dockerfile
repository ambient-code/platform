FROM registry.access.redhat.com/ubi9/python-312:latest

USER 0
RUN pip install --no-cache-dir mcp-proxy kubernetes-mcp-server && \
    chown -R 1001:0 /opt/app-root
USER 1001

EXPOSE 8093

ENTRYPOINT ["mcp-proxy", "--sse-port", "8093", "--"]
CMD ["python", "-m", "kubernetes_mcp_server", "--kubeconfig", "/tmp/.ambient_kubeconfig", "--disable-multi-cluster"]
