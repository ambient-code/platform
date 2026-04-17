#!/usr/bin/env python3
"""Analyze categorized CodeRabbit comments to identify recurring patterns and produce metrics.

Usage:
    python3 analyze.py data/v0.2.0 metrics/
    python3 analyze.py data/all metrics/       # for backfill, produces metrics/all.json
"""

import json
import os
import sys
from collections import Counter, defaultdict
from difflib import SequenceMatcher
from pathlib import Path


EXISTING_GUARDRAILS = {
    "GetK8sClientsForRequest": [
        "GetK8sClientsForRequest",
        "service account",
        "user token auth",
    ],
    "No panic()": ["panic(", "panic in production"],
    "No any types": ["any type", "'any'", "no `any`"],
    "OwnerReferences": ["OwnerReference", "owner ref"],
    "No tokens in logs": [
        "token in log",
        "secret in log",
        "credential in log",
        "plaintext token",
        "log.*token",
        "token.*log",
    ],
    "Feature flags": ["feature flag", "unleash", "gate behind"],
    "Shadcn UI": ["shadcn", "raw HTML element", "raw <button", "raw <input"],
    "React Query": ["React Query", "manual fetch("],
    "Image reference consistency": [
        "image name mismatch",
        "image.*mismatch",
        "runner_image.*match",
        "image tag.*match",
    ],
    "Reconcile not create-or-skip": [
        "alreadyexists",
        "reconcile.*rbac",
        "reconcile existing",
        "create-and-ignore",
    ],
    "No silent error swallowing": [
        "silently swallowed",
        "partial failure.*silent",
        "error.*silently",
        "failure.*treated as success",
    ],
    "Namespace-scoped keys": [
        "namespaced key",
        "project-scoped key",
        "bare sessionid",
        "namespace.*prefix",
    ],
    "Restricted SecurityContext": [
        "restricted scc",
        "securitycontext",
        "runasnonroot",
        "drop all capabilities",
    ],
}


def load_comments(data_dir: Path) -> list[dict]:
    """Load categorized.json and return only non-filtered (Critical + Major) entries."""
    categorized_path = data_dir / "categorized.json"
    with open(categorized_path) as f:
        entries = json.load(f)
    return [e for e in entries if not e.get("filtered", True)]


def cluster_patterns(comments: list[dict]) -> list[dict]:
    """Group comments with similar titles using greedy clustering with SequenceMatcher."""
    clusters: list[list[dict]] = []

    for comment in comments:
        title = comment.get("title", "")
        placed = False
        for cluster in clusters:
            for member in cluster:
                ratio = SequenceMatcher(None, title, member.get("title", "")).ratio()
                if ratio > 0.5:
                    cluster.append(comment)
                    placed = True
                    break
            if placed:
                break
        if not placed:
            clusters.append([comment])

    patterns = []
    for cluster in clusters:
        titles = [c.get("title", "") for c in cluster]
        title_counts = Counter(titles)
        most_common_title = title_counts.most_common(1)[0][0]
        # Use shortest title as fallback if most common is empty
        if not most_common_title:
            most_common_title = min(titles, key=len) if titles else "Unknown"

        patterns.append(
            {
                "pattern_name": most_common_title,
                "comments": cluster,
            }
        )

    return patterns


def rank_patterns(patterns: list[dict]) -> list[dict]:
    """Compute impact metrics for each pattern cluster and sort by impact_score."""
    ranked = []
    for pattern in patterns:
        comments = pattern["comments"]
        critical_count = sum(1 for c in comments if c.get("severity") == "Critical")
        major_count = sum(1 for c in comments if c.get("severity") == "Major")
        components = sorted({c.get("component", "unknown") for c in comments})
        impact_score = critical_count * 4 + major_count * 3

        example_comments = [
            {
                "id": c.get("id"),
                "title": c.get("title"),
                "path": c.get("path"),
                "html_url": c.get("html_url"),
                "ai_prompt": c.get("ai_prompt"),
            }
            for c in comments[:3]
        ]

        ranked.append(
            {
                "pattern_name": pattern["pattern_name"],
                "count": len(comments),
                "critical_count": critical_count,
                "major_count": major_count,
                "impact_score": impact_score,
                "components": components,
                "example_comments": example_comments,
                "comments": comments,
            }
        )

    ranked.sort(key=lambda p: p["impact_score"], reverse=True)
    return ranked


def compute_component_breakdown(comments: list[dict]) -> dict:
    """Count total comments per component, and per component+severity."""
    breakdown: dict[str, dict[str, int]] = defaultdict(
        lambda: {"critical": 0, "major": 0, "total": 0}
    )
    for c in comments:
        component = c.get("component", "unknown")
        severity = c.get("severity", "")
        breakdown[component]["total"] += 1
        if severity == "Critical":
            breakdown[component]["critical"] += 1
        elif severity == "Major":
            breakdown[component]["major"] += 1
    return dict(breakdown)


def check_coverage_gaps(
    patterns: list[dict],
) -> tuple[list[dict], int]:
    """Check each pattern against known guardrails. Match on titles only to avoid false positives."""
    gap_count = 0
    for pattern in patterns:
        titles = " ".join(c.get("title", "").lower() for c in pattern["comments"])
        pattern_name = pattern.get("pattern_name", "").lower()
        combined_text = f"{pattern_name} {titles}"

        matched_guardrail = None
        for guardrail_name, keywords in EXISTING_GUARDRAILS.items():
            for keyword in keywords:
                if keyword.lower() in combined_text:
                    matched_guardrail = guardrail_name
                    break
            if matched_guardrail:
                break

        pattern["covered_by_guardrail"] = matched_guardrail
        if matched_guardrail is None:
            gap_count += 1

    return patterns, gap_count


def compute_pattern_categories(comments: list[dict]) -> dict[str, int]:
    """Count comments per pattern_category."""
    counts: Counter = Counter()
    for c in comments:
        cat = c.get("pattern_category", "uncategorized")
        counts[cat] += 1
    return dict(counts.most_common())


def determine_date(comments: list[dict]) -> str:
    """Use the latest created_at from comments, or today."""
    dates = [c.get("created_at", "") for c in comments if c.get("created_at")]
    if dates:
        # created_at is ISO 8601; lexicographic max gives latest
        latest = max(dates)
        # Extract date portion (YYYY-MM-DD)
        return latest[:10]
    from datetime import date

    return date.today().isoformat()


def build_metrics(
    release: str,
    comments: list[dict],
    ranked_patterns: list[dict],
    by_component: dict,
    gap_count: int,
    pattern_categories: dict[str, int],
) -> dict:
    """Assemble the metrics JSON structure."""
    prs = {c.get("pr_number") for c in comments if c.get("pr_number")}
    critical = sum(1 for c in comments if c.get("severity") == "Critical")
    major = sum(1 for c in comments if c.get("severity") == "Major")

    top_patterns = []
    for p in ranked_patterns:
        top_patterns.append(
            {
                "name": p["pattern_name"],
                "count": p["count"],
                "critical": p["critical_count"],
                "major": p["major_count"],
                "impact_score": p["impact_score"],
                "components": p["components"],
                "covered_by_guardrail": p["covered_by_guardrail"],
                "example_comments": p["example_comments"],
            }
        )

    return {
        "release": release,
        "date": determine_date(comments),
        "prs_analyzed": len(prs),
        "total_comments": len(comments),
        "critical": critical,
        "major": major,
        "by_component": by_component,
        "top_patterns": top_patterns,
        "coverage_gaps": gap_count,
        "pattern_categories": pattern_categories,
    }


def build_patterns_output(ranked_patterns: list[dict]) -> list[dict]:
    """Build the full patterns.json structure with all clustering details."""
    output = []
    for p in ranked_patterns:
        output.append(
            {
                "pattern_name": p["pattern_name"],
                "count": p["count"],
                "critical_count": p["critical_count"],
                "major_count": p["major_count"],
                "impact_score": p["impact_score"],
                "components": p["components"],
                "covered_by_guardrail": p["covered_by_guardrail"],
                "comments": [
                    {
                        "id": c.get("id"),
                        "pr_number": c.get("pr_number"),
                        "title": c.get("title"),
                        "severity": c.get("severity"),
                        "path": c.get("path"),
                        "line": c.get("line"),
                        "created_at": c.get("created_at"),
                        "html_url": c.get("html_url"),
                        "type": c.get("type"),
                        "component": c.get("component"),
                        "pattern_category": c.get("pattern_category"),
                        "ai_prompt": c.get("ai_prompt"),
                    }
                    for c in p["comments"]
                ],
            }
        )
    return output


def main():
    if len(sys.argv) != 3:
        print(f"Usage: {sys.argv[0]} <data_dir> <metrics_dir>", file=sys.stderr)
        sys.exit(1)

    data_dir = Path(sys.argv[1])
    metrics_dir = Path(sys.argv[2])

    if not data_dir.is_dir():
        print(f"Error: data directory not found: {data_dir}", file=sys.stderr)
        sys.exit(1)

    categorized_path = data_dir / "categorized.json"
    if not categorized_path.exists():
        print(f"Error: categorized.json not found in {data_dir}", file=sys.stderr)
        sys.exit(1)

    # Derive release name from directory name
    release = data_dir.name

    # Load and filter
    comments = load_comments(data_dir)
    if not comments:
        print("No non-filtered comments found. Nothing to analyze.")
        sys.exit(0)

    # 1. Pattern clustering
    patterns = cluster_patterns(comments)

    # 2. Impact ranking
    ranked = rank_patterns(patterns)

    # 3. Component breakdown
    by_component = compute_component_breakdown(comments)

    # 4. Coverage gap analysis
    ranked, gap_count = check_coverage_gaps(ranked)

    # 5. Pattern categories
    pattern_categories = compute_pattern_categories(comments)

    # 6. Build and write metrics JSON
    metrics = build_metrics(
        release, comments, ranked, by_component, gap_count, pattern_categories
    )
    os.makedirs(metrics_dir, exist_ok=True)
    metrics_path = metrics_dir / f"{release}.json"
    with open(metrics_path, "w") as f:
        json.dump(metrics, f, indent=2)
        f.write("\n")

    # 7. Write patterns.json to data directory
    patterns_output = build_patterns_output(ranked)
    patterns_path = data_dir / "patterns.json"
    with open(patterns_path, "w") as f:
        json.dump(patterns_output, f, indent=2)
        f.write("\n")

    print(
        f"Analyzed {len(comments)} comments → {len(ranked)} patterns "
        f"({gap_count} coverage gaps)"
    )
    print(f"  Metrics: {metrics_path}")
    print(f"  Patterns: {patterns_path}")


if __name__ == "__main__":
    main()
