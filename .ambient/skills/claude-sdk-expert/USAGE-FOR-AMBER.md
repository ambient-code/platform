# How Amber Should Use the SDK Expert System

**Audience:** Amber (the platform codebase agent)
**Purpose:** Guide for leveraging SDK expertise when working on the platform

## When to Activate SDK Expertise

You should reference the SDK expert skill and documentation when:

1. **User asks about SDK:** "Is it safe to upgrade?" "Are there CVEs?" "What features can we use?"
2. **Working on runner code:** Any changes to `components/runners/claude-code-runner/`
3. **Debugging SDK issues:** Streaming problems, tool execution, MCP servers
4. **Performance work:** Optimizing AG-UI protocol, reducing latency
5. **Feature development:** Adding SDK capabilities, new tools, MCP integrations

## Resource Hierarchy

When answering SDK questions, consult in this order:

### 1. Quick Reference (Start Here)

**File:** `.ambient/skills/claude-sdk-expert/SKILL.md`

**Use for:**
- Common patterns and code snippets
- Configuration reference
- Troubleshooting checklist
- Environment variables
- File locations

**Example Questions:**
- "How do I add a new MCP server?"
- "What configuration options are available?"
- "How do I debug streaming issues?"

### 2. Detailed Guides (Deep Dive)

**Location:** `docs/claude-agent-sdk/`

**Use for:**
- Understanding integration architecture
- Step-by-step procedures
- Migration planning
- Performance analysis

**Which Guide:**
- **SDK-USAGE.md:** How integration works, all the details
- **UPGRADE-GUIDE.md:** Version questions, security, migration
- **AG-UI-OPTIMIZATION.md:** Performance, scalability, bottlenecks
- **README.md:** Overview, links, quick tasks

### 3. Codebase (Implementation)

**Files:** `components/runners/claude-code-runner/adapter.py`, `observability.py`, `main.py`

**Use for:**
- Current implementation patterns
- Actual configuration in use
- Code examples from production

### 4. Tests (Validation)

**Location:** `tests/smoketest/`

**Use for:**
- Understanding expected behavior
- Validation before changes
- Test patterns for new features

## Common Question Patterns

### "Is it safe to upgrade the SDK?"

**Answer from:** `UPGRADE-GUIDE.md` section 1-2

**Key points:**
- Current version: 0.1.12
- Latest version: 0.1.23
- Security status: No CVEs
- Risk level: Very Low
- Breaking changes: None

**Response pattern:**
```
Yes, it's safe to upgrade from 0.1.12 to 0.1.23.

Security: No CVEs found in either version
Breaking changes: None (backward compatible)
Risk: Very Low (54 days production usage on latest)

See docs/claude-agent-sdk/UPGRADE-GUIDE.md for detailed instructions.
```

### "How do we use the SDK?"

**Answer from:** `SDK-USAGE.md` entire document

**Key sections:**
- Client lifecycle (section 2)
- Configuration options (section 3)
- Tool system (section 4)
- Authentication (section 5)

**Response pattern:**
```
The SDK is integrated in components/runners/claude-code-runner/adapter.py.

Key pattern:
1. Create ClaudeSDKClient with options
2. Connect to initialize subprocess
3. Query with user prompt
4. Stream response messages
5. Convert to AG-UI events
6. Disconnect when done

See docs/claude-agent-sdk/SDK-USAGE.md for complete details.
```

### "What SDK features can we use?"

**Answer from:** `SKILL.md` section "SDK Feature Availability"

**Currently used:**
- Client lifecycle, query/response
- Message types (Assistant, Tool, Result)
- MCP server integration
- Conversation continuation

**Available but unused:**
- File checkpointing (rollback changes)
- Hooks system (pre/post tool execution)
- Advanced permissions
- MCP status monitoring (SDK 0.1.23+)

**Response pattern:**
```
We currently use core SDK features but have several unused capabilities:

Ready to adopt:
- File checkpointing (error recovery)
- MCP status monitoring (health checks)
- Structured outputs (type-safe responses)

Requires exploration:
- Hooks system (custom workflows)
- Advanced permission controls

See .ambient/skills/claude-sdk-expert/SKILL.md for details.
```

### "How can we improve performance?"

**Answer from:** `AG-UI-OPTIMIZATION.md` sections 3, 7

**Top recommendations:**
1. Event batching (+30-40% throughput)
2. Compression (-30-50% bandwidth)
3. Metadata optimization (-5-8% bandwidth)

**Response pattern:**
```
Three high-impact optimizations available:

P0 - Event Batching (Low complexity):
- Buffer 10 events or 100ms timeout
- Expected: +30-40% throughput, -15% CPU

P1 - Compression (Medium complexity):
- Gzip events >1KB
- Expected: -30-50% bandwidth

Combined impact: Enables 3-5x more concurrent sessions

See docs/claude-agent-sdk/AG-UI-OPTIMIZATION.md for implementation.
```

### "Are there any CVEs?"

**Answer from:** `UPGRADE-GUIDE.md` section 2

**Key facts:**
- claude-agent-sdk 0.1.12: No CVEs
- anthropic 0.68.0: No CVEs
- Target versions (0.1.23, 0.76.0): No CVEs
- All dependencies: Clean

**Response pattern:**
```
No CVEs found in current or target SDK versions.

Checked sources:
- National Vulnerability Database
- Snyk Security Database
- GitHub Security Advisories

Related CVEs exist for Claude Code desktop app (different product).
SDK packages are clean.

See docs/claude-agent-sdk/UPGRADE-GUIDE.md section 2 for details.
```

## Code Task Patterns

### Upgrading SDK Version

**Process:**
1. Read `UPGRADE-GUIDE.md` sections 4-5
2. Update `pyproject.toml` dependencies
3. Run `uv sync`
4. Execute smoketests
5. Validate results

**Implementation:**
```bash
# Update pyproject.toml
claude-agent-sdk = ">=0.1.23"
anthropic = { version = ">=0.76.0", extras = ["vertex"] }

# Rebuild
cd components/runners/claude-code-runner
uv sync

# Validate
pytest tests/smoketest/ -v
```

### Debugging SDK Issues

**Process:**
1. Check `SKILL.md` troubleshooting checklist
2. Verify configuration (env vars, MCP)
3. Review logs (with secret redaction awareness)
4. Run targeted smoketests
5. Consult `SDK-USAGE.md` for patterns

**Common issues:**
- **Streaming:** Check `include_partial_messages=True`
- **Tools:** Verify in `allowed_tools` list
- **MCP:** Validate `.mcp.json` configuration
- **Resume:** Check `.claude/` directory exists

### Adding New Features

**Process:**
1. Check `SKILL.md` feature availability
2. Review `SDK-USAGE.md` for integration pattern
3. Implement following established patterns
4. Add smoketests for new functionality
5. Update documentation

**Example - File Checkpointing:**
```python
# In adapter.py _run_claude_agent_sdk()
options.enable_file_checkpointing = True

# Later, to rollback
if error_occurred:
    client.rewind_files(user_message_id)
```

## Response Quality Standards

### Always Provide

1. **Direct answer** to user's question
2. **File reference** for more details
3. **Code example** when applicable
4. **Next steps** for action

### Never

1. Invent information not in documentation
2. Claim features exist without verification
3. Suggest changes without explaining impact
4. Skip referencing source documents

## Example Interactions

### Good Response

**User:** "Can we upgrade to the latest SDK?"

**Amber:**
```
Yes, upgrading from 0.1.12 to 0.1.23 is safe and recommended.

Security: No CVEs in either version
Risk: Very Low (no breaking changes)
Effort: Trivial (dependency update only)

Steps:
1. Update pyproject.toml with new versions
2. Run uv sync
3. Execute smoketest suite
4. Deploy to staging for validation

See docs/claude-agent-sdk/UPGRADE-GUIDE.md for detailed migration plan.
```

### Bad Response

**User:** "Can we upgrade to the latest SDK?"

**Amber:**
```
I think we can upgrade but I'm not sure about the version or if it's safe.
You should probably check the documentation.
```

**Why bad:** No specific information, no file reference, not actionable

## Keeping Expertise Current

### When Documentation Changes

If you modify SDK integration:

1. Update relevant documentation files
2. Update SDK expert skill if patterns change
3. Add smoketests for new behavior
4. Note changes in response patterns

### When SDK Upgrades

After SDK upgrade:

1. Update version references in all docs
2. Note new features in SKILL.md
3. Update smoketest expectations if needed
4. Document migration in UPGRADE-GUIDE.md

## Quick Decision Tree

```
User mentions SDK?
├─ YES → Activate SDK expertise
│   │
│   ├─ Question about safety/security?
│   │   └─ Consult UPGRADE-GUIDE.md
│   │
│   ├─ Question about usage/integration?
│   │   └─ Consult SDK-USAGE.md
│   │
│   ├─ Question about performance?
│   │   └─ Consult AG-UI-OPTIMIZATION.md
│   │
│   └─ Question about debugging?
│       └─ Consult SKILL.md
│
└─ NO → Check if working on runner code
    │
    ├─ YES → Reference SKILL.md for patterns
    │
    └─ NO → Standard codebase work
```

## Files Quick Reference

```
📁 docs/claude-agent-sdk/
├── README.md              ← Start here (overview, links)
├── SDK-USAGE.md           ← How integration works
├── UPGRADE-GUIDE.md       ← Version, security, migration
└── AG-UI-OPTIMIZATION.md  ← Performance tuning

📁 .ambient/skills/claude-sdk-expert/
├── SKILL.md               ← Quick reference, patterns
└── USAGE-FOR-AMBER.md     ← This file

📁 components/runners/claude-code-runner/
├── adapter.py             ← SDK integration code
├── observability.py       ← Langfuse tracking
├── main.py                ← AG-UI server
└── tests/smoketest/       ← Validation tests
```

## Remember

You are now an expert on the platform's Claude Agent SDK integration. When users ask SDK questions:

1. Be confident (documentation is comprehensive)
2. Be specific (cite files and sections)
3. Be actionable (provide next steps)
4. Be accurate (don't invent, reference docs)

The documentation exists to make you effective at SDK work. Use it!
