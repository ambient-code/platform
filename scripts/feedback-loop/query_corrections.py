#!/usr/bin/env python3
"""
Feedback Loop: Query Langfuse corrections and create improvement sessions.

Queries Langfuse for ``session-correction`` scores logged by the corrections
MCP tool, groups them by repository and workflow, and creates Ambient Code
Platform sessions that analyze the corrections and propose improvements to
workflow instructions, CLAUDE.md, and pattern files.

Usage:
    python scripts/feedback-loop/query_corrections.py \
        --langfuse-host https://langfuse.example.com \
        --langfuse-public-key pk-xxx \
        --langfuse-secret-key sk-xxx \
        --api-url https://ambient.example.com/api \
        --api-token <bot-token> \
        --project <project-name> \
        [--since-days 7] \
        [--min-severity 2] \
        [--min-corrections 2] \
        [--dry-run]
"""

import argparse
import json
import logging
import os
import sys
from collections import defaultdict
from datetime import datetime, timedelta, timezone
from pathlib import Path

import requests

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
)
logger = logging.getLogger(__name__)

SCORE_NAME = "session-correction"
LAST_RUN_FILE = Path(__file__).parent / ".last-run"
PAGE_SIZE = 100


# ------------------------------------------------------------------
# Langfuse query
# ------------------------------------------------------------------


def fetch_correction_scores(
    langfuse_host: str,
    public_key: str,
    secret_key: str,
    since: datetime,
    min_severity: int = 1,
) -> list[dict]:
    """Fetch session-correction scores from Langfuse via the v2 API.

    Uses Basic auth (public_key:secret_key) against the REST endpoint
    so we don't need the langfuse Python package installed in CI.
    """
    all_scores: list[dict] = []
    page = 1

    while True:
        url = f"{langfuse_host.rstrip('/')}/api/public/scores"
        params = {
            "name": SCORE_NAME,
            "dataType": "NUMERIC",
            "limit": PAGE_SIZE,
            "page": page,
        }

        resp = requests.get(
            url,
            params=params,
            auth=(public_key, secret_key),
            timeout=30,
        )
        resp.raise_for_status()
        data = resp.json()

        scores = data.get("data", data.get("scores", []))
        if not scores:
            break

        for score in scores:
            # Filter by timestamp
            created = score.get("createdAt") or score.get("timestamp", "")
            if created:
                try:
                    ts = datetime.fromisoformat(
                        created.replace("Z", "+00:00")
                    )
                    if ts < since:
                        continue
                except (ValueError, TypeError):
                    pass

            # Filter by minimum severity
            try:
                value = int(score.get("value") or 0)
            except (TypeError, ValueError):
                value = 0
            if value < min_severity:
                continue

            all_scores.append(score)

        # Check if there are more pages
        meta = data.get("meta", {})
        total_pages = meta.get("totalPages", 1)
        if page >= total_pages:
            break
        page += 1

    logger.info(f"Fetched {len(all_scores)} correction scores from Langfuse")
    return all_scores


# ------------------------------------------------------------------
# Grouping
# ------------------------------------------------------------------


def group_corrections(scores: list[dict]) -> list[dict]:
    """Group correction scores by (repo_url, workflow).

    Returns:
        List of group dicts with aggregated stats.
    """
    groups: dict[tuple, list] = defaultdict(list)

    for score in scores:
        metadata = score.get("metadata") or {}
        repo_url = metadata.get("repo_url", "unknown")
        workflow = metadata.get("workflow", "unknown")
        groups[(repo_url, workflow)].append(score)

    result = []
    for (repo_url, workflow), group_scores in groups.items():
        severities = [
            s.get("value", 0) for s in group_scores if s.get("value") is not None
        ]
        category_counts: dict[str, int] = defaultdict(int)
        scope_counts: dict[str, int] = defaultdict(int)

        corrections = []
        for s in group_scores:
            meta = s.get("metadata") or {}
            category = meta.get("category", "unknown")
            scope = meta.get("scope", "unknown")
            category_counts[category] += 1
            scope_counts[scope] += 1
            corrections.append(
                {
                    "category": category,
                    "severity": s.get("value", 0),
                    "scope": scope,
                    "description": s.get("comment", ""),
                    "correction_details": meta.get("correction_details", ""),
                    "session_name": meta.get("session_name", ""),
                    "trace_id": s.get("traceId", ""),
                }
            )

        result.append(
            {
                "repo_url": repo_url,
                "workflow": workflow,
                "corrections": corrections,
                "total_count": len(group_scores),
                "avg_severity": (
                    sum(severities) / len(severities) if severities else 0
                ),
                "category_counts": dict(category_counts),
                "scope_counts": dict(scope_counts),
            }
        )

    # Sort by total count descending
    result.sort(key=lambda g: g["total_count"], reverse=True)
    return result


# ------------------------------------------------------------------
# Prompt generation
# ------------------------------------------------------------------

SCOPE_FILE_MAP = {
    "workflow_instructions": ".ambient/ workflow files (system prompt, instructions)",
    "repo_context": "CLAUDE.md or .claude/ context files",
    "code_patterns": ".claude/patterns/ files",
    "documentation": "docs/ directory",
}


def build_improvement_prompt(group: dict) -> str:
    """Build an improvement prompt for an Ambient session.

    The session will analyze corrections and propose targeted
    improvements to the relevant files.
    """
    repo_url = group["repo_url"]
    workflow = group["workflow"]
    total = group["total_count"]
    avg_sev = group["avg_severity"]
    category_counts = group["category_counts"]
    scope_counts = group["scope_counts"]
    corrections = group["corrections"]

    top_category = max(category_counts, key=category_counts.get) if category_counts else "N/A"
    top_scope = max(scope_counts, key=scope_counts.get) if scope_counts else "N/A"

    # Build corrections detail section
    corrections_detail = ""
    for i, c in enumerate(corrections, 1):
        corrections_detail += (
            f"### Correction {i}\n"
            f"- **Category**: {c['category']}\n"
            f"- **Severity**: {c['severity']}/3\n"
            f"- **Scope**: {c['scope']}\n"
            f"- **Description**: {c['description']}\n"
        )
        if c.get("correction_details"):
            corrections_detail += f"- **Details**: {c['correction_details']}\n"
        if c.get("session_name"):
            corrections_detail += f"- **Session**: {c['session_name']}\n"
        corrections_detail += "\n"

    # Build scope targets section
    scope_targets = ""
    for scope, count in sorted(scope_counts.items(), key=lambda x: -x[1]):
        file_desc = SCOPE_FILE_MAP.get(scope, scope)
        scope_targets += f"- **{scope}** ({count} corrections) → Update {file_desc}\n"

    prompt = f"""# Feedback Loop: Improvement Session

## Context

You are analyzing {total} user corrections collected from Ambient Code Platform
sessions for the following context:

- **Repository**: {repo_url}
- **Workflow**: {workflow}
- **Period**: Last week
- **Average severity**: {avg_sev:.1f}/3
- **Most common correction type**: {top_category} ({category_counts.get(top_category, 0)} occurrences)
- **Most common improvement scope**: {top_scope} ({scope_counts.get(top_scope, 0)} occurrences)

## Category Breakdown

{chr(10).join(f'- **{cat}**: {count}' for cat, count in sorted(category_counts.items(), key=lambda x: -x[1]))}

## Target Files for Improvement

{scope_targets}

## Detailed Corrections

{corrections_detail}

## Your Task

1. **Analyze patterns**: Look for recurring themes across the corrections.
   Single incidents may be agent errors, but patterns indicate systemic gaps.

2. **Make targeted improvements** to the files indicated by the scope:
   - `workflow_instructions` → Update .ambient/ workflow files to provide
     better instructions or constraints
   - `repo_context` → Update CLAUDE.md or .claude/ context files to provide
     missing knowledge or patterns
   - `code_patterns` → Update .claude/patterns/ files to document correct
     approaches
   - `documentation` → Update docs/ with missing information

3. **Be surgical**: Only update files related to the identified scopes.
   Preserve existing content. Add or modify, don't replace wholesale.

4. **Add context comments**: For each change, include a brief note explaining
   which correction(s) prompted it.

5. **Commit and push**: Create a branch, commit your changes with a descriptive
   message, and push.

## Requirements

- Do NOT over-generalize from isolated incidents
- Focus on the highest-severity and most-frequent corrections first
- Each improvement should directly address one or more specific corrections
- Keep changes minimal and focused
- Test that any modified configuration files are still valid
"""

    return prompt


# ------------------------------------------------------------------
# Session creation
# ------------------------------------------------------------------


def create_improvement_session(
    api_url: str,
    api_token: str,
    project: str,
    prompt: str,
    group: dict,
) -> dict | None:
    """Create an Ambient session via the backend API.

    Returns:
        Session creation response dict, or None on failure.
    """
    repo_url = group["repo_url"]
    workflow = group["workflow"]

    # Build a display name from the repo
    repo_name = repo_url.rstrip("/").split("/")[-1] if repo_url else "unknown"
    display_name = f"Feedback Loop: {repo_name}"
    if workflow and workflow != "unknown":
        workflow_short = workflow.rstrip("/").split("/")[-1]
        display_name += f" ({workflow_short})"

    body = {
        "initialPrompt": prompt,
        "displayName": display_name,
        "environmentVariables": {
            "LANGFUSE_MASK_MESSAGES": "false",
        },
        "labels": {
            "feedback-loop": "true",
            "source": "github-action",
        },
    }

    # Add repo if it's a valid URL
    if repo_url and repo_url.startswith("http"):
        body["repos"] = [{"url": repo_url, "autoPush": True}]

    url = f"{api_url.rstrip('/')}/projects/{project}/agentic-sessions"

    try:
        resp = requests.post(
            url,
            headers={
                "Authorization": f"Bearer {api_token}",
                "Content-Type": "application/json",
            },
            json=body,
            timeout=30,
        )
        resp.raise_for_status()
        result = resp.json()
        logger.info(
            f"Created improvement session: {result.get('name', 'unknown')} "
            f"for {repo_name}"
        )
        return result
    except requests.RequestException as e:
        logger.error(f"Failed to create improvement session for {repo_name}: {e}")
        return None


# ------------------------------------------------------------------
# Timestamp persistence
# ------------------------------------------------------------------


def load_last_run() -> datetime | None:
    """Load last run timestamp from file."""
    if LAST_RUN_FILE.exists():
        try:
            ts_str = LAST_RUN_FILE.read_text().strip()
            return datetime.fromisoformat(ts_str)
        except (ValueError, OSError) as e:
            logger.warning(f"Could not read last run file: {e}")
    return None


def save_last_run(ts: datetime) -> None:
    """Save current run timestamp to file."""
    try:
        LAST_RUN_FILE.write_text(ts.isoformat())
    except OSError as e:
        logger.warning(f"Could not save last run file: {e}")


# ------------------------------------------------------------------
# Main
# ------------------------------------------------------------------


def main():
    parser = argparse.ArgumentParser(
        description="Query Langfuse corrections and create improvement sessions."
    )
    parser.add_argument("--langfuse-host", required=True, help="Langfuse host URL")
    parser.add_argument("--langfuse-public-key", required=True, help="Langfuse public key")
    parser.add_argument("--langfuse-secret-key", required=True, help="Langfuse secret key")
    parser.add_argument("--api-url", required=True, help="Ambient backend API URL")
    parser.add_argument("--api-token", required=True, help="Bot user token")
    parser.add_argument("--project", required=True, help="Ambient project name")
    parser.add_argument(
        "--since-days",
        type=int,
        default=7,
        help="Number of days to look back (default: 7)",
    )
    parser.add_argument(
        "--min-severity",
        type=int,
        default=2,
        choices=[1, 2, 3],
        help="Minimum severity to include (default: 2)",
    )
    parser.add_argument(
        "--min-corrections",
        type=int,
        default=2,
        help="Minimum corrections per group to trigger improvement (default: 2)",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Query and report without creating sessions",
    )

    args = parser.parse_args()

    # Determine the since date
    since = datetime.now(timezone.utc) - timedelta(days=args.since_days)

    # Check for last run timestamp
    last_run = load_last_run()
    if last_run and last_run > since:
        logger.info(f"Using last run timestamp: {last_run.isoformat()}")
        since = last_run

    logger.info(f"Querying corrections since {since.isoformat()}")

    # Fetch scores
    scores = fetch_correction_scores(
        langfuse_host=args.langfuse_host,
        public_key=args.langfuse_public_key,
        secret_key=args.langfuse_secret_key,
        since=since,
        min_severity=args.min_severity,
    )

    if not scores:
        logger.info("No corrections found in the specified period. Exiting.")
        save_last_run(datetime.now(timezone.utc))
        return

    # Group corrections
    groups = group_corrections(scores)

    logger.info(f"Found {len(groups)} repo/workflow groups:")
    for g in groups:
        logger.info(
            f"  - {g['repo_url']} / {g['workflow']}: "
            f"{g['total_count']} corrections, avg severity {g['avg_severity']:.1f}"
        )

    # Filter by minimum corrections threshold
    qualifying = [g for g in groups if g["total_count"] >= args.min_corrections]
    skipped = len(groups) - len(qualifying)

    if skipped:
        logger.info(
            f"Skipped {skipped} groups with fewer than "
            f"{args.min_corrections} corrections"
        )

    if not qualifying:
        logger.info("No groups meet the minimum corrections threshold. Exiting.")
        save_last_run(datetime.now(timezone.utc))
        return

    # Process each qualifying group
    sessions_created = 0
    for group in qualifying:
        prompt = build_improvement_prompt(group)

        if args.dry_run:
            logger.info(
                f"[DRY RUN] Would create session for "
                f"{group['repo_url']} / {group['workflow']} "
                f"({group['total_count']} corrections)"
            )
            logger.info(f"[DRY RUN] Prompt length: {len(prompt)} chars")
            continue

        result = create_improvement_session(
            api_url=args.api_url,
            api_token=args.api_token,
            project=args.project,
            prompt=prompt,
            group=group,
        )
        if result:
            sessions_created += 1

    # Save last run timestamp
    save_last_run(datetime.now(timezone.utc))

    if args.dry_run:
        logger.info(f"[DRY RUN] Would have created {len(qualifying)} sessions")
    else:
        logger.info(f"Created {sessions_created}/{len(qualifying)} improvement sessions")


if __name__ == "__main__":
    main()
