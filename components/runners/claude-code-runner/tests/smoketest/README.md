# SDK Smoketest Suite

Validates Claude Agent SDK integration health before and after upgrades.

## Purpose

Catch regressions in SDK functionality:

- Client lifecycle (connect, disconnect)
- Message handling (streaming, parsing)
- Tool execution (Read, Write, Bash)
- State persistence (conversation continuation)
- Error handling (interrupts, failures)

## Quick Start

### Prerequisites

```bash
export ANTHROPIC_API_KEY=sk-ant-...
```

### Run All Tests

```bash
cd components/runners/claude-code-runner
pytest tests/smoketest/ -v
```

### Run Quick Tests Only

Skips slow integration tests:

```bash
pytest tests/smoketest/ -v -m "not slow"
```

### Run Specific Test Class

```bash
pytest tests/smoketest/test_sdk_integration.py::TestSDKClientLifecycle -v
```

## Test Organization

### TestSDKClientLifecycle

Basic client operations:

- Client creation
- Connection establishment
- Clean disconnection
- Multiple sequential clients

**Run time:** <5 seconds

### TestSDKMessageHandling

Message type parsing:

- AssistantMessage with TextBlock
- ResultMessage with usage data
- Streaming support

**Run time:** 10-20 seconds (marked slow)

### TestSDKToolExecution

Tool invocation:

- Read tool execution
- Write tool execution
- Tool result handling

**Run time:** 15-30 seconds (marked slow)

### TestSDKConfiguration

Configuration options:

- Permission modes
- Tool restrictions
- System prompt handling

**Run time:** 10-15 seconds

### TestSDKStatePersistence

State management:

- Conversation continuation
- Disk state persistence
- Multi-session memory

**Run time:** 20-30 seconds (marked slow)

### TestSDKErrorHandling

Error scenarios:

- Invalid inputs
- Interrupt handling
- Recovery mechanisms

**Run time:** 5-10 seconds

### TestSDKVersionCompatibility

Version checks:

- SDK version detection
- Required types available
- Client methods exist

**Run time:** <1 second

## Test Matrix

| Test | Live API | Workspace | Slow |
|------|----------|-----------|------|
| Client creation | Yes | Yes | No |
| Connect/disconnect | Yes | Yes | No |
| Simple query | Yes | Yes | Yes |
| Text parsing | Yes | Yes | Yes |
| Read tool | Yes | Yes | Yes |
| Write tool | Yes | Yes | Yes |
| Permission mode | Yes | Yes | No |
| Tool restrictions | Yes | Yes | No |
| Continuation | Yes | Yes | Yes |
| Interrupts | Yes | Yes | Yes |
| Version check | No | No | No |

## Usage Scenarios

### Before SDK Upgrade

Run full suite to establish baseline:

```bash
pytest tests/smoketest/ -v --tb=short > smoketest-before.log
```

Review results:

```bash
grep -E "(PASSED|FAILED|ERROR)" smoketest-before.log
```

### After SDK Upgrade

Update dependencies:

```bash
uv sync
```

Run full suite again:

```bash
pytest tests/smoketest/ -v --tb=short > smoketest-after.log
```

Compare results:

```bash
diff smoketest-before.log smoketest-after.log
```

### Continuous Integration

Add to CI pipeline:

```yaml
# .github/workflows/test.yml
- name: SDK Smoketest
  env:
    ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
  run: |
    cd components/runners/claude-code-runner
    pytest tests/smoketest/ -v -m "not slow"
```

### Pre-Deployment Validation

Run before production deploy:

```bash
# Quick validation (30 seconds)
pytest tests/smoketest/ -v -m "not slow"

# Full validation (2-3 minutes)
pytest tests/smoketest/ -v
```

## Expected Results

### Successful Run

```
tests/smoketest/test_sdk_integration.py::TestSDKClientLifecycle::test_client_creation PASSED
tests/smoketest/test_sdk_integration.py::TestSDKClientLifecycle::test_client_connect_disconnect PASSED
tests/smoketest/test_sdk_integration.py::TestSDKMessageHandling::test_simple_query_response PASSED
...

======================== 15 passed in 45.23s ========================
```

### Version Compatibility Pass

All tests should pass when upgrading within compatible versions (for example, 0.1.12 → 0.1.23).

### Breaking Change Detection

Tests will fail if SDK introduces breaking changes:

- Message type changes
- Client API changes
- Tool interface changes

## Troubleshooting

### Tests Skipped

```
SKIPPED [15] ANTHROPIC_API_KEY not set
```

**Solution:** Set API key:

```bash
export ANTHROPIC_API_KEY=sk-ant-...
```

### Connection Timeouts

```
ERROR: TimeoutError during client.connect()
```

**Possible causes:**

- Network issues
- API service down
- Rate limiting

**Solution:** Wait and retry, check Anthropic status page.

### Tool Execution Failures

```
FAILED: Tool 'Write' not available
```

**Possible causes:**

- SDK version incompatibility
- Tool restrictions changed
- Permission mode issues

**Solution:** Check SDK changelog, verify configuration.

### State Persistence Failures

```
FAILED: Could not continue conversation
```

**Possible causes:**

- `.claude/` directory not writable
- Disk state format changed
- Session ID mismatch

**Solution:** Check workspace permissions, review SDK migration notes.

## Adding New Tests

Follow this pattern:

```python
class TestNewFeature:
    """Test description."""

    @pytest.mark.skipif(not ANTHROPIC_API_KEY, reason=SKIP_REASON)
    async def test_feature_works(self, sdk_options):
        """Feature behaves correctly."""
        client = ClaudeSDKClient(options=sdk_options)
        await client.connect()

        try:
            # Test logic here
            await client.query("test prompt")

            async for message in client.receive_response():
                # Validate message
                pass

        finally:
            await client.disconnect()
```

Mark slow tests:

```python
@pytest.mark.slow
async def test_expensive_operation(self, sdk_options):
    """This test takes >5 seconds."""
    # ...
```

## Integration with Platform

These tests validate SDK behavior in isolation.
For full platform integration tests, see:

- `tests/integration/test_adapter.py` - Adapter integration
- `tests/integration/test_ag_ui.py` - AG-UI protocol
- `tests/integration/test_observability.py` - Langfuse tracking

## References

- [Claude Agent SDK Documentation](https://github.com/anthropics/claude-agent-sdk-python)
- [pytest Documentation](https://docs.pytest.org/)
- [Platform Testing Guide](../../docs/TESTING.md)
