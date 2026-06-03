#!/usr/bin/env python3
"""Audit a git repo against the 25 Patterns in Agentic Engineering.

Scans agent instructions, hooks, docs, git history, and directory structure
to score how well a repo aligns with field-tested agentic engineering practices.

Reference: "25 Patterns in Agentic Engineering" by Greg Ceccarelli
(SpecStory Press, 2026)
"""

from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from dataclasses import asdict, dataclass, field
from datetime import datetime, timezone
from pathlib import Path

# ---------------------------------------------------------------------------
# U1: Data model
# ---------------------------------------------------------------------------

PARTS = {
    1: "Verification Is the Job",
    2: "Steering, Not Typing",
    3: "The Brief Is the Work",
    4: "Docs Are the API Between Turns",
    5: "Code Is Cheap, Understanding Is Dear",
    6: "You Run an Org, Not a Pair",
}


@dataclass
class PatternResult:
    pattern_id: int
    name: str
    part: int
    score: int  # 0, 1, or 2
    evidence: list[str] = field(default_factory=list)
    manual_review: bool = False


PATTERN_DEFS: list[tuple[int, str, int]] = [
    (1, "Calibrated Distrust", 1),
    (2, "Keep a Source of Truth Outside the Agent's Reach", 1),
    (3, "The Premise Auditor", 1),
    (4, "The Read-Only Turn", 1),
    (5, "The Interrupt Is the Keyboard", 2),
    (6, "Assert the Ground Truth, Collapse the Branch", 2),
    (7, "Steer by Reference, Not Spec", 2),
    (8, "The Human Is the Runtime Sensor", 2),
    (9, "License the Agent to Ask Before It Acts", 2),
    (10, "The Prompt Is an Engineered Brief", 3),
    (11, "Structure Compounds, Incantation Depreciates", 3),
    (12, "Pin the Work to a SHA", 3),
    (13, "The Self-Grading Spec (Goal + Rider)", 3),
    (14, "Commit at Phase Boundaries, Never Push", 3),
    (15, "Write for a Reader Who Remembers Nothing", 4),
    (16, "The Incident Doc Programs the Next Agent", 4),
    (17, "The AS-BUILT Map Is a Test That Can Go Stale", 4),
    (18, "Delete and Regenerate from a Clean Base", 5),
    (19, "Band-Aid Is a Verdict, Not a Default", 5),
    (20, "Fix the Generator, Defend the Shape", 5),
    (21, "The Model Is a Swapped Dependency", 6),
    (22, "The Human Is the Message Bus", 6),
    (23, "Agents Self-Assign; You Pick the Lane", 6),
    (24, "Fan Out Along Non-Overlapping Seams", 6),
    (25, "Make the Agent Show Its Rails", 6),
]

# ---------------------------------------------------------------------------
# U2: Scanner utilities
# ---------------------------------------------------------------------------


def file_exists(repo: Path, *paths: str) -> list[str]:
    """Return list of paths that exist under repo."""
    return [p for p in paths if (repo / p).exists()]


def glob_files(repo: Path, pattern: str) -> list[Path]:
    """Glob for files under repo."""
    return sorted(repo.glob(pattern))


def read_file(repo: Path, path: str) -> str | None:
    """Read a file under repo, return None if missing."""
    fp = repo / path
    if not fp.is_file():
        return None
    try:
        return fp.read_text(errors="replace")
    except OSError:
        return None


def grep_files(
    repo: Path, pattern: str, paths: list[str], max_results: int = 50
) -> list[tuple[str, int, str]]:
    """Search for regex pattern in files. Returns (path, line_no, line)."""
    regex = re.compile(pattern, re.IGNORECASE)
    results: list[tuple[str, int, str]] = []
    for rel in paths:
        content = read_file(repo, rel)
        if content is None:
            continue
        for i, line in enumerate(content.splitlines(), 1):
            if regex.search(line):
                results.append((rel, i, line.strip()))
                if len(results) >= max_results:
                    return results
    return results


def grep_recursive(
    repo: Path,
    pattern: str,
    dirs: list[str],
    extensions: list[str] | None = None,
    max_results: int = 30,
) -> list[tuple[str, int, str]]:
    """Recursively grep directories for a pattern."""
    regex = re.compile(pattern, re.IGNORECASE)
    results: list[tuple[str, int, str]] = []
    for d in dirs:
        dp = repo / d
        if not dp.is_dir():
            continue
        for fp in dp.rglob("*"):
            if not fp.is_file():
                continue
            if extensions and fp.suffix not in extensions:
                continue
            try:
                content = fp.read_text(errors="replace")
            except OSError:
                continue
            rel = str(fp.relative_to(repo))
            for i, line in enumerate(content.splitlines(), 1):
                if regex.search(line):
                    results.append((rel, i, line.strip()))
                    if len(results) >= max_results:
                        return results
    return results


def git_log(repo: Path, n: int = 100, fmt: str = "%H %s") -> list[str]:
    """Return recent git log lines."""
    try:
        result = subprocess.run(
            ["git", "log", f"-{n}", f"--format={fmt}"],
            cwd=repo,
            capture_output=True,
            text=True,
            timeout=10,
        )
        if result.returncode != 0:
            return []
        return [line for line in result.stdout.strip().splitlines() if line]
    except (subprocess.TimeoutExpired, FileNotFoundError):
        return []


def count_pattern_in_files(
    repo: Path,
    pattern: str,
    dirs: list[str],
    extensions: list[str] | None = None,
) -> int:
    """Count total matches of pattern across files in dirs."""
    regex = re.compile(pattern, re.IGNORECASE)
    total = 0
    for d in dirs:
        dp = repo / d
        if not dp.is_dir():
            continue
        for fp in dp.rglob("*"):
            if not fp.is_file():
                continue
            if extensions and fp.suffix not in extensions:
                continue
            try:
                content = fp.read_text(errors="replace")
            except OSError:
                continue
            total += len(regex.findall(content))
    return total


def collect_agent_instruction_files(repo: Path) -> list[str]:
    """Find all agent instruction files in the repo."""
    candidates = [
        "CLAUDE.md",
        "AGENTS.md",
        ".claude/CLAUDE.md",
        "GEMINI.md",
        ".gemini/GEMINI.md",
        "COPILOT.md",
        ".github/copilot-instructions.md",
    ]
    found = file_exists(repo, *candidates)

    for d in [".claude/agents", ".claude/skills"]:
        for fp in glob_files(repo, f"{d}/**/*.md"):
            found.append(str(fp.relative_to(repo)))

    return found


def load_agent_instructions(repo: Path) -> str:
    """Load all agent instruction content into one string."""
    files = collect_agent_instruction_files(repo)
    parts = []
    for f in files:
        content = read_file(repo, f)
        if content:
            parts.append(content)
    return "\n".join(parts)


# ---------------------------------------------------------------------------
# U3: Pattern checkers
# ---------------------------------------------------------------------------


def check_01(repo: Path) -> PatternResult:
    """Calibrated Distrust: tests + CI + verification language."""
    evidence = []
    score = 0

    test_dirs = [
        d for d in ["tests", "test", "e2e", "__tests__", "spec"] if (repo / d).is_dir()
    ]
    test_files = glob_files(repo, "**/*_test.go") + glob_files(repo, "**/*.test.*")
    if test_dirs or test_files:
        evidence.append(
            f"Test infrastructure: {', '.join(test_dirs) if test_dirs else f'{len(test_files)} test files'}"
        )
        score += 1

    ci_files = file_exists(
        repo,
        ".github/workflows",
        ".gitlab-ci.yml",
        "Jenkinsfile",
        ".circleci/config.yml",
        ".travis.yml",
    )
    if ci_files:
        evidence.append(f"CI config: {', '.join(ci_files)}")
        score += 1

    instructions = load_agent_instructions(repo)
    verify_terms = re.findall(
        r"\b(verif(?:y|ied|ication)|test(?:s|ing)|prove|evidence)\b",
        instructions,
        re.IGNORECASE,
    )
    if verify_terms:
        evidence.append(
            f"Agent instructions mention verification ({len(verify_terms)} refs)"
        )

    return PatternResult(1, PATTERN_DEFS[0][1], 1, min(score, 2), evidence)


def check_02(repo: Path) -> PatternResult:
    """Source of Truth Outside Agent's Reach: external refs in instructions."""
    evidence = []
    score = 0
    instructions = load_agent_instructions(repo)

    url_matches = re.findall(r"https?://[^\s\)]+", instructions)
    if url_matches:
        evidence.append(
            f"External URLs in agent instructions ({len(url_matches)} found)"
        )
        score += 1

    external_terms = re.findall(
        r"\b(running binary|billing|metered|logs?|dashboard|monitor|external|oracle)\b",
        instructions,
        re.IGNORECASE,
    )
    if external_terms:
        evidence.append(f"External system references ({len(external_terms)} terms)")
        score += 1

    return PatternResult(2, PATTERN_DEFS[1][1], 1, min(score, 2), evidence)


def check_03(repo: Path) -> PatternResult:
    """Premise Auditor: pushback/disconfirmation in instructions."""
    evidence = []
    score = 0
    instructions = load_agent_instructions(repo)

    pushback_patterns = [
        r"push\s*back",
        r"don'?t\s+be\s+a?\s*sycophant",
        r"disagree",
        r"challenge",
        r"disconfirm",
        r"refus(?:e|al)",
        r"not\s+a\s+yes",
    ]
    for pat in pushback_patterns:
        matches = re.findall(pat, instructions, re.IGNORECASE)
        if matches:
            evidence.append(f"Pushback language: '{matches[0]}'")
            score += 1
            break

    reward_patterns = [
        r"reward.*refusal",
        r"pay.*disconfirm",
        r"the.*fix.*is.*the.*deliverable",
    ]
    for pat in reward_patterns:
        if re.search(pat, instructions, re.IGNORECASE):
            evidence.append("Explicit disconfirmation reward pattern")
            score += 1
            break

    return PatternResult(3, PATTERN_DEFS[2][1], 1, min(score, 2), evidence)


def check_04(repo: Path) -> PatternResult:
    """Read-Only Turn: 'Do not edit files' fences in agent configs."""
    evidence = []
    score = 0
    instructions = load_agent_instructions(repo)

    readonly_patterns = [
        r"do\s+not\s+edit\s+files",
        r"read[- ]only",
        r"no\s+edit",
        r"forensic\s+report",
        r"diagnosis\s+only",
        r"inspect.*don'?t.*modify",
    ]
    for pat in readonly_patterns:
        matches = re.findall(pat, instructions, re.IGNORECASE)
        if matches:
            evidence.append(f"Read-only pattern: '{matches[0]}'")
            score += 1

    return PatternResult(4, PATTERN_DEFS[3][1], 1, min(score, 2), evidence)


def check_05(repo: Path) -> PatternResult:
    """Interrupt Is the Keyboard: hooks and permission gates."""
    evidence = []
    score = 0

    hook_files = file_exists(
        repo,
        ".pre-commit-config.yaml",
        ".husky",
        ".git/hooks/pre-commit",
        ".git/hooks/pre-push",
    )
    if hook_files:
        evidence.append(f"Git hooks: {', '.join(hook_files)}")
        score += 1

    settings = read_file(repo, ".claude/settings.json")
    if settings:
        if "PreToolUse" in settings:
            evidence.append("Claude PreToolUse hooks configured")
            score += 1
        if "permissions" in settings.lower():
            evidence.append("Permission gates in settings")

    return PatternResult(5, PATTERN_DEFS[4][1], 2, min(score, 2), evidence)


def check_06(repo: Path) -> PatternResult:
    """Assert Ground Truth: runtime behavior refs in instructions."""
    evidence = []
    score = 0
    instructions = load_agent_instructions(repo)

    ground_truth_terms = [
        r"running\s+(binary|system|app)",
        r"runtime\s+behavior",
        r"the\s+real\s+(app|system|thing)",
        r"ground\s*truth",
        r"actual\s+(output|behavior|result)",
    ]
    for pat in ground_truth_terms:
        if re.search(pat, instructions, re.IGNORECASE):
            evidence.append("Ground truth reference found")
            score += 1
            break

    if re.search(
        r"run.*test|make.*test|npm\s+test|pytest|go\s+test", instructions, re.IGNORECASE
    ):
        evidence.append("Test execution commands in agent instructions")
        score += 1

    return PatternResult(6, PATTERN_DEFS[5][1], 2, min(score, 2), evidence)


def check_07(repo: Path) -> PatternResult:
    """Steer by Reference: exemplar refs in instructions."""
    evidence = []
    score = 0
    instructions = load_agent_instructions(repo)

    ref_patterns = [
        r"look\s+at\s+[\w/]+\.\w+",
        r"behave\s+like",
        r"follow.*pattern.*in",
        r"see\s+[\w/]+\.\w+",
        r"match.*existing",
        r"mirror.*convention",
        r"patterns?\s+to\s+follow",
    ]
    matches_found = 0
    for pat in ref_patterns:
        m = re.findall(pat, instructions, re.IGNORECASE)
        if m:
            matches_found += len(m)
            evidence.append(f"Reference pattern: '{m[0][:60]}'")

    if matches_found >= 3:
        score = 2
    elif matches_found >= 1:
        score = 1

    return PatternResult(7, PATTERN_DEFS[6][1], 2, min(score, 2), evidence)


def check_08(repo: Path) -> PatternResult:
    """Human Is Runtime Sensor: manual test steps, artifact workflows."""
    evidence = []
    score = 0
    instructions = load_agent_instructions(repo)

    sensor_patterns = [
        r"run\s+the\s+(app|server|dev)",
        r"screenshot",
        r"manual.*test",
        r"open.*browser",
        r"visually\s+confirm",
        r"start.*dev.*server",
    ]
    for pat in sensor_patterns:
        if re.search(pat, instructions, re.IGNORECASE):
            evidence.append("Runtime sensor pattern found")
            score += 1
            break

    if re.search(
        r"artifact|journal|receipt|evidence.*paste", instructions, re.IGNORECASE
    ):
        evidence.append("Artifact-based verification in instructions")
        score += 1

    return PatternResult(8, PATTERN_DEFS[7][1], 2, min(score, 2), evidence)


def check_09(repo: Path) -> PatternResult:
    """License Agent to Ask: ask-before-act clauses."""
    evidence = []
    score = 0
    instructions = load_agent_instructions(repo)

    ask_patterns = [
        r"ask\s+(me\s+)?first",
        r"clarifying\s+questions",
        r"if\s+(unclear|ambiguous|unsure)",
        r"ask\s+before",
        r"before\s+writing\s+anything",
        r"tell\s+me\s+what\s+you",
    ]
    matches_found = 0
    for pat in ask_patterns:
        if re.search(pat, instructions, re.IGNORECASE):
            matches_found += 1
            evidence.append("Ask-before-act clause found")

    if matches_found >= 2:
        score = 2
    elif matches_found >= 1:
        score = 1

    return PatternResult(9, PATTERN_DEFS[8][1], 2, min(score, 2), evidence)


def check_10(repo: Path) -> PatternResult:
    """Prompt Is Engineered Brief: labeled fields in templates."""
    evidence = []
    score = 0
    instructions = load_agent_instructions(repo)

    labeled_fields = [
        r"Context:",
        r"Focus\s+files?:",
        r"Live\s+evidence:",
        r"Deliverable:",
        r"Scope:",
        r"Constraints?:",
    ]
    found_fields = 0
    for pat in labeled_fields:
        if re.search(pat, instructions):
            found_fields += 1

    if found_fields >= 3:
        evidence.append(f"Structured brief with {found_fields} labeled fields")
        score = 2
    elif found_fields >= 1:
        evidence.append(f"Some labeled fields ({found_fields}) in agent templates")
        score = 1

    skills = glob_files(repo, ".claude/skills/**/*.md") + glob_files(
        repo, "skills/**/*.md"
    )
    if skills:
        evidence.append(f"Skill templates: {len(skills)} found")
        if score < 2:
            score = max(score, 1)

    return PatternResult(10, PATTERN_DEFS[9][1], 3, min(score, 2), evidence)


def check_11(repo: Path) -> PatternResult:
    """Structure Compounds: scope fences, allowlists, structured templates."""
    evidence = []
    score = 0
    instructions = load_agent_instructions(repo)

    structure_patterns = [
        (r"scope.*fence|Do\s+not\s+edit", "Scope fence"),
        (r"allowlist|allow.*list|focus.*files", "Allowlist/focus"),
        (r"##\s+(Context|Scope|Constraints|Requirements)", "Structured sections"),
    ]
    hits = 0
    for pat, label in structure_patterns:
        if re.search(pat, instructions, re.IGNORECASE):
            evidence.append(f"{label} pattern found")
            hits += 1

    if hits >= 2:
        score = 2
    elif hits >= 1:
        score = 1

    return PatternResult(11, PATTERN_DEFS[10][1], 3, min(score, 2), evidence)


def check_12(repo: Path) -> PatternResult:
    """Pin Work to SHA: full 40-char SHAs in docs/specs."""
    evidence = []
    score = 0

    sha_pattern = r"\b[0-9a-f]{40}\b"
    search_dirs = ["docs", "specs", ".claude"]
    results = grep_recursive(
        repo, sha_pattern, search_dirs, extensions=[".md", ".txt", ".yaml", ".yml"]
    )
    if len(results) >= 3:
        evidence.append(f"Full SHAs pinned in docs ({len(results)} found)")
        score = 2
    elif results:
        evidence.append(f"Some SHA references in docs ({len(results)} found)")
        score = 1

    instructions = load_agent_instructions(repo)
    if re.search(r"git\s+show\s+\S+:", instructions):
        evidence.append("git show <sha>:<path> pattern in instructions")
        score = max(score, 1)

    return PatternResult(12, PATTERN_DEFS[11][1], 3, min(score, 2), evidence)


def check_13(repo: Path) -> PatternResult:
    """Self-Grading Spec: shell predicates in specs/goals."""
    evidence = []
    score = 0

    shell_patterns = [
        r"```(bash|sh|shell)",
        r"grep\s+-[cqlr]",
        r"wc\s+-l",
        r"test\s+-[fedsz]",
        r"\$\(",
    ]
    search_dirs = ["specs", "docs", ".claude"]
    for pat in shell_patterns:
        results = grep_recursive(repo, pat, search_dirs, extensions=[".md", ".txt"])
        if results:
            evidence.append(f"Shell predicate pattern: {results[0][2][:50]}")
            score += 1
            break

    instructions = load_agent_instructions(repo)
    verification_section = re.search(
        r"##\s*Verification.*?\n(.*?)(?=\n##|\Z)",
        instructions,
        re.DOTALL | re.IGNORECASE,
    )
    if verification_section:
        evidence.append("Verification section in agent instructions")
        score += 1

    return PatternResult(13, PATTERN_DEFS[12][1], 3, min(score, 2), evidence)


def check_14(repo: Path) -> PatternResult:
    """Commit at Phase Boundaries, Never Push: push restrictions."""
    evidence = []
    score = 0

    hook_files = file_exists(repo, ".git/hooks/pre-push", "scripts/git-hooks/pre-push")
    if hook_files:
        evidence.append(f"Pre-push hooks: {', '.join(hook_files)}")
        score += 1

    instructions = load_agent_instructions(repo)
    push_patterns = [
        r"never\s+push",
        r"don'?t\s+push",
        r"push.*human",
        r"commit.*phase.*boundar",
        r"push.*review",
        r"never\s+force\s+push",
    ]
    for pat in push_patterns:
        if re.search(pat, instructions, re.IGNORECASE):
            evidence.append("Push restriction in agent instructions")
            score += 1
            break

    return PatternResult(14, PATTERN_DEFS[13][1], 3, min(score, 2), evidence)


def check_15(repo: Path) -> PatternResult:
    """Write for Reader Who Remembers Nothing: file:line citations."""
    evidence = []
    score = 0

    citation_pat = r"[\w/]+\.\w+:\d+"
    search_dirs = ["docs", "specs"]
    results = grep_recursive(
        repo, citation_pat, search_dirs, extensions=[".md", ".txt"]
    )
    if len(results) >= 5:
        evidence.append(f"file:line citations in docs ({len(results)} found)")
        score = 2
    elif results:
        evidence.append(f"Some file:line citations ({len(results)} found)")
        score = 1

    sot_pat = r"source.of.truth|ground.truth.table"
    sot = grep_recursive(repo, sot_pat, search_dirs, extensions=[".md"])
    if sot:
        evidence.append("Source-of-truth references in docs")
        score = max(score, 1)

    return PatternResult(15, PATTERN_DEFS[14][1], 4, min(score, 2), evidence)


def check_16(repo: Path) -> PatternResult:
    """Incident Doc Programs Next Agent: postmortem docs."""
    evidence = []
    score = 0

    incident_dirs = [
        "docs/incidents",
        "docs/postmortems",
        "docs/post-mortems",
        "incidents",
        "postmortems",
    ]
    found_dirs = [d for d in incident_dirs if (repo / d).is_dir()]
    if found_dirs:
        evidence.append(f"Incident doc directory: {', '.join(found_dirs)}")
        score += 1

    search_dirs = ["docs", "specs"]
    postmortem_pat = r"postmortem|incident.*report|dead.end|rejected.*approach"
    results = grep_recursive(repo, postmortem_pat, search_dirs, extensions=[".md"])
    if results:
        evidence.append(f"Postmortem/incident language in docs ({len(results)} refs)")
        score += 1

    return PatternResult(16, PATTERN_DEFS[15][1], 4, min(score, 2), evidence)


def check_17(repo: Path) -> PatternResult:
    """AS-BUILT Map: per-subsystem architecture docs."""
    evidence = []
    score = 0

    arch_patterns = [
        "docs/architecture",
        "docs/internal/architecture",
        "docs/adr",
        "docs/internal/adr",
    ]
    found = [d for d in arch_patterns if (repo / d).is_dir()]
    if found:
        md_files = []
        for d in found:
            md_files.extend(glob_files(repo, f"{d}/**/*.md"))
        evidence.append(
            f"Architecture docs: {len(md_files)} files in {', '.join(found)}"
        )
        score += 1

    sha_stamp_pat = r"(last\s+trued|verified|stamped|dated).*\b[0-9a-f]{7,40}\b"
    search_dirs = ["docs"]
    results = grep_recursive(repo, sha_stamp_pat, search_dirs, extensions=[".md"])
    if results:
        evidence.append("SHA-stamped architecture docs")
        score += 1

    return PatternResult(17, PATTERN_DEFS[16][1], 4, min(score, 2), evidence)


def check_18(repo: Path) -> PatternResult:
    """Delete and Regenerate: light check (manual review recommended)."""
    evidence = []
    score = 0

    regen_pat = r"regenerat|clean\s+base|start\s+fresh|rebuild\s+from|throw\s+away"
    search_dirs = ["docs", "specs"]
    results = grep_recursive(repo, regen_pat, search_dirs, extensions=[".md"])
    if results:
        evidence.append(f"Regeneration language in docs ({len(results)} refs)")
        score = 1

    return PatternResult(
        18, PATTERN_DEFS[17][1], 5, min(score, 2), evidence, manual_review=True
    )


def check_19(repo: Path) -> PatternResult:
    """Band-Aid Is a Verdict: TODO/FIXME density + verdict patterns."""
    evidence = []
    score = 0

    code_dirs = ["components", "src", "lib", "app", "pkg", "internal", "cmd"]
    existing_dirs = [d for d in code_dirs if (repo / d).is_dir()]
    if not existing_dirs:
        return PatternResult(
            19, PATTERN_DEFS[18][1], 5, 0, ["No code directories found to scan"]
        )

    todo_count = count_pattern_in_files(
        repo,
        r"\b(TODO|FIXME|HACK|XXX)\b",
        existing_dirs,
        extensions=[".go", ".py", ".ts", ".tsx", ".js", ".jsx", ".rb"],
    )

    total_lines = 0
    for d in existing_dirs:
        dp = repo / d
        if not dp.is_dir():
            continue
        for fp in dp.rglob("*"):
            if fp.is_file() and fp.suffix in {
                ".go",
                ".py",
                ".ts",
                ".tsx",
                ".js",
                ".jsx",
                ".rb",
            }:
                try:
                    total_lines += len(fp.read_text(errors="replace").splitlines())
                except OSError:
                    pass

    if total_lines > 0:
        density = todo_count / (total_lines / 1000)
        evidence.append(
            f"TODO/FIXME density: {density:.1f} per 1k lines ({todo_count} total)"
        )
        if density < 2.0:
            score += 1
            evidence.append("Low tech-debt marker density (good)")
        elif density > 10.0:
            evidence.append("High tech-debt marker density (review)")

    verdict_pat = r"verdict.*band.aid|band.aid.*verdict|workaround.*reject"
    search_dirs = ["docs", "specs"]
    results = grep_recursive(repo, verdict_pat, search_dirs, extensions=[".md"])
    if results:
        evidence.append("Explicit band-aid verdict language in docs")
        score += 1

    return PatternResult(19, PATTERN_DEFS[18][1], 5, min(score, 2), evidence)


def check_20(repo: Path) -> PatternResult:
    """Fix Generator, Defend Shape: defensive parsers + prompt patterns."""
    evidence = []
    score = 0

    code_dirs = ["components", "src", "lib", "app", "pkg", "internal"]
    existing_dirs = [d for d in code_dirs if (repo / d).is_dir()]

    parser_pat = (
        r"(json\.loads|JSON\.parse|yaml\.safe_load).*(?:try|catch|except|rescue)"
    )
    results = grep_recursive(
        repo,
        parser_pat,
        existing_dirs,
        extensions=[".go", ".py", ".ts", ".tsx", ".js", ".rb"],
    )
    if results:
        evidence.append(f"Defensive parsers: {len(results)} found")
        score += 1

    fallback_pat = r"(fallback|retry|graceful|defensive)\s*(pars|handl|mode)"
    results2 = grep_recursive(
        repo,
        fallback_pat,
        existing_dirs + ["docs"],
        extensions=[".go", ".py", ".ts", ".js", ".md"],
    )
    if results2:
        evidence.append("Fallback/defensive handling patterns")
        score += 1

    return PatternResult(20, PATTERN_DEFS[19][1], 5, min(score, 2), evidence)


def check_21(repo: Path) -> PatternResult:
    """Model Is Swapped Dependency: Co-Authored-By trailers."""
    evidence = []
    score = 0

    commits = git_log(repo, 100, "%H|||%b")
    if not commits:
        return PatternResult(
            21, PATTERN_DEFS[20][1], 6, 0, ["No git history available"]
        )

    coauthor_count = sum(
        1 for c in commits if "Co-authored-by:" in c or "Co-Authored-By:" in c
    )
    if commits:
        pct = coauthor_count / len(commits) * 100
        evidence.append(
            f"Co-Authored-By trailers: {coauthor_count}/{len(commits)} commits ({pct:.0f}%)"
        )
        if pct >= 20:
            score = 2
        elif coauthor_count > 0:
            score = 1

    return PatternResult(21, PATTERN_DEFS[20][1], 6, min(score, 2), evidence)


def check_22(repo: Path) -> PatternResult:
    """Human Is Message Bus: cross-agent handoff patterns (manual review)."""
    evidence = []
    score = 0
    instructions = load_agent_instructions(repo)

    handoff_patterns = [
        r"handoff|hand.*off",
        r"paste.*into.*another",
        r"cross.*agent|multi.*agent",
        r"second.*model|other.*agent",
        r"red[- ]team",
        r"carry.*state.*between",
    ]
    for pat in handoff_patterns:
        if re.search(pat, instructions, re.IGNORECASE):
            evidence.append("Cross-agent handoff pattern in instructions")
            score = 1
            break

    return PatternResult(
        22, PATTERN_DEFS[21][1], 6, min(score, 2), evidence, manual_review=True
    )


def check_23(repo: Path) -> PatternResult:
    """Agents Self-Assign; You Pick the Lane: backlog/task systems."""
    evidence = []
    score = 0

    task_indicators = file_exists(
        repo, ".beads", "TODO.md", ".github/ISSUE_TEMPLATE", ".linear", "backlog.md"
    )
    if task_indicators:
        evidence.append(f"Task/backlog system: {', '.join(task_indicators)}")
        score += 1

    instructions = load_agent_instructions(repo)
    if re.search(
        r"self[- ]assign|pick.*lane|backlog|next.*task|available.*work",
        instructions,
        re.IGNORECASE,
    ):
        evidence.append("Agent self-assignment language in instructions")
        score += 1

    return PatternResult(23, PATTERN_DEFS[22][1], 6, min(score, 2), evidence)


def check_24(repo: Path) -> PatternResult:
    """Fan Out Non-Overlapping Seams: parallel agent configs."""
    evidence = []
    score = 0

    worktree_indicators = file_exists(repo, ".claude/worktrees", ".worktrees")
    if worktree_indicators:
        evidence.append(f"Worktree infrastructure: {', '.join(worktree_indicators)}")
        score += 1

    instructions = load_agent_instructions(repo)
    parallel_patterns = [
        r"parallel.*agent|fan.*out|non[- ]overlapping\s+seam",
        r"worktree|isolation.*worktree",
        r"lane\s+\d|scoped\s+to\s+its\s+own",
    ]
    for pat in parallel_patterns:
        if re.search(pat, instructions, re.IGNORECASE):
            evidence.append("Parallel agent / fan-out patterns in instructions")
            score += 1
            break

    return PatternResult(24, PATTERN_DEFS[23][1], 6, min(score, 2), evidence)


def check_25(repo: Path) -> PatternResult:
    """Make Agent Show Its Rails: destructive-op guardrails."""
    evidence = []
    score = 0
    instructions = load_agent_instructions(repo)

    rails_patterns = [
        r"force[- ]with[- ]lease",
        r"backup.*branch",
        r"scope.*check|blast.*radius",
        r"show.*rail",
        r"narrate.*before.*act",
        r"destructive.*confirm",
        r"never\s+skip\s+hooks",
        r"--no-verify",
    ]
    hits = 0
    for pat in rails_patterns:
        if re.search(pat, instructions, re.IGNORECASE):
            evidence.append("Destructive-op guardrail in instructions")
            hits += 1

    if hits >= 2:
        score = 2
    elif hits >= 1:
        score = 1

    pre_push = file_exists(repo, ".git/hooks/pre-push", "scripts/git-hooks/pre-push")
    if pre_push:
        evidence.append("Pre-push hook enforcing rails")
        score = max(score, 1)

    return PatternResult(25, PATTERN_DEFS[24][1], 6, min(score, 2), evidence)


CHECKERS = [
    check_01,
    check_02,
    check_03,
    check_04,
    check_05,
    check_06,
    check_07,
    check_08,
    check_09,
    check_10,
    check_11,
    check_12,
    check_13,
    check_14,
    check_15,
    check_16,
    check_17,
    check_18,
    check_19,
    check_20,
    check_21,
    check_22,
    check_23,
    check_24,
    check_25,
]

# ---------------------------------------------------------------------------
# U4: Report formatters
# ---------------------------------------------------------------------------

GRADE_THRESHOLDS = [(40, "A"), (30, "B"), (20, "C"), (10, "D"), (0, "F")]


def compute_grade(total: int) -> str:
    for threshold, grade in GRADE_THRESHOLDS:
        if total >= threshold:
            return grade
    return "F"


def format_human(
    results: list[PatternResult], repo_path: str, verbose: bool = False
) -> str:
    is_tty = sys.stdout.isatty()

    def c(code: str, text: str) -> str:
        if not is_tty:
            return text
        return f"\033[{code}m{text}\033[0m"

    lines = []
    lines.append("")
    lines.append(c("1", f"  Agentic Patterns Audit: {repo_path}"))
    lines.append(f"  {'=' * 60}")
    lines.append("")

    current_part = 0
    part_scores: dict[int, list[int]] = {}

    for r in results:
        if r.part != current_part:
            if current_part != 0:
                lines.append("")
            current_part = r.part
            lines.append(c("1;34", f"  Part {_roman(r.part)}: {PARTS[r.part]}"))
            lines.append(f"  {'-' * 50}")

        part_scores.setdefault(r.part, []).append(r.score)

        if r.score == 2:
            indicator = c("32", "++")
        elif r.score == 1:
            indicator = c("33", "+ ")
        else:
            indicator = c("2", "  ")

        manual = c("2", " [manual review]") if r.manual_review else ""
        lines.append(f"  [{indicator}] {r.pattern_id:02d}. {r.name}{manual}")

        if verbose and r.evidence:
            for ev in r.evidence:
                lines.append(c("2", f"        {ev}"))

    total = sum(r.score for r in results)
    grade = compute_grade(total)

    lines.append("")
    lines.append(f"  {'=' * 60}")
    lines.append("")

    for part_num in sorted(part_scores):
        ps = sum(part_scores[part_num])
        pm = len(part_scores[part_num]) * 2
        lines.append(
            f"  Part {_roman(part_num):>4}: {ps:2d}/{pm:2d}  {PARTS[part_num]}"
        )

    lines.append(f"  {'─' * 40}")
    grade_color = {"A": "32", "B": "32", "C": "33", "D": "33", "F": "31"}.get(
        grade, "0"
    )
    lines.append(f"  Total: {total}/50  Grade: {c(grade_color, grade)}")
    lines.append("")

    manual_count = sum(1 for r in results if r.manual_review)
    if manual_count:
        lines.append(c("2", f"  {manual_count} pattern(s) flagged for manual review"))
        lines.append("")

    return "\n".join(lines)


def format_json(results: list[PatternResult], repo_path: str) -> str:
    total = sum(r.score for r in results)
    parts_data = {}
    for r in results:
        parts_data.setdefault(r.part, []).append(asdict(r))

    output = {
        "repo": repo_path,
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "score": total,
        "max_score": 50,
        "grade": compute_grade(total),
        "parts": [
            {
                "part": p,
                "name": PARTS[p],
                "patterns": parts_data.get(p, []),
            }
            for p in sorted(PARTS)
        ],
    }
    return json.dumps(output, indent=2)


def format_tsv(results: list[PatternResult], repo_path: str) -> str:
    lines = ["pattern_id\tname\tpart\tscore\tevidence"]
    for r in results:
        ev = "; ".join(r.evidence) if r.evidence else ""
        lines.append(f"{r.pattern_id}\t{r.name}\t{r.part}\t{r.score}\t{ev}")
    total = sum(r.score for r in results)
    lines.append(f"#\tTOTAL\t\t{total}\tGrade: {compute_grade(total)}")
    return "\n".join(lines)


def _roman(n: int) -> str:
    return {1: "I", 2: "II", 3: "III", 4: "IV", 5: "V", 6: "VI"}.get(n, str(n))


# ---------------------------------------------------------------------------
# U5: CLI entry point
# ---------------------------------------------------------------------------


def run_audit(repo_path: str, part_filter: int | None = None) -> list[PatternResult]:
    repo = Path(repo_path).resolve()
    if not repo.is_dir():
        print(f"Error: {repo_path} is not a directory", file=sys.stderr)
        sys.exit(2)

    results = []
    for checker in CHECKERS:
        result = checker(repo)
        if part_filter is not None and result.part != part_filter:
            continue
        results.append(result)

    return results


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Audit a repo against the 25 Patterns in Agentic Engineering.",
    )
    parser.add_argument(
        "repo",
        nargs="?",
        default=".",
        help="Path to git repo (default: current directory)",
    )
    parser.add_argument(
        "--format",
        choices=["human", "json", "tsv"],
        default=None,
        help="Output format (auto-detect if omitted)",
    )
    parser.add_argument(
        "--part",
        type=int,
        choices=range(1, 7),
        default=None,
        help="Filter to a specific part (1-6)",
    )
    parser.add_argument(
        "--verbose", action="store_true", help="Show detailed evidence in human format"
    )

    args = parser.parse_args()

    fmt = args.format
    if fmt is None:
        fmt = "human" if sys.stdout.isatty() else "tsv"

    results = run_audit(args.repo, args.part)

    if fmt == "human":
        print(format_human(results, args.repo, verbose=args.verbose))
    elif fmt == "json":
        print(format_json(results, args.repo))
    elif fmt == "tsv":
        print(format_tsv(results, args.repo))

    return 0


if __name__ == "__main__":
    sys.exit(main())
