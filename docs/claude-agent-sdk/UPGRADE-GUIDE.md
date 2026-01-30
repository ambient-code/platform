# Claude Agent SDK Upgrade Guide

**Status:** Approved
**Date:** 2026-01-27
**Author:** Platform Team

## Executive Summary

Upgrade from SDK 0.1.12 to 0.1.23 is **safe and recommended**.

- **Security:** No CVEs found
- **Breaking Changes:** None
- **Risk Level:** Very Low
- **Migration Effort:** Trivial (dependency update only)

## Version Comparison

### Current State

```toml
claude-agent-sdk = ">=0.1.12"
anthropic = { version = ">=0.68.0", extras = ["vertex"] }
```

### Target State

```toml
claude-agent-sdk = ">=0.1.23"
anthropic = { version = ">=0.76.0", extras = ["vertex"] }
```

### Version Gap

- **claude-agent-sdk:** 11 releases behind (54 days)
- **anthropic:** 8 releases behind (75 days)

## Security Assessment

### CVE Status

**No security vulnerabilities found** in either package.

Checked sources:

- National Vulnerability Database (NVD)
- Snyk Security Database
- GitHub Security Advisories
- Anthropic Security Updates

Related CVEs exist for **different products**:

- CVE-2025-54794/54795: Claude Code desktop application
- CVE-2025-49596: MCP Inspector tool
- CVE-2026-21852: Claude Code data leak (fixed in 2.0.65+)

None affect the Python SDK packages.

### Dependency Security

All transitive dependencies are clean:

- anyio: No vulnerabilities
- httpx: No vulnerabilities
- pydantic: No vulnerabilities
- typing-extensions: No vulnerabilities

## Breaking Changes Analysis

### claude-agent-sdk (0.1.12 → 0.1.23)

**No breaking changes** in this version range.

All changes are backward compatible:

- Feature additions (opt-in)
- Bug fixes
- Internal improvements

### anthropic (0.68.0 → 0.76.0)

One breaking change that **does not impact** the platform:

**v0.72.1:** Dropped Python 3.8 support, requires Python 3.9+

**Impact:** None (platform uses Python 3.10+)

## New Features

### claude-agent-sdk

**v0.1.15 - File Checkpointing**

Rollback file changes during sessions:

```python
options.enable_file_checkpointing = True
# Later:
client.rewind_files(user_message_id)
```

**Use Case:** Error recovery, exploratory workflows

**v0.1.17 - UserMessage UUID Field**

Direct access to message identifiers:

```python
message.uuid  # Access message ID for checkpointing
```

**v0.1.22 - Tool Use Results**

Enhanced tool result tracking:

```python
user_message.tool_use_result  # Direct access to tool results
```

**v0.1.23 - MCP Status Querying**

Monitor MCP server health:

```python
status = client.get_mcp_status()
# Returns: {server_name: {connected: bool, error: str}}
```

**Use Case:** Debugging MCP integrations, health checks

### anthropic

**v0.73.0 - Structured Outputs (Beta)**

Type-safe JSON responses:

```python
response = client.messages.create(
    model="claude-sonnet-4-5",
    schema=MyPydanticModel,
    beta=["structured-outputs-2024-12-13"]
)
```

**v0.75.0 - Claude Opus 4.5**

Support for latest flagship model:

```python
model="claude-opus-4-5-20251101"
```

**v0.76.0 - Server-Side Tools**

Tools execute on Anthropic infrastructure:

```python
# Reduces latency, improves security
```

## Bug Fixes Highlight

### claude-agent-sdk

- **v0.1.13:** Fixed concurrent subagent write conflicts
- **v0.1.13:** Faster error propagation (was 60s timeout)
- **v0.1.13:** Pydantic 2.12+ compatibility
- **v0.1.16:** Rate limit detection now working

### anthropic

- **v0.71.0:** Improved stream handling (can close without full consumption)
- **v0.72.0:** Better TypedDict type inference
- **v0.74.0:** Cross-platform file collection fixes
- **v0.75.0:** Auth header validation improvements

## Migration Steps

### Phase 1: Update Dependencies

Update `components/runners/claude-code-runner/pyproject.toml`:

```toml
[project]
dependencies = [
    "claude-agent-sdk>=0.1.23",
    "anthropic[vertex]>=0.76.0",
    # ... other deps unchanged
]
```

Rebuild:

```bash
cd components/runners/claude-code-runner
uv sync
```

### Phase 2: No Code Changes Required

The upgrade is **drop-in compatible**.
No changes to `adapter.py` or other integration code are needed.

### Phase 3: Verification

Run smoketest suite (see SMOKETEST.md):

```bash
cd components/runners/claude-code-runner
pytest tests/smoketest/
```

### Phase 4: Integration Testing

Test in development environment:

```bash
# Deploy to dev cluster
kubectl apply -k deploy/overlays/dev/

# Create test session
curl -X POST http://localhost:8080/api/projects/test/agentic-sessions \
  -H "Content-Type: application/json" \
  -d '{"prompt": "Hello, Claude!"}'

# Verify streaming works
# Verify tools execute
# Verify MCP servers connect
```

### Phase 5: Staging Deployment

Deploy to staging:

```bash
kubectl apply -k deploy/overlays/staging/
```

Monitor for 24-48 hours:

- Error rates (should be <0.1%)
- Latency (should be unchanged or better)
- Tool success rate (should be >99%)
- MCP connection stability

### Phase 6: Production Rollout

Rolling deployment to production:

```bash
kubectl set image deployment/claude-runner \
  runner=ghcr.io/your-org/claude-runner:v0.1.23
```

Monitor metrics:

- Session success rate
- Average response time
- Tool execution latency
- MCP health checks

## Rollback Plan

If issues arise, rollback is simple:

```bash
cd components/runners/claude-code-runner

# Revert pyproject.toml
git checkout HEAD~1 pyproject.toml

# Rebuild
uv sync

# Redeploy
kubectl set image deployment/claude-runner \
  runner=ghcr.io/your-org/claude-runner:v0.1.12
```

## Testing Checklist

### Unit Tests

- [ ] SDK client creation
- [ ] Message parsing (AssistantMessage, ToolUseBlock, etc.)
- [ ] Event conversion (SDK → AG-UI)
- [ ] Error handling (resume failures, interrupts)

### Integration Tests

- [ ] Simple query/response
- [ ] Tool execution (Read, Write, Bash)
- [ ] MCP server integration
- [ ] Streaming functionality
- [ ] Conversation continuation
- [ ] Interrupt handling

### Regression Tests

- [ ] Multi-repo sessions
- [ ] Workflow execution
- [ ] File upload handling
- [ ] Vertex AI authentication
- [ ] Langfuse observability
- [ ] Privacy masking

## Risk Assessment

### Very Low Risk

**Reasons:**

1. No breaking changes in upgrade path
2. 54 days of production usage on latest version
3. All changes are backward compatible
4. Extensive testing by Anthropic team
5. No CVEs or security concerns

### Mitigation Strategies

**Even though risk is low, we still prepare:**

1. **Gradual Rollout:** Dev → Staging → Production
2. **Monitoring:** Error rates, latency, success metrics
3. **Rollback Plan:** Simple dependency revert
4. **Communication:** Notify team before production deploy

## Recommended Timeline

### Week 1

- Review this guide with team
- Update dependencies in development
- Run smoketest suite
- Perform integration testing

### Week 2

- Deploy to staging environment
- Monitor for 48 hours
- Collect feedback from early users
- Address any issues found

### Week 3

- Deploy to production (rolling)
- Monitor metrics closely
- Communicate upgrade to users
- Document any learnings

## Success Criteria

The upgrade is successful when:

- [ ] All smoketests pass
- [ ] Error rate unchanged or lower
- [ ] Latency unchanged or better
- [ ] No user-reported issues
- [ ] MCP servers remain stable
- [ ] Observability data validates

## Optional Feature Adoption

After successful upgrade, consider enabling new features:

### File Checkpointing

Add to `adapter.py`:

```python
options.enable_file_checkpointing = True
```

Implement rewind UI in frontend for error recovery.

### MCP Health Monitoring

Add to observability:

```python
mcp_status = client.get_mcp_status()
for server, status in mcp_status.items():
    if not status['connected']:
        logger.warning(f"MCP server {server} disconnected: {status['error']}")
```

### Structured Outputs

For workflows requiring typed responses:

```python
response = client.messages.create(
    model=model,
    schema=WorkflowOutputSchema,
    beta=["structured-outputs-2024-12-13"]
)
```

## Documentation Updates

After upgrade, update:

- [ ] SDK-USAGE.md (version numbers)
- [ ] README.md (dependencies)
- [ ] ADR-0004 (technology stack)
- [ ] Deployment guides (container tags)

## Support Resources

- [Claude Agent SDK Releases](https://github.com/anthropics/claude-agent-sdk-python/releases)
- [Anthropic SDK Releases](https://github.com/anthropics/anthropic-sdk-python/releases)
- [Platform Slack Channel](#ambient-support)
- [On-Call Rotation](https://wiki/oncall)

## Approval

**Approved by:** Platform Team
**Date:** 2026-01-27
**Deployment Window:** Next sprint
