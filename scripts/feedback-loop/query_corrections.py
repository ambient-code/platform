#!/usr/bin/env python3
"""
Feedback Loop: Query Langfuse corrections and create improvement sessions.

Queries Langfuse for ``session-correction`` scores logged by the corrections
MCP tool, groups them by workflow, and creates Ambient Code Platform sessions
that analyze the corrections and propose improvements to workflow instructions,
CLAUDE.md, and pattern files.

Usage:
    python scripts/feedback-loop/query_corrections.py \
        --langfuse-host https://langfuse.example.com \
        --langfuse-public-key pk-xxx \
        --langfuse-secret-key sk-xxx \
        --api-url https://ambient.example.com/api \
        --api-token <bot-token> \
        --project <project-name> \
        [--since-days 7] \
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

CORRECTION_TYPE_DESCRIPTIONS = {
    "incomplete": "missed something that should have been done",
    "incorrect": "did the wrong thing",
    "out_of_scope": "worked on wrong files or area",
    "style": "right result, wrong approach or pattern",
}


# ------------------------------------------------------------------
# Langfuse query
# ------------------------------------------------------------------


def fetch_correction_scores(
    langfuse_host: str,
    public_key: str,
    secret_key: str,
    since: datetime,
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
            "dataType": "CATEGORICAL",
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
    """Group correction scores by workflow (repo_url, branch, path).

    Returns:
        List of group dicts with aggregated stats.
    """
    groups: dict[tuple, list] = defaultdict(list)

    for score in scores:
        metadata = score.get("metadata") or {}
        workflow_repo_url = metadata.get("workflow_repo_url", "")
        workflow_branch = metadata.get("workflow_branch", "")
        workflow_path = metadata.get("workflow_path", "")
        groups[(workflow_repo_url, workflow_branch, workflow_path)].append(score)

    result = []
    for (workflow_repo_url, workflow_branch, workflow_path), group_scores in groups.items():
        type_counts: dict[str, int] = defaultdict(int)
        seen_repos: list[dict] = []
        seen_repo_urls: set[str] = set()

        corrections = []
        for s in group_scores:
            meta = s.get("metadata") or {}

            # correction_type is the score value for CATEGORICAL scores
            correction_type = s.get("value") or meta.get("correction_type", "unknown")
            type_counts[correction_type] += 1

            # Collect unique repos across all corrections in this group
            repos_raw = meta.get("repos", "")
            if repos_raw:
                try:
                    repos = json.loads(repos_raw) if isinstance(repos_raw, str) else repos_raw
                    for r in repos:
                        url = r.get("url", "")
                        if url and url not in seen_repo_urls:
                            seen_repo_urls.add(url)
                            seen_repos.append(r)
                except Exception:
                    pass

            corrections.append(
                {
                    "correction_type": correction_type,
                    "agent_action": meta.get("agent_action", s.get("comment", "")),
                    "user_correction": meta.get("user_correction", ""),
                    "session_name": meta.get("session_name", ""),
                    "trace_id": s.get("traceId", ""),
                }
            )

        result.append(
            {
                "workflow_repo_url": workflow_repo_url,
                "workflow_branch": workflow_branch,
                "workflow_path": workflow_path,
                "repos": seen_repos,
                "corrections": corrections,
                "total_count": len(group_scores),
                "correction_type_counts": dict(type_counts),
            }
        )

    # Sort by total count descending
    result.sort(key=lambda g: g["total_count"], reverse=True)
    return result


# ------------------------------------------------------------------
# Prompt generation
# ------------------------------------------------------------------


def build_improvement_prompt(group: dict) -> str:
    """Build an improvement prompt for an Ambient session.

    The session will analyze corrections and propose targeted
    improvements to workflow instructions and repo context files.
    """
    workflow_repo_url = group["workflow_repo_url"]
    workflow_branch = group["workflow_branch"]
    workflow_path = group["workflow_path"]
    repos = group.get("repos", [])
    total = group["total_count"]
    type_counts = group["correction_type_counts"]
    corrections = group["corrections"]

    top_type = max(type_counts, key=type_counts.get) if type_counts else "N/A"

    # Build corrections detail section
    corrections_detail = ""
    for i, c in enumerate(corrections, 1):
        corrections_detail += (
            f"### Correction {i} ({c['correction_type']})\n"
            f"- **Agent did**: {c['agent_action']}\n"
            f"- **User corrected to**: {c['user_correction']}\n"
        )
        if c.get("session_name"):
            corrections_detail += f"- **Session**: {c['session_name']}\n"
        corrections_detail += "\n"

    # Build correction type breakdown
    type_breakdown = "\n".join(
        f"- **{t}** ({CORRECTION_TYPE_DESCRIPTIONS.get(t, t)}): {count}"
        for t, count in sorted(type_counts.items(), key=lambda x: -x[1])
    )

    # Build repos section
    repos_section = ""
    if repos:
        repos_list = "\n".join(
            f"- {r.get('url', '')} (branch: {r.get('branch', 'default')})"
            for r in repos
        )
        repos_section = f"\n## Target Repositories\n\n{repos_list}\n"

    prompt = f"""# Feedback Loop: Improvement Session

## Context

You are analyzing {total} user corrections collected from Ambient Code Platform
sessions for the following workflow:

- **Workflow**: `{workflow_path}`
- **Workflow repo**: {workflow_repo_url} (branch: {workflow_branch or 'default'})
- **Most common correction type**: {top_type} ({type_counts.get(top_type, 0)} occurrences)
{repos_section}
## Correction Type Breakdown

{type_breakdown}

## Detailed Corrections

{corrections_detail}
## Your Task

1. **Analyze patterns**: Look for recurring themes across the corrections.
   Single incidents may be agent errors, but patterns indicate systemic gaps.

2. **Make targeted improvements**:
   - Update workflow files in `{workflow_path}` (system prompt, instructions)
     if the workflow is guiding the agent incorrectly or incompletely
   - Update `CLAUDE.md` or `.claude/` context files in the target repositories
     if the agent lacked necessary knowledge about those repos
   - Update `.claude/patterns/` files if the agent consistently used wrong patterns

3. **Use the corrections as a guide**: For each change, ask "would this correction
   have been prevented if this information existed in the context?"

4. **Be surgical**: Only update files directly related to the corrections.
   Preserve existing content. Add or modify — do not replace wholesale.

5. **Commit and push**: Create a branch, commit your changes with a descriptive
   message, and push.

## Requirements

- Do NOT over-generalize from isolated incidents
- Focus on the most frequent correction types first
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
    workflow_repo_url = group["workflow_repo_url"]
    workflow_branch = group["workflow_branch"]
    workflow_path = group["workflow_path"]
    repos = group.get("repos", [])

    workflow_short = workflow_path.rstrip("/").split("/")[-1] if workflow_path else ""
    repo_short = (
        workflow_repo_url.rstrip("/").split("/")[-1]
        if workflow_repo_url
        else "unknown"
    )
    display_name = f"Feedback Loop: {repo_short}"
    if workflow_short:
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

    # Build repo list: workflow repo first, then target repos
    session_repos = []
    if workflow_repo_url and workflow_repo_url.startswith("http"):
        workflow_repo: dict = {"url": workflow_repo_url, "autoPush": True}
        if workflow_branch:
            workflow_repo["branch"] = workflow_branch
        session_repos.append(workflow_repo)

    for r in repos:
        if r.get("url", "").startswith("http"):
            target_repo: dict = {"url": r["url"]}
            if r.get("branch"):
                target_repo["branch"] = r["branch"]
            session_repos.append(target_repo)

    if session_repos:
        body["repos"] = session_repos

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
            f"for {repo_short}"
        )
        return result
    except requests.RequestException as e:
        logger.error(f"Failed to create improvement session for {repo_short}: {e}")
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
    )

    if not scores:
        logger.info("No corrections found in the specified period. Exiting.")
        save_last_run(datetime.now(timezone.utc))
        return

    # Group corrections
    groups = group_corrections(scores)

    logger.info(f"Found {len(groups)} workflow groups:")
    for g in groups:
        logger.info(
            f"  - {g['workflow_repo_url']} / {g['workflow_path']}: "
            f"{g['total_count']} corrections"
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
                f"{group['workflow_repo_url']} / {group['workflow_path']} "
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
