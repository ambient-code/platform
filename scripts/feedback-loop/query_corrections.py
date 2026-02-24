#!/usr/bin/env python3
"""
Feedback Loop: Query Langfuse corrections and create improvement sessions.

Queries Langfuse for ``session-correction`` scores logged by the corrections
MCP tool, groups them by target (one group per workflow, one per repo), and
creates Ambient Code Platform sessions that analyze the corrections and propose
improvements to workflow instructions, CLAUDE.md, and pattern files.

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
import time
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

CORRECTION_SOURCE_DESCRIPTIONS = {
    "human": "user-provided correction during a session",
    "rubric": "automatically detected from a rubric evaluation",
}


# ------------------------------------------------------------------
# Langfuse query
# ------------------------------------------------------------------


def fetch_correction_scores(
    langfuse_host: str,
    public_key: str,
    secret_key: str,
    since: datetime,
    verify_ssl: bool = True,
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
            verify=verify_ssl,
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
    """Group correction scores by target (type, repo_url, path).

    Each unique (target_type, target_repo_url, target_path) produces one
    group. ``target_type`` is either ``'workflow'`` or ``'repo'``.

    For **repo** targets the branch is intentionally excluded from the
    grouping key because sessions typically work on ephemeral feature
    branches while corrections apply to the repo as a whole.  Workflow
    targets keep the branch since different branches may have different
    workflow instructions.

    For backward compatibility, scores using the old schema
    (``workflow_repo_url`` instead of ``target_repo_url``) are migrated
    on the fly.

    Returns:
        List of group dicts with aggregated stats.
    """
    groups: dict[tuple, list] = defaultdict(list)

    for score in scores:
        metadata = score.get("metadata") or {}
        target_type, target_repo_url, target_branch, target_path = (
            _extract_target_fields(metadata)
        )
        # Repo corrections apply regardless of branch; workflows keep branch
        group_branch = target_branch if target_type == "workflow" else ""
        groups[
            (target_type, target_repo_url, group_branch, target_path)
        ].append(score)

    result = []
    for (target_type, target_repo_url, target_branch, target_path), group_scores in groups.items():
        type_counts: dict[str, int] = defaultdict(int)
        source_counts: dict[str, int] = defaultdict(int)

        corrections = []
        for s in group_scores:
            meta = s.get("metadata") or {}

            correction_type = s.get("value") or meta.get("correction_type", "unknown")
            type_counts[correction_type] += 1

            source = meta.get("source", "human")
            source_counts[source] += 1

            corrections.append(
                {
                    "correction_type": correction_type,
                    "source": source,
                    "agent_action": meta.get("agent_action", s.get("comment", "")),
                    "user_correction": meta.get("user_correction", ""),
                    "session_name": meta.get("session_name", ""),
                    "trace_id": s.get("traceId", ""),
                }
            )

        result.append(
            {
                "target_type": target_type,
                "target_repo_url": target_repo_url,
                "target_branch": target_branch,
                "target_path": target_path,
                "corrections": corrections,
                "total_count": len(group_scores),
                "correction_type_counts": dict(type_counts),
                "source_counts": dict(source_counts),
            }
        )

    # Sort by total count descending
    result.sort(key=lambda g: g["total_count"], reverse=True)
    return result


def _extract_target_fields(metadata: dict) -> tuple[str, str, str, str]:
    """Extract (target_type, target_repo_url, target_branch, target_path).

    Handles both the new schema (``target_*`` fields) and the old schema
    (``workflow_repo_url`` / ``workflow_branch`` / ``workflow_path``) for
    backward compatibility with scores logged before the migration.
    """
    target_type = metadata.get("target_type", "")
    target_repo_url = metadata.get("target_repo_url", "")
    target_branch = metadata.get("target_branch", "")
    target_path = metadata.get("target_path", "")

    if target_type:
        return target_type, target_repo_url, target_branch, target_path

    # Old schema: infer from workflow fields
    wf_url = metadata.get("workflow_repo_url", "")
    wf_branch = metadata.get("workflow_branch", "")
    wf_path = metadata.get("workflow_path", "")

    if wf_url:
        return "workflow", wf_url, wf_branch, wf_path

    return "repo", target_repo_url, target_branch, target_path


# ------------------------------------------------------------------
# Prompt generation
# ------------------------------------------------------------------


def build_improvement_prompt(group: dict) -> str:
    """Build an improvement prompt for an Ambient session.

    The prompt is tailored to the target_type: workflow corrections get
    instructions to fix workflow files; repo corrections get instructions
    to fix repo context files (CLAUDE.md, patterns, etc.).
    """
    target_type = group["target_type"]
    target_repo_url = group["target_repo_url"]
    target_branch = group["target_branch"]
    target_path = group["target_path"]
    total = group["total_count"]
    type_counts = group["correction_type_counts"]
    corrections = group["corrections"]

    top_type = max(type_counts, key=type_counts.get) if type_counts else "N/A"

    source_counts = group.get("source_counts", {})

    corrections_detail = ""
    for i, c in enumerate(corrections, 1):
        source_label = c.get("source", "human")
        source_tag = " [rubric]" if source_label == "rubric" else ""
        corrections_detail += (
            f"### Correction {i} ({c['correction_type']}{source_tag})\n"
            f"- **Agent did**: {c['agent_action']}\n"
            f"- **User corrected to**: {c['user_correction']}\n"
        )
        if c.get("session_name"):
            corrections_detail += f"- **Session**: {c['session_name']}\n"
        corrections_detail += "\n"

    type_breakdown = "\n".join(
        f"- **{t}** ({CORRECTION_TYPE_DESCRIPTIONS.get(t, t)}): {count}"
        for t, count in sorted(type_counts.items(), key=lambda x: -x[1])
    )

    source_breakdown = "\n".join(
        f"- **{s}** ({CORRECTION_SOURCE_DESCRIPTIONS.get(s, s)}): {count}"
        for s, count in sorted(source_counts.items(), key=lambda x: -x[1])
    )

    if target_type == "workflow":
        target_description = (
            f"- **Target type**: workflow\n"
            f"- **Workflow path**: `{target_path}`\n"
            f"- **Workflow repo**: {target_repo_url} (branch: {target_branch or 'default'})"
        )
        task_instructions = (
            "2. **Make targeted improvements**:\n"
            f"   - Update workflow files in `{target_path}` (system prompt, instructions)\n"
            "     where the workflow is guiding the agent incorrectly or incompletely\n"
            "   - Update rubric criteria if rubric-sourced corrections indicate misaligned expectations\n"
            "   - Update `.claude/patterns/` files if the agent consistently used wrong patterns"
        )
    else:
        target_description = (
            f"- **Target type**: repository\n"
            f"- **Repository**: {target_repo_url} (branch: {target_branch or 'default'})"
        )
        task_instructions = (
            "2. **Make targeted improvements**:\n"
            "   - Update `CLAUDE.md` or `.claude/` context files where the agent\n"
            "     lacked necessary knowledge about this repository\n"
            "   - Update `.claude/patterns/` files if the agent consistently used wrong patterns\n"
            "   - Add missing documentation that would have prevented these corrections"
        )

    prompt = f"""# Feedback Loop: Improvement Session

## Context

You are analyzing {total} corrections collected from Ambient Code Platform sessions.

{target_description}
- **Most common correction type**: {top_type} ({type_counts.get(top_type, 0)} occurrences)

## Correction Type Breakdown

{type_breakdown}

## Correction Sources

{source_breakdown}

## Detailed Corrections

{corrections_detail}
## Your Task

1. **Analyze patterns**: Look for recurring themes across the corrections.
   Single incidents may be agent errors, but patterns indicate systemic gaps.

{task_instructions}

3. **Use the corrections as a guide**: For each change, ask "would this correction
   have been prevented if this information existed in the context?"

4. **Be surgical**: Only update files directly related to the corrections.
   Preserve existing content. Add or modify — do not replace wholesale.

5. **Commit, push, and open a PR**: Commit your changes with a descriptive
   message, push to the feature branch using `git push -u origin <branch>`,
   then create a pull request with `gh pr create` targeting the default branch.
   NEVER push directly to main or master.

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


def _repo_short_name(url: str) -> str:
    """Extract short name from a repo URL."""
    name = url.rstrip("/").split("/")[-1]
    return name[:-4] if name.endswith(".git") else name


def _normalise_url(url: str) -> str:
    """Normalise a repo URL for comparison.

    Handles trailing ``/``, ``.git`` suffix, and SSH-to-HTTPS conversion
    (``git@github.com:org/repo`` → ``https://github.com/org/repo``).
    """
    u = url.strip().rstrip("/").removesuffix(".git")
    if u.startswith("git@"):
        u = u.replace(":", "/", 1).replace("git@", "https://", 1)
    return u.lower()


def create_improvement_session(
    api_url: str,
    api_token: str,
    project: str,
    prompt: str,
    group: dict,
    verify_ssl: bool = True,
) -> dict | None:
    """Create an Ambient session via the backend API.

    Returns:
        Session creation response dict, or None on failure.
    """
    target_type = group["target_type"]
    target_repo_url = group["target_repo_url"]
    target_branch = group["target_branch"]
    target_path = group["target_path"]

    repo_short = _repo_short_name(target_repo_url) if target_repo_url else "unknown"
    if target_type == "workflow":
        path_short = target_path.rstrip("/").split("/")[-1] if target_path else ""
        display_name = f"Feedback Loop: {repo_short}"
        if path_short:
            display_name += f" ({path_short})"
    else:
        display_name = f"Feedback Loop: {repo_short} (repo)"

    body: dict = {
        "initialPrompt": prompt,
        "displayName": display_name,
        "environmentVariables": {
            "LANGFUSE_MASK_MESSAGES": "false",
        },
        "labels": {
            "feedback-loop": "true",
            "target-type": target_type,
            "source": "github-action",
        },
    }

    if target_repo_url and target_repo_url.startswith("http"):
        repo_entry: dict = {"url": target_repo_url, "autoPush": True}
        if target_branch:
            repo_entry["branch"] = target_branch
        body["repos"] = [repo_entry]

    url = f"{api_url.rstrip('/')}/projects/{project}/agentic-sessions"

    max_retries = 3
    for attempt in range(max_retries):
        try:
            resp = requests.post(
                url,
                headers={
                    "Authorization": f"Bearer {api_token}",
                    "Content-Type": "application/json",
                },
                json=body,
                timeout=30,
                verify=verify_ssl,
            )
            resp.raise_for_status()
            result = resp.json()
            logger.info(
                f"Created improvement session: {result.get('name', 'unknown')} "
                f"for {target_type}:{repo_short}"
            )
            return result
        except requests.RequestException as e:
            is_server_error = (
                hasattr(e, "response")
                and e.response is not None
                and e.response.status_code == 500
            )
            if is_server_error and attempt < max_retries - 1:
                wait = 1.5 * (attempt + 1)
                logger.warning(
                    f"Session creation failed (attempt {attempt + 1}/{max_retries}), "
                    f"retrying in {wait}s: {e}"
                )
                time.sleep(wait)
                continue

            logger.error(
                f"Failed to create improvement session for "
                f"{target_type}:{repo_short}: {e}"
            )
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
# Structured output (for CI/CD integration)
# ------------------------------------------------------------------


def _group_summary(group: dict) -> dict:
    """Build a compact summary dict for one group (safe for JSON output)."""
    return {
        "target_type": group.get("target_type", ""),
        "target_repo_url": group.get("target_repo_url", ""),
        "target_path": group.get("target_path", ""),
        "total_count": group.get("total_count", 0),
        "correction_type_counts": group.get("correction_type_counts", {}),
        "source_counts": group.get("source_counts", {}),
    }


def _write_output(
    output_file: str,
    corrections_found: int,
    sessions_created: int,
    groups: list[dict],
) -> None:
    """Write a JSON summary to *output_file* if a path was provided."""
    if not output_file:
        return
    try:
        summary = {
            "corrections_found": corrections_found,
            "sessions_created": sessions_created,
            "groups": groups,
        }
        with open(output_file, "w") as f:
            json.dump(summary, f, indent=2)
        logger.info(f"Output written to {output_file}")
    except Exception as e:
        logger.warning(f"Failed to write output file: {e}")


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
    parser.add_argument(
        "--no-verify-ssl",
        action="store_true",
        help="Disable SSL certificate verification (for self-signed certs)",
    )
    parser.add_argument(
        "--repos-filter",
        default="",
        help="Comma-separated repo URLs to process (empty = all repos)",
    )
    parser.add_argument(
        "--output-file",
        default="",
        help="Write JSON summary to this file (for CI/CD integration)",
    )

    args = parser.parse_args()
    verify_ssl = not args.no_verify_ssl
    if not verify_ssl:
        import urllib3

        urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
        logger.info("SSL verification disabled")

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
        verify_ssl=verify_ssl,
    )

    corrections_found = len(scores)
    sessions_created = 0
    groups_summary: list[dict] = []

    if not scores:
        logger.info("No corrections found in the specified period. Exiting.")
        save_last_run(datetime.now(timezone.utc))
        _write_output(args.output_file, 0, 0, [])
        return

    # Group corrections
    groups = group_corrections(scores)

    logger.info(f"Found {len(groups)} target groups:")
    for g in groups:
        label = f"{g['target_type']}:{g['target_repo_url']}"
        if g["target_path"]:
            label += f" / {g['target_path']}"
        logger.info(f"  - {label}: {g['total_count']} corrections")

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
        _write_output(args.output_file, corrections_found, 0, [])
        return

    # Filter by repo allowlist
    if args.repos_filter:
        allowed = {
            _normalise_url(u)
            for u in args.repos_filter.split(",")
            if u.strip()
        }
        before = len(qualifying)
        qualifying = [
            g for g in qualifying
            if _normalise_url(g["target_repo_url"]) in allowed
        ]
        filtered = before - len(qualifying)
        if filtered:
            logger.info(f"Repos filter: skipped {filtered} groups not in allowlist")
        if not qualifying:
            logger.info("No groups match the repos filter. Exiting.")
            save_last_run(datetime.now(timezone.utc))
            _write_output(args.output_file, corrections_found, 0, [])
            return

    # Process each qualifying group
    for group in qualifying:
        prompt = build_improvement_prompt(group)

        if args.dry_run:
            label = f"{group['target_type']}:{group['target_repo_url']}"
            if group["target_path"]:
                label += f" / {group['target_path']}"
            logger.info(
                f"[DRY RUN] Would create session for "
                f"{label} ({group['total_count']} corrections)"
            )
            logger.info(f"[DRY RUN] Prompt length: {len(prompt)} chars")
            groups_summary.append(_group_summary(group))
            continue

        result = create_improvement_session(
            api_url=args.api_url,
            api_token=args.api_token,
            project=args.project,
            prompt=prompt,
            group=group,
            verify_ssl=verify_ssl,
        )
        if result:
            sessions_created += 1
        groups_summary.append(_group_summary(group))

    # Save last run timestamp
    save_last_run(datetime.now(timezone.utc))

    if args.dry_run:
        logger.info(f"[DRY RUN] Would have created {len(qualifying)} sessions")
    else:
        logger.info(f"Created {sessions_created}/{len(qualifying)} improvement sessions")

    _write_output(args.output_file, corrections_found, sessions_created, groups_summary)


if __name__ == "__main__":
    main()
