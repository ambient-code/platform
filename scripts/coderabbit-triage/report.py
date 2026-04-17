#!/usr/bin/env python3
"""Generate a markdown trend summary from per-release CodeRabbit triage metrics."""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path
from typing import Any


def parse_semver(version: str) -> tuple[int, int, int]:
    """Parse a semantic version string like 'v0.2.0' into a sortable tuple."""
    match = re.match(r"v?(\d+)\.(\d+)\.(\d+)", version)
    if not match:
        return (0, 0, 0)
    return (int(match.group(1)), int(match.group(2)), int(match.group(3)))


def load_metrics(metrics_dir: str) -> list[dict[str, Any]]:
    """Load all JSON metric files from the directory, skipping all.json."""
    metrics_path = Path(metrics_dir)
    if not metrics_path.is_dir():
        return []

    results: list[dict[str, Any]] = []
    for filepath in sorted(metrics_path.glob("*.json")):
        if filepath.name == "all.json":
            continue
        try:
            with open(filepath) as f:
                data = json.load(f)
            if "release" in data:
                results.append(data)
        except (json.JSONDecodeError, OSError):
            continue

    results.sort(key=lambda d: parse_semver(d.get("release", "")))
    return results


def format_delta(
    current: float, previous: float | None, lower_is_better: bool = True
) -> str:
    """Format a delta value with arrow indicators.

    lower_is_better=True means decreasing values get a down arrow (good),
    increasing values get an up arrow (concerning).
    """
    if previous is None:
        return "---"

    diff = current - previous

    if isinstance(diff, float):
        if abs(diff) < 0.05:
            return "0"
        sign = "+" if diff > 0 else ""
        formatted = f"{sign}{diff:.1f}"
    else:
        if diff == 0:
            return "0"
        sign = "+" if diff > 0 else ""
        formatted = f"{sign}{diff}"

    if diff > 0:
        arrow = " ↑" if not lower_is_better else " ↑"
    elif diff < 0:
        arrow = " ↓"
    else:
        arrow = ""

    return f"{formatted}{arrow}"


def issues_per_pr(data: dict[str, Any]) -> float:
    """Calculate issues per PR, handling missing or zero values."""
    prs = data.get("prs_analyzed", 0)
    total = data.get("total_comments", 0)
    if prs == 0:
        return 0.0
    return round(total / prs, 1)


def render_release_summary(
    release_data: dict[str, Any],
    previous_data: dict[str, Any] | None,
) -> str:
    """Render Section 1: Release Summary."""
    lines: list[str] = []
    version = release_data.get("release", "unknown")
    lines.append(f"## CodeRabbit Triage: {version}")
    lines.append("")
    lines.append("| Metric | Value | \u0394 vs Previous |")
    lines.append("|--------|-------|---------------|")

    prs = release_data.get("prs_analyzed", 0)
    critical = release_data.get("critical", 0)
    major = release_data.get("major", 0)
    per_pr = issues_per_pr(release_data)
    gaps = release_data.get("coverage_gaps", 0)

    prev_prs = previous_data.get("prs_analyzed") if previous_data else None
    prev_critical = previous_data.get("critical") if previous_data else None
    prev_major = previous_data.get("major") if previous_data else None
    prev_per_pr = issues_per_pr(previous_data) if previous_data else None
    prev_gaps = previous_data.get("coverage_gaps") if previous_data else None

    lines.append(
        f"| PRs analyzed | {prs} | {format_delta(prs, prev_prs, lower_is_better=False)} |"
    )
    lines.append(
        f"| Critical issues | {critical} | {format_delta(critical, prev_critical)} |"
    )
    lines.append(f"| Major issues | {major} | {format_delta(major, prev_major)} |")
    lines.append(f"| Issues per PR | {per_pr} | {format_delta(per_pr, prev_per_pr)} |")
    lines.append(f"| Coverage gaps | {gaps} | {format_delta(gaps, prev_gaps)} |")

    return "\n".join(lines)


def render_trend_table(all_releases: list[dict[str, Any]]) -> str:
    """Render Section 2: Trend Table."""
    lines: list[str] = []
    lines.append("### Trend")
    lines.append("")
    lines.append("| Release | Date | PRs | Critical | Major | Per PR | Gaps |")
    lines.append("|---------|------|-----|----------|-------|--------|------|")

    for data in all_releases:
        version = data.get("release", "?")
        date = data.get("date", "?")
        prs = data.get("prs_analyzed", 0)
        critical = data.get("critical", 0)
        major = data.get("major", 0)
        per_pr = issues_per_pr(data)
        gaps = data.get("coverage_gaps", 0)
        lines.append(
            f"| {version} | {date} | {prs} | {critical} | {major} | {per_pr} | {gaps} |"
        )

    return "\n".join(lines)


def render_uncovered_patterns(release_data: dict[str, Any]) -> str:
    """Render Section 3: Top Uncovered Patterns."""
    patterns = release_data.get("top_patterns", [])
    uncovered = [
        p for p in patterns if isinstance(p, dict) and not p.get("covered_by_guardrail")
    ]
    if not uncovered:
        return ""

    lines: list[str] = []
    lines.append("### Top Uncovered Patterns")
    lines.append("")

    for i, pattern in enumerate(uncovered[:10], start=1):
        name = pattern.get("name", "Unknown")
        count = pattern.get("count", 0)
        impact = pattern.get("impact_score", 0)
        components = pattern.get("components", [])
        comp_str = (
            ", ".join(components) if isinstance(components, list) else str(components)
        )
        suffix = f" \u2014 {comp_str}" if comp_str else ""
        lines.append(f"{i}. **{name}** ({count} occurrences, impact: {impact}){suffix}")

    return "\n".join(lines)


def render_recommended_guardrails(release_data: dict[str, Any]) -> str:
    """Render Section 4: Recommended Guardrails based on uncovered patterns."""
    patterns = release_data.get("top_patterns", [])
    uncovered = [
        p for p in patterns if isinstance(p, dict) and not p.get("covered_by_guardrail")
    ]
    if not uncovered:
        return ""

    top = uncovered[:10]

    lines: list[str] = []
    lines.append("### Recommended Guardrails")
    lines.append("")
    lines.append("#### CLAUDE.md Conventions")

    for pattern in top:
        name = pattern.get("name", "Unknown")
        lines.append(f"- **{name}**: Enforce via convention (needs specific rule)")

    lines.append("")
    lines.append("#### Hookify Rules")

    for pattern in top:
        name = pattern.get("name", "Unknown")
        components = pattern.get("components", [])
        lang = (
            "Go"
            if any(c in ("backend", "operator") for c in components)
            else "Python"
            if "runner" in components
            else "TypeScript"
        )
        lines.append(f"- PreToolUse hook for {name.lower()} enforcement in {lang} code")

    return "\n".join(lines)


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Generate markdown trend summary from CodeRabbit triage metrics."
    )
    parser.add_argument(
        "metrics_dir",
        help="Path to directory containing per-release metrics JSON files.",
    )
    parser.add_argument(
        "--release",
        default=None,
        help="Focus on a specific release version (e.g., v0.2.0).",
    )

    args = parser.parse_args()

    all_releases = load_metrics(args.metrics_dir)

    if not all_releases:
        print("No metrics files found")
        return

    # Determine the target release and its predecessor
    if args.release:
        target_idx = None
        for i, r in enumerate(all_releases):
            if r.get("release") == args.release:
                target_idx = i
                break
        if target_idx is None:
            print(f"Release {args.release} not found in metrics", file=sys.stderr)
            sys.exit(1)
        target_data = all_releases[target_idx]
        previous_data = all_releases[target_idx - 1] if target_idx > 0 else None
    else:
        target_data = all_releases[-1]
        previous_data = all_releases[-2] if len(all_releases) >= 2 else None

    # Section 1: Release Summary
    print(render_release_summary(target_data, previous_data))
    print()

    # Section 2: Trend Table
    print(render_trend_table(all_releases))
    print()

    # Section 3: Top Uncovered Patterns
    uncovered = render_uncovered_patterns(target_data)
    if uncovered:
        print(uncovered)
        print()

    # Section 4: Recommended Guardrails
    guardrails = render_recommended_guardrails(target_data)
    if guardrails:
        print(guardrails)


if __name__ == "__main__":
    main()
