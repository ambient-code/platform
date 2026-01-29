# Claude Model Pricing

This document lists all Claude models supported by the Ambient Code platform with their complete pricing configuration for Langfuse observability tracking.

## Supported Models

### Claude Sonnet 4.5 (Latest)

**Model Name**: `claude-sonnet-4-5-20250929`

**Match Pattern**:
```regex
(?i)^(claude-sonnet-4-5-20250929|(eu\.|us\.)?anthropic\.claude-sonnet-4-5-20250929-v1:0|claude-sonnet-4-5-V1@20250929|claude-sonnet-4-5@20250929)$
```

**Tokenizer**: `claude`

**Unit**: `TOKENS`

**Pricing** (per 1M tokens):

| Token Type | Price | Notes |
|------------|-------|-------|
| Input | $3.00 | Regular input tokens |
| Output | $15.00 | Output tokens |
| Cache Creation | $3.75 | 25% premium over input |
| Cache Read | $0.30 | 90% discount from input |

**API Configuration**:
```json
{
  "inputPrice": 0.000003,
  "outputPrice": 0.000015,
  "inputCacheCreationPrice": 0.00000375,
  "inputCacheReadPrice": 0.0000003
}
```

**Supported Model IDs**:
- Anthropic API: `claude-sonnet-4-5-20250929`
- Vertex AI (US): `us.anthropic.claude-sonnet-4-5-20250929-v1:0`
- Vertex AI (EU): `eu.anthropic.claude-sonnet-4-5-20250929-v1:0`
- Vertex AI (Generic): `claude-sonnet-4-5@20250929`

---

### Claude Haiku 4.5 (Fast & Cost-Effective)

**Model Name**: `claude-haiku-4-5-20251001`

**Match Pattern**:
```regex
(?i)^(claude-haiku-4-5-20251001|(eu\.|us\.)?anthropic\.claude-haiku-4-5-20251001-v1:0|claude-4-5-haiku@20251001)$
```

**Tokenizer**: `claude`

**Unit**: `TOKENS`

**Pricing** (per 1M tokens):

| Token Type | Price | Notes |
|------------|-------|-------|
| Input | $1.00 | Regular input tokens |
| Output | $5.00 | Output tokens |
| Cache Creation | $1.25 | 25% premium over input |
| Cache Read | $0.10 | 90% discount from input |

**API Configuration**:
```json
{
  "inputPrice": 0.000001,
  "outputPrice": 0.000005,
  "inputCacheCreationPrice": 0.00000125,
  "inputCacheReadPrice": 0.0000001
}
```

**Supported Model IDs**:
- Anthropic API: `claude-haiku-4-5-20251001`
- Vertex AI (US): `us.anthropic.claude-haiku-4-5-20251001-v1:0`
- Vertex AI (EU): `eu.anthropic.claude-haiku-4-5-20251001-v1:0`
- Vertex AI (Generic): `claude-4-5-haiku@20251001`

---

### Claude Opus 4.1 (Most Capable)

**Model Name**: `claude-opus-4-1-20250805`

**Match Pattern**:
```regex
(?i)^(claude-opus-4-1-20250805|(eu\.|us\.)?anthropic\.claude-opus-4-1-20250805-v1:0|claude-opus-4-1@20250805)$
```

**Tokenizer**: `claude`

**Unit**: `TOKENS`

**Pricing** (per 1M tokens):

| Token Type | Price | Notes |
|------------|-------|-------|
| Input | $15.00 | Regular input tokens |
| Output | $75.00 | Output tokens |
| Cache Creation | $18.75 | 25% premium over input |
| Cache Read | $1.50 | 90% discount from input |

**API Configuration**:
```json
{
  "inputPrice": 0.000015,
  "outputPrice": 0.000075,
  "inputCacheCreationPrice": 0.00001875,
  "inputCacheReadPrice": 0.0000015
}
```

**Supported Model IDs**:
- Anthropic API: `claude-opus-4-1-20250805`
- Vertex AI (US): `us.anthropic.claude-opus-4-1-20250805-v1:0`
- Vertex AI (EU): `eu.anthropic.claude-opus-4-1-20250805-v1:0`
- Vertex AI (Generic): `claude-opus-4-1@20250805`

---

### Cost Optimization with Prompt Caching

All models support Anthropic's **Prompt Caching** feature, which can reduce costs by up to 90% for repeated content:

**Cache Creation** (First use):
- 25% premium over regular input tokens
- Useful for large context that will be reused

**Cache Read** (Subsequent uses):
- 90% discount from regular input tokens
- Applies when cached content is within 5 minute window

**Best Practices**:
1. Place static/repeated content at the beginning of prompts
2. Mark caching breakpoints in system prompts
3. Reuse sessions when possible to benefit from caching
4. Monitor cache hit rates in Langfuse

---

## Configuration in Langfuse

### Required Fields

When configuring models in Langfuse UI or API:

1. **Model Name**: Exact match (e.g., `claude-sonnet-4-5-20250929`)
2. **Match Pattern**: Regex to match all model ID variants
3. **Tokenizer ID**: Always `claude` for Claude models
4. **Unit**: Always `TOKENS`
5. **Input Price**: Base input token price
6. **Output Price**: Base output token price
7. **Input Cache Creation Price**: Cache creation price (25% premium)
8. **Input Cache Read Price**: Cache read price (90% discount)

---

## References

- [Anthropic Pricing](https://www.anthropic.com/pricing)
- [Anthropic Prompt Caching](https://docs.anthropic.com/en/docs/build-with-claude/prompt-caching)
- [Vertex AI Model Pricing](https://cloud.google.com/vertex-ai/generative-ai/pricing)
- [Langfuse Models API](https://langfuse.com/docs/model-based-usage)
