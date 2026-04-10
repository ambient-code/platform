#!/usr/bin/env python3
"""
Generate dynamic loading tips for the Ambient Code platform frontend.

Pulls release metadata from git history to create tips that highlight:
- First-time contributors
- Top commits by lines of code added (noteworthy changes)

Outputs a TypeScript file with a RELEASE_TIPS array alongside the static DEFAULT_LOADING_TIPS.
"""

import subprocess
import sys
import re


def run_git(args: list[str]) -> str:
    result = subprocess.run(
        ["git"] + args, capture_output=True, text=True
    )
    if result.returncode != 0:
        print(f"Warning: git {' '.join(args)} failed: {result.stderr}", file=sys.stderr)
        return ""
    return result.stdout.strip()


def get_first_time_contributors(latest_tag: str, current_authors: set[str]) -> list[str]:
    if not latest_tag:
        return sorted(current_authors)

    tag_date = run_git(["log", "-1", "--format=%ci", latest_tag])
    if not tag_date:
        return []

    prior_raw = run_git(["log", "--all", f"--before={tag_date}", "--format=%an"])
    prior_authors = set(prior_raw.split("\n")) if prior_raw else set()

    return sorted(current_authors - prior_authors)


def get_top_commits_by_loc(latest_tag: str, top_n: int = 3) -> list[dict]:
    """Get the top N commits by lines added between latest_tag and HEAD."""
    commit_range = f"{latest_tag}..HEAD" if latest_tag else "HEAD"
    raw = run_git([
        "log", commit_range,
        "--format=%h<DELIM>%s<DELIM>%an",
        "--numstat",
    ])
    if not raw:
        return []

    commits = []
    current = None

    for line in raw.split("\n"):
        if "<DELIM>" in line:
            if current:
                commits.append(current)
            parts = line.split("<DELIM>", 2)
            current = {
                "hash": parts[0],
                "subject": parts[1] if len(parts) > 1 else "",
                "author": parts[2] if len(parts) > 2 else "",
                "additions": 0,
            }
        elif current and line.strip():
            # numstat lines: <additions>\t<deletions>\t<file>
            numstat = line.split("\t")
            if len(numstat) >= 2 and numstat[0] != "-":
                try:
                    current["additions"] += int(numstat[0])
                except ValueError:
                    pass

    if current:
        commits.append(current)

    commits.sort(key=lambda c: c["additions"], reverse=True)
    return commits[:top_n]


def clean_subject(subject: str) -> str:
    """Strip conventional commit prefix and PR number for display."""
    cleaned = re.sub(r"^\w+(\([^)]*\))?:\s*", "", subject)
    cleaned = re.sub(r"\s*\(#\d+\)$", "", cleaned)
    return cleaned


def generate_tips(new_tag: str, latest_tag: str) -> list[str]:
    tips = []

    commit_range = f"{latest_tag}..HEAD" if latest_tag else "HEAD"
    author_raw = run_git(["log", commit_range, "--format=%an"])
    if not author_raw:
        return tips
    current_authors = set(author_raw.split("\n"))

    # First-time contributors (always first)
    first_timers = get_first_time_contributors(latest_tag, current_authors)
    for name in first_timers:
        tips.append(f"Welcome {name}, who made their first contribution in {new_tag}!")

    # Top 3 commits by lines added
    top_commits = get_top_commits_by_loc(latest_tag, top_n=3)
    for commit in top_commits:
        subject = clean_subject(commit["subject"])
        loc = commit["additions"]
        if subject and loc > 0:
            tips.append(f"New in {new_tag}: {subject} (+{loc:,} lines)")

    return tips


def select_tips(tips: list[str], count: int = 10) -> list[str]:
    """Select up to `count` tips, prioritizing first-timer shoutouts."""
    first_timer_tips = [t for t in tips if t.startswith("Welcome ")]
    other_tips = [t for t in tips if not t.startswith("Welcome ")]

    selected = first_timer_tips[:count]
    remaining = count - len(selected)
    if remaining > 0:
        selected.extend(other_tips[:remaining])

    return selected[:count]


STATIC_TIPS = [
    "Tip: Clone sessions to quickly duplicate your setup for similar tasks",
    "Tip: Export chat transcripts as Markdown or PDF for documentation",
    "Tip: Add multiple repositories as context for cross-repo analysis",
    "Tip: Stopped sessions can be resumed without losing your progress",
    "Tip: Check MCP Servers to see which tools are available in your session",
    "Tip: Repository URLs are remembered for quick re-use across sessions",
    "Tip: Connect Google Drive to export chats directly to your Drive",
    "Tip: Load custom workflows from your own Git repositories",
    "Tip: Use the Explorer panel to browse and download files created by AI",
]


def write_loading_tips_ts(tips: list[str], output_path: str):
    escaped = [t.replace("\\", "\\\\").replace('"', '\\"') for t in tips]
    release_lines = ",\n".join(f'  "{t}"' for t in escaped)

    static_lines = ",\n".join(f'  "{t}"' for t in STATIC_TIPS)

    content = (
        "/**\n"
        " * Release-generated loading tips for the Ambient Code platform.\n"
        " * Auto-generated by scripts/generate-loading-tips.py during the release pipeline.\n"
        " * These tips highlight recent changes, contributors, and platform milestones.\n"
        " *\n"
        " * DO NOT EDIT MANUALLY — this array is regenerated on every release.\n"
        " */\n"
        "export const RELEASE_TIPS: string[] = [\n"
        f"{release_lines},\n"
        "];\n"
        "\n"
        "/**\n"
        " * Default loading tips shown during AI response generation.\n"
        " * These are used as fallback when LOADING_TIPS env var is not configured.\n"
        " * Tips support markdown-style links: [text](url)\n"
        " */\n"
        "export const DEFAULT_LOADING_TIPS = [\n"
        "  ...RELEASE_TIPS,\n"
        f"{static_lines},\n"
        "];\n"
    )

    with open(output_path, "w") as f:
        f.write(content)

    print(f"Wrote {len(tips)} release tips to {output_path}")


def main():
    if len(sys.argv) < 4:
        print(f"Usage: {sys.argv[0]} <new_tag> <latest_tag> <repo> [output_path]", file=sys.stderr)
        sys.exit(1)

    new_tag = sys.argv[1]
    latest_tag = sys.argv[2]
    # repo arg kept for interface compatibility but not used currently
    output_path = sys.argv[4] if len(sys.argv) > 4 else "components/frontend/src/lib/loading-tips.ts"

    all_tips = generate_tips(new_tag, latest_tag)
    selected = select_tips(all_tips, count=10)

    print(f"Generated {len(all_tips)} candidate tips, selected {len(selected)}:")
    for i, tip in enumerate(selected, 1):
        print(f"  {i}. {tip}")

    write_loading_tips_ts(selected, output_path)


if __name__ == "__main__":
    main()
