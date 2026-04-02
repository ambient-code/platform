"""
SDK Feature Analysis Report Generator

Parses GitHub release notes into structured feature data and generates
an intelligent adoption report for the Ambient Code Platform.
"""

import re
from dataclasses import dataclass, field
from enum import Enum


class FeatureStatus(Enum):
    NEW = "New"
    GA = "GA"
    BETA = "Beta"
    DEPRECATED = "Deprecated"
    INTERNAL = "Internal"


class RunnerRelevance(Enum):
    CLAUDE = "Claude"
    GEMINI = "Gemini"
    BOTH = "Both"
    NONE = "N/A"


class ActionRequired(Enum):
    NONE = "Auto"
    OPT_IN = "Opt-in"
    REVIEW = "Review"
    MIGRATE = "Migrate"


@dataclass
class Feature:
    """A single feature extracted from release notes."""

    name: str
    version: str
    description: str
    category: str  # "feature", "bugfix", "behavior_change", "breaking", "internal"
    default_behavior: str  # "Enabled", "Disabled", "N/A", or specific value
    status: FeatureStatus
    action: ActionRequired
    runner_relevance: RunnerRelevance
    pr_ref: str = ""  # e.g. "#667"


@dataclass
class FeatureReport:
    """Aggregated feature report across versions."""

    package_name: str
    from_version: str
    to_version: str
    features: list[Feature] = field(default_factory=list)


# Keywords that indicate specific runner relevance
CLAUDE_KEYWORDS = [
    "claude",
    "anthropic",
    "mcp",
    "tool_use",
    "thinking",
    "agent_definition",
    "agentdefinition",
    "session",
    "hook",
    "subagent",
    "allowed_tools",
    "claude_code",
    "claude cli",
    "bundled claude",
]
GEMINI_KEYWORDS = ["gemini", "google", "vertex"]

# Behavior change indicators
BEHAVIOR_CHANGE_KEYWORDS = [
    "changed",
    "breaking",
    "deprecated",
    "removed",
    "now.*instead",
    "no longer",
    "replaced",
    "migrat",
    "default.*changed",
    "takes precedence",
    "now-deprecated",
]

# Items that are purely internal and should not appear in behavior changes
INTERNAL_NOISE_PATTERNS = [
    r"updated bundled",
    r"updated ci",
    r"hardened.*workflow",
    r"added.*wheel",
    r"upload.*artifact",
    r"pypi.*publish",
    r"pypi.*storage",
    r"pypi.*quota",
    r"docs:.*clarified",
    r"macos.*wheel",
    r"manual updates",
    r"chore\(",
    r"update.*readme",
    r"update.*changelog",
    r"ci:",
    r"refactor\(",
    r"correct.*typo",
]


def _classify_section_header(header: str) -> str | None:
    """Classify a markdown section header into a category.

    Returns None for version-like headers that should be skipped.
    """
    if re.match(r"\d+\.\d+", header):
        return None
    if "feature" in header:
        return "feature"
    if "bug fix" in header or "bugfix" in header:
        return "bugfix"
    if "break" in header:
        return "breaking"
    if "deprecat" in header:
        return "deprecated"
    if "behavior" in header or "change" in header:
        return "behavior_change"
    if "internal" in header or "other" in header:
        return "internal"
    return ""


def parse_release_body(version: str, body: str) -> list[Feature]:
    """Parse a single release body into structured features.

    Handles multi-line bullets by joining sub-bullets (lines starting with
    whitespace + "-") into their parent bullet.

    Args:
        version: Version string.
        body: Raw markdown body from GitHub release.

    Returns:
        List of Feature objects.
    """
    if not body:
        return []

    features = []
    current_section = ""

    # First pass: group lines into logical bullets
    bullets: list[tuple[str, str]] = []  # (section, full_text)
    current_bullet_lines: list[str] = []

    for line in body.split("\n"):
        stripped = line.strip()

        # Detect section headers
        if stripped.startswith("### ") or stripped.startswith("## "):
            # Flush current bullet
            if current_bullet_lines:
                bullets.append((current_section, " ".join(current_bullet_lines)))
                current_bullet_lines = []

            prefix_len = 4 if stripped.startswith("### ") else 3
            header = stripped[prefix_len:].strip().lower()
            classified = _classify_section_header(header)
            if classified is not None:
                current_section = classified
            continue

        # Skip non-content lines
        if not stripped or stripped.startswith("---") or stripped.startswith("```"):
            continue
        if stripped.startswith("**PyPI:**") or stripped.startswith("pip install"):
            continue
        if stripped.startswith("Full Changelog:"):
            continue

        # Top-level bullet (supports both "- " and "* " formats)
        is_top_bullet = (
            stripped.startswith("- ") or stripped.startswith("* ")
        ) and not line.startswith("  ")
        if is_top_bullet:
            # Flush previous bullet
            if current_bullet_lines:
                bullets.append((current_section, " ".join(current_bullet_lines)))
            current_bullet_lines = [stripped[2:]]
        elif current_bullet_lines and (
            line.startswith("  ")
            or stripped.startswith("- ")
            or stripped.startswith("* ")
        ):
            # Sub-bullet or continuation — append to current bullet
            sub = re.sub(r"^[-*]\s+", "", stripped).strip()
            if sub:
                current_bullet_lines.append(sub)
        elif current_bullet_lines:
            # Continuation text
            if stripped:
                current_bullet_lines.append(stripped)

    # Flush last bullet
    if current_bullet_lines:
        bullets.append((current_section, " ".join(current_bullet_lines)))

    # Second pass: parse each bullet into a Feature
    for section, text in bullets:
        feature = _parse_feature_line(version, text, section)
        if feature:
            features.append(feature)

    return features


def _parse_feature_line(version: str, line: str, section: str) -> Feature | None:
    """Parse a single bullet line into a Feature."""
    if not line:
        return None

    # Strip trailing commit hash references like ([hash](url))
    line = re.sub(r"\s*\(\[[\da-f]+\]\([^)]+\)\)\s*$", "", line).strip()

    # Strip markdown links from the line for cleaner names
    clean_line = re.sub(r"\[([^\]]+)\]\([^)]+\)", r"\1", line)

    # Extract bold name if present: **Name**: description or **scope:** description
    name_match = re.match(r"\*\*([^*]+)\*\*:?\s*(.*)", clean_line, re.DOTALL)
    if name_match:
        name = name_match.group(1).strip().rstrip(":")
        description = name_match.group(2).strip()
        # If name is a generic scope (e.g. "api", "internal"), use description as name
        if (
            name.lower() in ("api", "internal", "tests", "docs", "chore")
            and description
        ):
            name = _extract_name(description)
        if not description:
            description = name
    else:
        name = _extract_name(clean_line)
        description = clean_line

    # Extract PR reference (GitHub #123 or [#123](url) format)
    pr_ref = ""
    pr_match = re.search(r"\[?#(\d+)\]?(?:\([^)]+\))?", line)
    if pr_match:
        pr_ref = f"#{pr_match.group(1)}"

    relevance = _detect_relevance(line)
    status = _detect_status(line, section)
    default_behavior = _detect_default(line, section)
    action = _detect_action(line, section)

    # Promote internal noise to true internal
    if _is_internal_noise(line) or _is_internal_noise(clean_line):
        section = "internal"
        status = FeatureStatus.INTERNAL
        action = ActionRequired.NONE
        default_behavior = "N/A"

    # Items whose bold scope was "internal" are internal
    if name_match and name_match.group(1).strip().rstrip(":").lower() == "internal":
        section = "internal"
        status = FeatureStatus.INTERNAL
        action = ActionRequired.NONE
        default_behavior = "N/A"

    # Detect behavior changes in feature sections
    category = section or "internal"
    if section == "feature" and _is_behavior_change(line):
        category = "behavior_change"

    return Feature(
        name=name,
        version=version,
        description=description,
        category=category,
        default_behavior=default_behavior,
        status=status,
        action=action,
        runner_relevance=relevance,
        pr_ref=pr_ref,
    )


def _extract_name(text: str) -> str:
    """Extract a clean feature name from unstructured text."""
    # Remove PR refs
    cleaned = re.sub(r"\s*\(#\d+\)", "", text)
    # Take first sentence or first 80 chars
    sentence_end = re.search(r"[.!](?:\s|$)", cleaned)
    if sentence_end and sentence_end.start() < 80:
        return cleaned[: sentence_end.start()].strip()
    if len(cleaned) > 80:
        return cleaned[:77].strip() + "..."
    return cleaned.strip()


def _detect_relevance(text: str) -> RunnerRelevance:
    """Determine which runner a feature is relevant to."""
    text_lower = text.lower()
    is_claude = any(kw in text_lower for kw in CLAUDE_KEYWORDS)
    is_gemini = any(kw in text_lower for kw in GEMINI_KEYWORDS)

    if is_claude and is_gemini:
        return RunnerRelevance.BOTH
    if is_claude:
        return RunnerRelevance.CLAUDE
    if is_gemini:
        return RunnerRelevance.GEMINI
    return RunnerRelevance.NONE


def _detect_status(text: str, section: str) -> FeatureStatus:
    """Determine feature status."""
    text_lower = text.lower()
    if "beta" in text_lower:
        return FeatureStatus.BETA
    if "deprecat" in text_lower:
        return FeatureStatus.DEPRECATED
    if section == "internal":
        return FeatureStatus.INTERNAL
    if "ga" in text_lower or "generally available" in text_lower:
        return FeatureStatus.GA
    return FeatureStatus.NEW


def _detect_default(text: str, section: str) -> str:
    """Determine default behavior of a feature."""
    if section in ("bugfix", "internal"):
        return "N/A"
    if section == "breaking":
        return "Changed"

    text_lower = text.lower()

    # Look for explicit default mentions
    default_match = re.search(r"default[s]?\s*(?:is|to|:)\s*[`\"']?(\w+)", text_lower)
    if default_match:
        return default_match.group(1).capitalize()

    # Features that add new fields/methods are typically opt-in
    if re.search(r"added.*(?:field|parameter|option|method|function)", text_lower):
        return "Opt-in"

    # Bug fixes are automatic
    if section == "bugfix":
        return "Auto"

    return "Available"


def _detect_action(text: str, section: str) -> ActionRequired:
    """Determine what action is needed to adopt this feature."""
    if section == "internal":
        return ActionRequired.NONE
    if section == "bugfix":
        return ActionRequired.NONE
    if section == "breaking":
        return ActionRequired.MIGRATE

    text_lower = text.lower()

    if "deprecat" in text_lower:
        return ActionRequired.MIGRATE
    if re.search(r"added.*(?:field|parameter|option|method|function)", text_lower):
        return ActionRequired.OPT_IN
    if _is_internal_noise(text):
        return ActionRequired.NONE
    if any(re.search(kw, text_lower) for kw in BEHAVIOR_CHANGE_KEYWORDS):
        return ActionRequired.REVIEW

    return ActionRequired.REVIEW


def _is_behavior_change(text: str) -> bool:
    """Check if a feature line describes a behavior change."""
    text_lower = text.lower()
    return any(re.search(kw, text_lower) for kw in BEHAVIOR_CHANGE_KEYWORDS)


def _is_internal_noise(text: str) -> bool:
    """Check if a line is internal noise that shouldn't appear in user-facing sections."""
    text_lower = text.lower()
    return any(re.search(p, text_lower) for p in INTERNAL_NOISE_PATTERNS)


def build_report(
    package_name: str,
    from_version: str,
    to_version: str,
    changelog_entries: list,
) -> FeatureReport:
    """Build a complete feature report from changelog entries."""
    report = FeatureReport(
        package_name=package_name,
        from_version=from_version,
        to_version=to_version,
    )

    for entry in changelog_entries:
        features = parse_release_body(entry.version, entry.body)
        report.features.extend(features)

    return report


def render_report_markdown(reports: list[FeatureReport]) -> str:
    """Render feature reports into a concise PR body.

    Structure: summary -> action items -> collapsible details.
    """
    lines = ["## SDK Version Bump", ""]

    # Version summary
    for r in reports:
        lines.append(f"- `{r.package_name}`: {r.from_version} -> {r.to_version}")
    lines.append("")

    # Collect categorized features across all reports
    migrate_items: list[tuple[Feature, str]] = []
    opt_in_items: list[tuple[Feature, str]] = []
    behavior_items: list[tuple[Feature, str]] = []
    new_features: list[tuple[Feature, str]] = []
    bugfixes: list[tuple[Feature, str]] = []

    for r in reports:
        for f in r.features:
            if f.status == FeatureStatus.INTERNAL:
                continue
            if _is_internal_noise(f.description):
                continue

            pair = (f, r.package_name)

            if f.action == ActionRequired.MIGRATE:
                migrate_items.append(pair)
            elif f.action == ActionRequired.OPT_IN and f.category == "feature":
                opt_in_items.append(pair)

            if f.category == "behavior_change" or (
                f.category == "feature" and _is_behavior_change(f.description)
            ):
                behavior_items.append(pair)
            elif f.category == "feature":
                new_features.append(pair)
            elif f.category == "bugfix":
                bugfixes.append(pair)

    # TL;DR
    lines.append("### TL;DR")
    lines.append("")
    if migrate_items:
        lines.append(
            f"- **{len(migrate_items)} migration(s) required** "
            "(deprecated APIs — see Action Items)"
        )
    if opt_in_items:
        lines.append(
            f"- **{len(opt_in_items)} opt-in feature(s)** available for adoption"
        )
    if behavior_items:
        lines.append(f"- {len(behavior_items)} behavior change(s) to review")
    lines.append(f"- {len(new_features)} new feature(s), {len(bugfixes)} bug fix(es)")
    lines.append("")

    # Action Items (always visible — the stuff that needs human attention)
    if migrate_items or opt_in_items:
        lines.append("## Action Items")
        lines.append("")

        if migrate_items:
            for f, pkg in migrate_items:
                desc = _truncate(f.description, 120)
                lines.append(
                    f"- **MIGRATE** — **{f.name}** (`{pkg}` v{f.version}): {desc}"
                )
            lines.append("")

        if opt_in_items:
            lines.append("**Opt-in features to evaluate:**")
            lines.append("")
            for f, pkg in opt_in_items:
                desc = _truncate(f.description, 100)
                lines.append(f"- [ ] **{f.name}** (`{pkg}` v{f.version}): {desc}")
            lines.append("")

    # Behavior Changes (always visible if present, but compact)
    if behavior_items:
        lines.append("## Behavior Changes")
        lines.append("")
        lines.append("| Change | Package | Version | Action |")
        lines.append("|--------|---------|---------|--------|")
        for f, pkg in behavior_items:
            lines.append(f"| **{f.name}** | `{pkg}` | {f.version} | {f.action.value} |")
        lines.append("")

    # New Features table — collapsible
    if new_features:
        lines.append(
            f"<details><summary><strong>New Features</strong> "
            f"({len(new_features)})</summary>"
        )
        lines.append("")
        lines.append(
            "| Feature | Package | Version | Claude | Gemini | Default | Action |"
        )
        lines.append(
            "|---------|---------|---------|--------|--------|---------|--------|"
        )
        for f, pkg in new_features:
            claude_col = _check_mark(
                f.runner_relevance in (RunnerRelevance.CLAUDE, RunnerRelevance.BOTH)
            )
            gemini_col = _check_mark(
                f.runner_relevance in (RunnerRelevance.GEMINI, RunnerRelevance.BOTH)
            )
            lines.append(
                f"| **{f.name}** | `{pkg}` | {f.version} "
                f"| {claude_col} | {gemini_col} "
                f"| {f.default_behavior} | {f.action.value} |"
            )
        lines.append("")
        lines.append("</details>")
        lines.append("")

    # Bug Fixes — collapsible
    if bugfixes:
        lines.append(
            f"<details><summary><strong>Bug Fixes</strong> ({len(bugfixes)})</summary>"
        )
        lines.append("")
        for f, pkg in bugfixes:
            desc = _truncate(f.description, 120)
            lines.append(f"- **{f.name}** (`{pkg}` {f.version}): {desc}")
        lines.append("")
        lines.append("</details>")
        lines.append("")

    # CLI version tracking — collapsible
    cli_versions = []
    for r in reports:
        for f in r.features:
            cli_match = re.search(
                r"[Uu]pdated bundled Claude CLI to version ([\d.]+)",
                f.description,
            )
            if cli_match:
                cli_versions.append((f.version, cli_match.group(1)))
    if cli_versions:
        lines.append(
            "<details><summary><strong>Bundled Claude CLI Versions</strong>"
            f" ({len(cli_versions)})</summary>"
        )
        lines.append("")
        lines.append("| SDK Version | CLI Version |")
        lines.append("|-------------|-------------|")
        for sdk_v, cli_v in cli_versions:
            lines.append(f"| {sdk_v} | {cli_v} |")
        lines.append("")
        lines.append("</details>")
        lines.append("")

    # Internal changes — collapsible one-liner summary
    for r in reports:
        internal = [f for f in r.features if f.status == FeatureStatus.INTERNAL]
        if internal:
            lines.append(
                f"<details><summary>Internal changes: "
                f"<code>{r.package_name}</code> ({len(internal)})</summary>"
            )
            lines.append("")
            for f in internal:
                lines.append(f"- {f.name} ({f.version})")
            lines.append("")
            lines.append("</details>")
            lines.append("")

    lines.append("---")
    lines.append("_Automated by `.github/workflows/sdk-version-bump.yml`_")

    return "\n".join(lines)


def _check_mark(condition: bool) -> str:
    return "Yes" if condition else "-"


def _truncate(text: str, max_len: int) -> str:
    """Truncate text to max_len, appending ellipsis if needed."""
    if len(text) <= max_len:
        return text
    return text[: max_len - 3].rstrip() + "..."
