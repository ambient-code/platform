#!/usr/bin/env python3
"""Parse CodeRabbit review comments from raw JSON into categorized structured data.

Usage:
    python3 parse.py data/v0.2.0          # parse a single release
    python3 parse.py data/all             # parse backfill data

Reads raw-comments.json from the given directory, extracts structured fields
(type, severity, title, ai_prompt, component, pattern_category), and writes
categorized.json to the same directory.

Requires Python 3.10+ stdlib only.
"""

from __future__ import annotations

import json
import logging
import re
import sys
from pathlib import Path

logging.basicConfig(level=logging.WARNING, format="%(levelname)s: %(message)s")
logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Type / severity extraction
# ---------------------------------------------------------------------------

# The first line looks like:  _<emoji> Type_ | _<emoji> Severity_
# We capture the text after the emoji for both type and severity.
_TYPE_SEVERITY_RE = re.compile(
    r"^_[^\w]*(?P<type>[^_]+?)_\s*\|\s*_[^\w]*(?P<severity>[^_]+?)_",
    re.MULTILINE,
)

# Fallback: try to match type or severity individually
_TYPE_RE = re.compile(
    r"_[^\w]*(?P<type>Potential issue|Nitpick|Refactor suggestion)_", re.IGNORECASE
)
_SEVERITY_RE = re.compile(
    r"_[^\w]*(?P<severity>Critical|Major|Minor|Trivial)_", re.IGNORECASE
)

# ---------------------------------------------------------------------------
# Title extraction — first **bold** text after the type/severity line
# ---------------------------------------------------------------------------

_TITLE_RE = re.compile(r"\*\*(?P<title>[^*]+)\*\*")

# ---------------------------------------------------------------------------
# AI prompt extraction
# ---------------------------------------------------------------------------

_AI_PROMPT_RE = re.compile(
    r"<details>\s*<summary>\s*Prompt for AI Agents\s*</summary>\s*(?P<prompt>.*?)\s*</details>",
    re.DOTALL | re.IGNORECASE,
)

# ---------------------------------------------------------------------------
# Component derivation from file path
# ---------------------------------------------------------------------------

_COMPONENT_RULES: list[tuple[str, str]] = [
    ("components/backend/", "backend"),
    ("components/operator/", "operator"),
    ("components/frontend/", "frontend"),
    ("components/runners/", "runner"),
    ("components/ambient-cli/", "cli"),
    ("components/ambient-api-server/", "api-server"),
    ("components/ambient-sdk/", "sdk"),
    ("components/manifests/", "manifests"),
    ("components/public-api/", "public-api"),
    (".github/workflows/", "ci"),
    ("docs/", "docs"),
    ("scripts/", "scripts"),
]


def _derive_component(path: str | None) -> str:
    if not path:
        return "other"
    for prefix, component in _COMPONENT_RULES:
        if path.startswith(prefix):
            return component
    return "other"


# ---------------------------------------------------------------------------
# Pattern category classification via keyword matching
# ---------------------------------------------------------------------------

_PATTERN_KEYWORDS: list[tuple[str, list[str]]] = [
    (
        "security",
        [
            "token",
            "secret",
            "credential",
            "auth",
            "rbac",
            "permission",
            "injection",
            "leak",
            "xss",
        ],
    ),
    (
        "error_handling",
        ["error", "panic", "nil", "undefined", "catch", "exception", "fail", "recover"],
    ),
    (
        "k8s_resources",
        [
            "ownerreference",
            "resource limit",
            "cleanup",
            "namespace",
            "rbac",
            "securitycontext",
        ],
    ),
    (
        "concurrency",
        ["race", "concurrent", "mutex", "goroutine", "async", "deadlock", "channel"],
    ),
    (
        "validation",
        ["validate", "check", "missing", "undefined", "type", "schema", "boundary"],
    ),
    (
        "performance",
        ["o(n", "n+1", "cache", "pagination", "unbounded", "timeout", "memory", "leak"],
    ),
    ("idempotency", ["idempotent", "duplicate", "retry", "reconcile"]),
    (
        "api_design",
        ["endpoint", "route", "handler", "middleware", "response", "request"],
    ),
    ("testing", ["test", "mock", "coverage", "assertion"]),
]


def _classify_pattern(title: str, body: str) -> str:
    combined = f"{title} {body}".lower()
    for category, keywords in _PATTERN_KEYWORDS:
        for kw in keywords:
            if kw in combined:
                return category
    return "other"


# ---------------------------------------------------------------------------
# Core parsing
# ---------------------------------------------------------------------------


def _parse_comment(raw: dict) -> dict:
    """Parse a single raw comment into the categorized structure."""
    body = raw.get("body", "") or ""

    # Type and severity
    comment_type = "unknown"
    severity = "unknown"

    m = _TYPE_SEVERITY_RE.search(body)
    if m:
        comment_type = m.group("type").strip()
        severity = m.group("severity").strip()
    else:
        # Try individual patterns
        mt = _TYPE_RE.search(body)
        if mt:
            comment_type = mt.group("type").strip()
        ms = _SEVERITY_RE.search(body)
        if ms:
            severity = ms.group("severity").strip()

        if comment_type == "unknown" and severity == "unknown":
            logger.warning(
                "Comment %s (PR #%s) has no structured type/severity — tagging as unknown",
                raw.get("id"),
                raw.get("pr_number"),
            )

    # Title — first bold text after the type/severity line
    title = ""
    tm = _TITLE_RE.search(body)
    if tm:
        title = tm.group("title").strip()

    # AI prompt
    ai_prompt = ""
    pm = _AI_PROMPT_RE.search(body)
    if pm:
        ai_prompt = pm.group("prompt").strip()

    # Component
    component = _derive_component(raw.get("path"))

    # Pattern category
    pattern_category = _classify_pattern(title, body)

    # Filtered: keep only Critical or Major
    severity_lower = severity.lower()
    filtered = severity_lower not in ("critical", "major")

    return {
        "id": raw.get("id"),
        "pr_number": raw.get("pr_number"),
        "path": raw.get("path"),
        "line": raw.get("line"),
        "created_at": raw.get("created_at"),
        "html_url": raw.get("html_url"),
        "type": comment_type,
        "severity": severity,
        "title": title,
        "ai_prompt": ai_prompt,
        "component": component,
        "pattern_category": pattern_category,
        "filtered": filtered,
    }


def main() -> None:
    if len(sys.argv) != 2:
        print(f"Usage: {sys.argv[0]} <data-directory>", file=sys.stderr)
        sys.exit(1)

    data_dir = Path(sys.argv[1])
    input_file = data_dir / "raw-comments.json"

    if not input_file.exists():
        print(f"Error: {input_file} not found", file=sys.stderr)
        sys.exit(1)

    with open(input_file, encoding="utf-8") as f:
        raw_comments: list[dict] = json.load(f)

    categorized = [_parse_comment(c) for c in raw_comments]

    output_file = data_dir / "categorized.json"
    with open(output_file, "w", encoding="utf-8") as f:
        json.dump(categorized, f, indent=2, ensure_ascii=False)
        f.write("\n")

    # Summary
    critical = sum(1 for c in categorized if c["severity"].lower() == "critical")
    major = sum(1 for c in categorized if c["severity"].lower() == "major")
    filtered_out = sum(1 for c in categorized if c["filtered"])

    print(
        f"Parsed {len(categorized)} comments: {critical} Critical, {major} Major, {filtered_out} filtered out"
    )


if __name__ == "__main__":
    main()
