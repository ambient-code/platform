# Claude Agent SDK Documentation

Complete documentation for Claude Agent SDK integration in the Ambient platform.

## Documents

### SDK-REFERENCE.md
Complete developer reference for SDK integration.

**Use when:**
- Understanding how SDK works
- Configuring client options
- Adding tools or MCP servers
- Debugging issues
- Optimizing performance

**Contents:**
- Client lifecycle and patterns
- Configuration options
- Tool system (built-in + MCP)
- Message handling
- Debugging checklist
- Performance optimization
- Security considerations

### UPGRADE-GUIDE.md
Version upgrade planning and execution.

**Use when:**
- Considering SDK upgrade
- Checking security status (CVEs)
- Planning migration
- Validating compatibility

**Contents:**
- Version comparison (current vs latest)
- Security assessment
- Breaking changes analysis
- Migration steps
- Testing checklist
- Rollback plan

## Quick Start

### Check Current Version

```bash
cd components/runners/claude-code-runner
uv run python -c "import claude_agent_sdk; print(claude_agent_sdk.__version__)"
```

### Run Smoketests

```bash
# Quick validation (30 sec)
pytest tests/smoketest/ -v -m "not slow"

# Full validation (2-3 min)
pytest tests/smoketest/ -v
```

### Common Tasks

**Upgrade SDK:**
1. Review [UPGRADE-GUIDE.md](UPGRADE-GUIDE.md)
2. Update `pyproject.toml`
3. Run smoketests
4. Deploy to staging

**Debug SDK issue:**
1. Check [SDK-REFERENCE.md#debugging](SDK-REFERENCE.md#debugging)
2. Verify configuration
3. Run targeted smoketests
4. Review logs

**Add MCP server:**
1. See [SDK-REFERENCE.md#mcp-tools](SDK-REFERENCE.md#mcp-tools)
2. Update `.mcp.json`
3. Grant tool permissions
4. Test connectivity

**Optimize performance:**
1. Review [SDK-REFERENCE.md#performance-optimization](SDK-REFERENCE.md#performance-optimization)
2. Implement P0/P1 items
3. Benchmark improvements
4. Monitor metrics

## Current Status

**Version:** 0.1.12 (latest: 0.1.23)
**Security:** No CVEs found
**Upgrade:** Safe (no breaking changes)
**Performance:** 3-5x scalability available via optimization

## Key Files

```
components/runners/claude-code-runner/
├── adapter.py              # SDK integration
├── observability.py        # Langfuse tracking
├── main.py                 # AG-UI server
└── tests/smoketest/        # Validation tests
```

## Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| ANTHROPIC_API_KEY | Required | API authentication |
| CLAUDE_CODE_USE_VERTEX | false | Vertex AI mode |
| LLM_MODEL | claude-sonnet-4-5 | Model selection |

Full reference: [SDK-REFERENCE.md#environment-variables](SDK-REFERENCE.md#environment-variables)

## Support

**For SDK questions:** Consult `.ambient/skills/claude-sdk-expert/SKILL.md` (Amber references automatically)
**For upgrade questions:** See [UPGRADE-GUIDE.md](UPGRADE-GUIDE.md)
**For integration questions:** See [SDK-REFERENCE.md](SDK-REFERENCE.md)
