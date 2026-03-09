"""
Pydantic data models and utilities for session-level metrics.

Provides tool classification, cost estimation, and structured metric models
that the ObservabilityManager uses to accumulate and emit session summary
scores to Langfuse.

Ported from the local Claude Code hooks (metrics_hook.py / models.py) and
adapted for in-memory use inside the Ambient runner.
"""

from __future__ import annotations

from datetime import datetime, timezone
from enum import Enum
from typing import Dict, Tuple

from pydantic import BaseModel, Field


# ── Tool classification ──────────────────────────────────────────────


class ToolCallType(str, Enum):
    """All known Claude Code tool types and MCP tool categories."""

    # Core file tools
    READ = "Read"
    WRITE = "Write"
    EDIT = "Edit"
    MULTI_EDIT = "MultiEdit"
    GLOB = "Glob"
    GREP = "Grep"
    NOTEBOOK_EDIT = "NotebookEdit"

    # Execution
    BASH = "Bash"

    # Web
    WEB_FETCH = "WebFetch"
    WEB_SEARCH = "WebSearch"

    # Task / Agent
    TASK = "Task"
    TASK_CREATE = "TaskCreate"
    TASK_UPDATE = "TaskUpdate"
    TASK_GET = "TaskGet"
    TASK_LIST = "TaskList"
    TASK_STOP = "TaskStop"
    TASK_OUTPUT = "TaskOutput"

    # Interaction
    ASK_USER_QUESTION = "AskUserQuestion"
    ENTER_PLAN_MODE = "EnterPlanMode"
    EXIT_PLAN_MODE = "ExitPlanMode"
    SKILL = "Skill"
    ENTER_WORKTREE = "EnterWorktree"

    # MCP tool categories
    MCP_GITHUB = "mcp_github"
    MCP_JIRA = "mcp_jira"
    MCP_CONFLUENCE = "mcp_confluence"
    MCP_ATLASSIAN = "mcp_atlassian"
    MCP_IDE = "mcp_ide"
    MCP_OTHER = "mcp_other"

    # Catch-all
    OTHER = "other"


# O(1) lookup table — built once at module load.
_TOOL_BY_VALUE: Dict[str, ToolCallType] = {m.value: m for m in ToolCallType}


def classify_tool(raw_name: str) -> ToolCallType:
    """Map a raw tool name to a ToolCallType.

    Direct matches use an O(1) dict lookup; MCP tools are mapped to their
    category (mcp_github, mcp_jira, etc.); anything else becomes OTHER.
    """
    t = _TOOL_BY_VALUE.get(raw_name)
    if t is not None:
        return t

    raw_lower = raw_name.lower()
    if raw_lower.startswith("mcp__") or raw_lower.startswith("mcp_"):
        if "github" in raw_lower:
            return ToolCallType.MCP_GITHUB
        if "jira" in raw_lower:
            return ToolCallType.MCP_JIRA
        if "confluence" in raw_lower:
            return ToolCallType.MCP_CONFLUENCE
        if "atlassian" in raw_lower:
            return ToolCallType.MCP_ATLASSIAN
        if "ide" in raw_lower:
            return ToolCallType.MCP_IDE
        return ToolCallType.MCP_OTHER

    return ToolCallType.OTHER


# ── Cost estimation ──────────────────────────────────────────────────

# Pricing per 1M tokens (USD): {model_prefix: (input, output, cache_read)}
MODEL_PRICING: Dict[str, Tuple[float, float, float]] = {
    "claude-opus-4": (15.0, 75.0, 1.50),
    "claude-sonnet-4": (3.0, 15.0, 0.30),
    "claude-sonnet-3": (3.0, 15.0, 0.30),
    "claude-haiku-3": (0.80, 4.0, 0.08),
    "claude-haiku-4": (0.80, 4.0, 0.08),
}


def _match_model_pricing(model: str) -> Tuple[float, float, float]:
    """Return (input, output, cache_read) pricing per 1M tokens for *model*."""
    model_lower = model.lower()
    for prefix, pricing in MODEL_PRICING.items():
        if prefix in model_lower:
            return pricing
    # Default to Sonnet pricing for unknown models
    return (3.0, 15.0, 0.30)


def estimate_cost(usage: dict, model: str) -> float:
    """Compute estimated USD cost from a token usage dict and model name.

    Args:
        usage: Dict with keys ``input_tokens``, ``output_tokens``,
               ``cache_creation_input_tokens``, ``cache_read_input_tokens``.
        model: Model name (e.g. ``"claude-sonnet-4-20250514"``).

    Returns:
        Estimated cost in USD.
    """
    input_price, output_price, cache_read_price = _match_model_pricing(model)

    inp = int(usage.get("input_tokens", 0))
    out = int(usage.get("output_tokens", 0))
    cache_create = int(usage.get("cache_creation_input_tokens", 0))
    cache_read = int(usage.get("cache_read_input_tokens", 0))

    cost = (inp / 1_000_000) * input_price
    cost += (out / 1_000_000) * output_price
    cost += (cache_create / 1_000_000) * input_price
    cost += (cache_read / 1_000_000) * cache_read_price
    return round(cost, 6)


# ── Clarification detection ─────────────────────────────────────────

_MAX_SHORT_RESPONSE = 500
_MAX_TRAILING_QUESTION = 200


def is_clarification_request(text: str) -> bool:
    """Detect if an assistant message is asking the user for clarification.

    Uses structural heuristics:
      - A short response (< 500 chars) ending with '?' is almost certainly
        Claude asking for input rather than delivering work.
      - A longer response whose last line ends with '?' and contains no code
        blocks is also likely a clarification.
      - Responses with code blocks are treated as work output even if they
        end with a question.
    """
    if not text or "?" not in text:
        return False

    stripped = text.strip()

    if "```" in stripped:
        return False

    if stripped.endswith("?") and len(stripped) < _MAX_SHORT_RESPONSE:
        return True

    last_line = stripped.rsplit("\n", 1)[-1].strip()
    if last_line.endswith("?") and len(last_line) < _MAX_TRAILING_QUESTION:
        return True

    return False


# ── Metric sub-models ────────────────────────────────────────────────


class TokenMetric(BaseModel):
    """Token usage and estimated cost for a session."""

    token_input: int = 0
    token_output: int = 0
    token_cache_creation: int = 0
    token_cache_read: int = 0
    token_total: int = 0
    estimated_cost_usd: float = 0.0
    models_seen: Dict[str, int] = Field(
        default_factory=dict,
        description="Model name -> number of API responses using that model",
    )


class InterruptMetric(BaseModel):
    """Counts of session interruptions by type."""

    interrupt_tool_failure_total: int = 0
    interrupt_tool_failure_count: Dict[str, int] = Field(
        default_factory=dict,
        description="ToolCallType value -> failure count",
    )
    interrupt_tool_reason_count: Dict[str, int] = Field(
        default_factory=dict,
        description="'ToolType: reason' -> failure count",
    )
    interrupt_unclear_context: int = 0
    interrupt_human: int = 0


class ToolsUsageMetric(BaseModel):
    """Tool call counts for the session."""

    tool_calls_total: int = 0
    tool_calls_group: Dict[str, int] = Field(
        default_factory=dict,
        description="ToolCallType value -> count",
    )


class SessionMetric(BaseModel):
    """Aggregate metrics for a single session."""

    session_id: str
    user_id: str
    timestamp: str = ""
    token_metrics: TokenMetric = Field(default_factory=TokenMetric)
    tools_usage_metric: ToolsUsageMetric = Field(default_factory=ToolsUsageMetric)
    interrupt_metric: InterruptMetric = Field(default_factory=InterruptMetric)

    def to_flat_scores(self) -> Dict[str, float]:
        """Flatten all metrics into ``{score_name: value}`` for Langfuse."""
        scores: Dict[str, float] = {}

        # Interrupts
        scores["interrupt_tool_failure_total"] = float(
            self.interrupt_metric.interrupt_tool_failure_total
        )
        for (
            tool_type,
            count,
        ) in self.interrupt_metric.interrupt_tool_failure_count.items():
            safe = tool_type.replace(" ", "_").replace(".", "_")
            scores[f"interrupt_tool_failure_{safe}"] = float(count)
        for reason, count in self.interrupt_metric.interrupt_tool_reason_count.items():
            safe = reason.replace(" ", "_").replace(".", "_")
            scores[f"interrupt_reason_{safe}"] = float(count)
        scores["interrupt_unclear_context"] = float(
            self.interrupt_metric.interrupt_unclear_context
        )
        scores["interrupt_human"] = float(self.interrupt_metric.interrupt_human)

        # Tool usage
        scores["tool_calls_total"] = float(self.tools_usage_metric.tool_calls_total)
        for tool_type, count in self.tools_usage_metric.tool_calls_group.items():
            safe = tool_type.replace(" ", "_").replace(".", "_")
            scores[f"tool_calls_{safe}"] = float(count)

        # Tokens
        scores["token_input"] = float(self.token_metrics.token_input)
        scores["token_output"] = float(self.token_metrics.token_output)
        scores["token_cache_creation"] = float(self.token_metrics.token_cache_creation)
        scores["token_cache_read"] = float(self.token_metrics.token_cache_read)
        scores["token_total"] = float(self.token_metrics.token_total)
        scores["estimated_cost_usd"] = self.token_metrics.estimated_cost_usd

        return scores

    @classmethod
    def build(
        cls,
        *,
        session_id: str,
        user_id: str,
        tool_calls: dict[str, int],
        tool_calls_total: int,
        tool_failures_total: int,
        tool_failure_counts: dict[str, int],
        tool_failure_reasons: dict[str, int],
        unclear_context: int,
        human_interrupts: int,
        accumulated_usage: dict[str, int],
        models_seen: dict[str, int],
        total_cost_usd: float,
    ) -> "SessionMetric":
        """Construct a SessionMetric from raw accumulated state."""
        return cls(
            session_id=session_id,
            user_id=user_id,
            timestamp=datetime.now(timezone.utc).isoformat(),
            token_metrics=TokenMetric(
                token_input=accumulated_usage.get("input_tokens", 0),
                token_output=accumulated_usage.get("output_tokens", 0),
                token_cache_creation=accumulated_usage.get(
                    "cache_creation_input_tokens", 0
                ),
                token_cache_read=accumulated_usage.get("cache_read_input_tokens", 0),
                token_total=sum(accumulated_usage.values()),
                estimated_cost_usd=round(total_cost_usd, 6),
                models_seen=dict(models_seen),
            ),
            tools_usage_metric=ToolsUsageMetric(
                tool_calls_total=tool_calls_total,
                tool_calls_group=dict(tool_calls),
            ),
            interrupt_metric=InterruptMetric(
                interrupt_tool_failure_total=tool_failures_total,
                interrupt_tool_failure_count=dict(tool_failure_counts),
                interrupt_tool_reason_count=dict(tool_failure_reasons),
                interrupt_unclear_context=unclear_context,
                interrupt_human=human_interrupts,
            ),
        )
