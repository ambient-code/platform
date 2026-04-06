"""
Git guardrails for detecting and classifying destructive git operations.

Provides command validation to identify dangerous git and GitHub CLI/API
operations that could cause irreversible damage (branch deletion, force
pushes, history rewriting, etc.).

These checks are used by the system prompt builder to inject safety
instructions and can be used by future hook-based enforcement layers.
"""

import logging
import re

logger = logging.getLogger(__name__)


class GitGuardrailViolation:
    """Describes a guardrail violation found in a command."""

    def __init__(self, rule: str, severity: str, command: str, explanation: str) -> None:
        self.rule = rule
        self.severity = severity  # "block" or "warn"
        self.command = command
        self.explanation = explanation

    def __repr__(self) -> str:
        return f"GitGuardrailViolation(rule={self.rule!r}, severity={self.severity!r})"


# ---------------------------------------------------------------------------
# Destructive command patterns
# ---------------------------------------------------------------------------

# Patterns that should be blocked outright (irreversible / high-blast-radius)
_BLOCKED_PATTERNS: list[tuple[str, str, re.Pattern[str]]] = [
    (
        "delete_remote_ref",
        "Deleting a remote branch/ref can permanently close associated PRs",
        re.compile(
            r"""
            (?:gh\s+api|curl)        # GitHub API call via gh or curl
            .*                       # any intervening flags/args
            -X\s*DELETE              # HTTP DELETE method
            .*                       # any intervening text
            /git/refs/               # targeting a git ref
            """,
            re.VERBOSE | re.IGNORECASE,
        ),
    ),
    (
        "api_force_update_ref",
        "Force-updating a remote ref via the GitHub API bypasses git safety mechanisms",
        re.compile(
            r"""
            (?:gh\s+api|curl)        # GitHub API call
            .*                       # any intervening flags/args
            (?:PATCH|PUT)            # HTTP update method
            .*                       # any intervening text
            /git/refs/               # targeting a git ref
            .*                       # any intervening text
            ["\']?force["\']?\s*     # force parameter
            :\s*true                 # set to true
            """,
            re.VERBOSE | re.IGNORECASE,
        ),
    ),
    (
        "api_create_commit_on_ref",
        "Creating commits directly via the GitHub API bypasses local git safeguards",
        re.compile(
            r"""
            (?:gh\s+api|curl)        # GitHub API call
            .*                       # any intervening flags/args
            (?:POST|PATCH|PUT)       # HTTP write method
            .*                       # any intervening text
            /git/(?:commits|trees|blobs)  # low-level git data API
            """,
            re.VERBOSE | re.IGNORECASE,
        ),
    ),
    (
        "force_push",
        "Force pushing overwrites remote history and can destroy others' work",
        re.compile(
            r"""
            git\s+push\s+           # git push command
            .*                      # any flags/args
            --force(?!\-with\-lease)  # --force but NOT --force-with-lease
            """,
            re.VERBOSE,
        ),
    ),
    (
        "force_push_short",
        "Force pushing (-f) overwrites remote history and can destroy others' work",
        re.compile(
            r"""
            git\s+push\s+           # git push command
            .*                      # any flags/args
            \s-[a-zA-Z]*f           # short flag containing -f
            """,
            re.VERBOSE,
        ),
    ),
    (
        "push_to_main",
        "Pushing directly to main/master can corrupt the default branch",
        re.compile(
            r"""
            git\s+push\s+           # git push command
            .*                      # remote name and flags
            \s(?:main|master)\b     # targeting main or master branch
            """,
            re.VERBOSE,
        ),
    ),
    (
        "reset_hard",
        "git reset --hard discards all uncommitted changes irreversibly",
        re.compile(
            r"""
            git\s+reset\s+          # git reset command
            .*                      # any flags
            --hard                  # hard reset flag
            """,
            re.VERBOSE,
        ),
    ),
    (
        "clean_force",
        "git clean -fd permanently deletes untracked files and directories",
        re.compile(
            r"""
            git\s+clean\s+          # git clean command
            .*                      # any flags
            -[a-zA-Z]*f             # force flag (required for clean to run)
            """,
            re.VERBOSE,
        ),
    ),
    (
        "checkout_discard",
        "git checkout -- . discards all unstaged changes irreversibly",
        re.compile(
            r"""
            git\s+checkout\s+       # git checkout command
            --\s+\.                 # discard all changes
            """,
            re.VERBOSE,
        ),
    ),
    (
        "branch_delete_remote",
        "Deleting a remote branch can permanently close associated PRs",
        re.compile(
            r"""
            git\s+push\s+           # git push command
            \S+\s+                  # remote name
            --delete\s+             # delete flag
            """,
            re.VERBOSE,
        ),
    ),
    (
        "branch_delete_remote_colon",
        "Deleting a remote branch via :branch syntax can permanently close associated PRs",
        re.compile(
            r"""
            git\s+push\s+           # git push command
            \S+\s+                  # remote name
            :\S+                    # :branch (delete syntax)
            """,
            re.VERBOSE,
        ),
    ),
]

# Patterns that should generate warnings (risky but sometimes necessary)
_WARN_PATTERNS: list[tuple[str, str, re.Pattern[str]]] = [
    (
        "rebase",
        "Rebasing rewrites commit history; create a backup branch first",
        re.compile(
            r"""
            git\s+rebase\s+         # git rebase command
            """,
            re.VERBOSE,
        ),
    ),
    (
        "force_with_lease",
        "Force push with lease is safer but still overwrites remote history",
        re.compile(
            r"""
            git\s+push\s+           # git push command
            .*                      # any flags/args
            --force-with-lease      # safer force push
            """,
            re.VERBOSE,
        ),
    ),
    (
        "amend_commit",
        "Amending commits rewrites history; avoid if already pushed",
        re.compile(
            r"""
            git\s+commit\s+         # git commit command
            .*                      # any flags
            --amend                 # amend flag
            """,
            re.VERBOSE,
        ),
    ),
]


def check_command(command: str) -> list[GitGuardrailViolation]:
    """Check a shell command for git guardrail violations.

    Args:
        command: The shell command string to validate.

    Returns:
        List of violations found (empty if command is safe).
    """
    if not command or not command.strip():
        return []

    violations: list[GitGuardrailViolation] = []

    for rule, explanation, pattern in _BLOCKED_PATTERNS:
        if pattern.search(command):
            violations.append(
                GitGuardrailViolation(
                    rule=rule,
                    severity="block",
                    command=command,
                    explanation=explanation,
                )
            )

    for rule, explanation, pattern in _WARN_PATTERNS:
        if pattern.search(command):
            violations.append(
                GitGuardrailViolation(
                    rule=rule,
                    severity="warn",
                    command=command,
                    explanation=explanation,
                )
            )

    return violations


def has_blocking_violation(command: str) -> bool:
    """Return True if the command contains any blocking git guardrail violation."""
    violations = check_command(command)
    return any(v.severity == "block" for v in violations)


def format_violations(violations: list[GitGuardrailViolation]) -> str:
    """Format violations into a human-readable message."""
    if not violations:
        return ""

    lines = ["Git guardrail violations detected:"]
    for v in violations:
        marker = "BLOCKED" if v.severity == "block" else "WARNING"
        lines.append(f"  [{marker}] {v.rule}: {v.explanation}")
    return "\n".join(lines)


# ---------------------------------------------------------------------------
# Token redaction helpers
# ---------------------------------------------------------------------------

# Patterns that match common token/secret formats in commands
_TOKEN_PATTERNS: list[re.Pattern[str]] = [
    # GitHub PATs (classic and fine-grained)
    re.compile(r"ghp_[A-Za-z0-9]{36,}"),
    re.compile(r"github_pat_[A-Za-z0-9_]{36,}"),
    # GitLab tokens
    re.compile(r"glpat-[A-Za-z0-9\-_]{20,}"),
    # Generic Bearer/token in URLs
    re.compile(r"(?<=://)([^:]+):([^@]+)@", re.IGNORECASE),
]


def redact_tokens_in_command(command: str) -> str:
    """Redact known token patterns in a command string.

    Args:
        command: The command string that may contain tokens.

    Returns:
        Command with tokens replaced by [REDACTED].
    """
    result = command
    for pattern in _TOKEN_PATTERNS:
        if pattern.groups:
            # For patterns with groups (like URL credentials), replace the whole match
            result = pattern.sub("[REDACTED]@", result)
        else:
            result = pattern.sub("[REDACTED]", result)
    return result
