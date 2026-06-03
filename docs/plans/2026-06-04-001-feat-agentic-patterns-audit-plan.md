---
title: "feat: Agentic Patterns Audit Tool"
status: active
created: 2026-06-04
origin: "25 Patterns in Agentic Engineering (Greg Ceccarelli, SpecStory Press, 2026)"
---

# feat: Agentic Patterns Audit Tool

## Summary

Build a standalone Python CLI tool that scans any git repo and scores it against the 25 patterns from "25 Patterns in Agentic Engineering." The tool checks file existence, content patterns, git history, and directory structure to produce a scored report. This becomes a reusable benchmark for measuring how well a repo's agent infrastructure aligns with field-tested agentic engineering practices.

---

## Problem Frame

Teams building software with coding agents lack a way to measure whether their repo infrastructure supports effective agentic workflows. The 25 patterns distilled from 1,310 agent sessions and 4,670 commits represent concrete, proven practices — but there's no automated way to check a repo against them. This tool fills that gap: run it, get a score, see what's missing.

---

## Scope Boundaries

**In scope:**
- All 25 patterns with concrete heuristic checks
- Three output formats (human/tsv/json) matching existing benchmark conventions
- Repo-agnostic scanning (works on any git repo, not ACP-specific)
- Makefile integration
- Verification against the ACP platform repo

**Deferred to follow-up work:**
- CI integration (GitHub Actions workflow to run on PRs)
- Historical trend tracking across runs
- Per-pattern remediation suggestions beyond what the tool already reports

### Not in scope:
- Runtime behavior analysis (the tool is static analysis only)
- Agent session transcript analysis

---

## Key Technical Decisions

1. **Standalone stdlib-only Python script** — matches repo convention (see `scripts/sdk-options-drift-check.py`). No external dependencies. Uses `dataclasses`, `argparse`, `pathlib`, `subprocess`, `json`, `re`.

2. **Scoring: 0/1/2 per pattern** — 0 (no evidence), 1 (some evidence), 2 (strong evidence). Total out of 50. Letter grades: A (40+), B (30-39), C (20-29), D (10-19), F (<10). Simple enough to be useful, granular enough to show progress.

3. **Behavioral patterns get heuristic checks, not false precision** — Patterns like "Human Is Message Bus" (22) and "Delete and Regenerate" (18) are primarily about human practice. These get lighter checks with evidence where available and a "manual review" annotation where not.

4. **Output format auto-detection** — human if TTY, tsv if piped. Explicit `--format` overrides. Matches `scripts/benchmarks/component-bench.sh` convention.

---

## Implementation Units

### U1. Pattern definitions and scoring data model

**Goal:** Define the 25 patterns as structured data with their check metadata.

**Requirements:** Each pattern needs: id, name, part (I-VI), description, and a list of check functions to run.

**Dependencies:** None

**Files:**
- `scripts/agentic-patterns-audit.py` (create — pattern definitions section)

**Approach:** Use `@dataclass` for `PatternResult` (pattern_id, name, part, score, evidence list, manual_review flag). Define a `PATTERNS` list of tuples mapping each pattern to its checker function name and description. Group by the six parts from the book.

**Patterns to follow:** `scripts/sdk-options-drift-check.py` dataclass style.

**Test expectation: none** — pure data definitions, validated by U4 integration.

---

### U2. Scanner utility functions

**Goal:** Build reusable scanner primitives that pattern checkers compose.

**Requirements:** Must handle missing files/dirs gracefully. Must work on any git repo, not just ACP.

**Dependencies:** U1

**Files:**
- `scripts/agentic-patterns-audit.py` (scanner section)

**Approach:** Small set of composable functions:
- `file_exists(repo, *paths)` → bool — check if any of the given paths exist
- `glob_files(repo, pattern)` → list of paths
- `grep_files(repo, pattern, paths)` → list of (path, line_number, line) matches
- `grep_content(content, pattern)` → list of matches within already-loaded content
- `read_file(repo, path)` → str or None
- `git_log(repo, n, format_str)` → list of commit strings
- `count_files_matching(repo, dir_pattern, file_pattern)` → int

All functions take `repo: Path` as first arg. All return empty/None/False on missing paths rather than raising.

**Patterns to follow:** `pathlib.Path` usage in `sdk-options-drift-check.py`.

**Test scenarios:**
- Scanner functions return empty results for non-existent paths (no exceptions)
- `grep_files` finds known patterns in test fixture files
- `git_log` returns empty list for repos with no commits

---

### U3. Per-pattern check functions (25 checkers)

**Goal:** Implement one checker function per pattern, each returning a score (0/1/2) and evidence list.

**Requirements:** Each checker must be self-contained and composable from U2 primitives. Checkers must work on repos that aren't ACP — scan for general patterns, not hardcoded file paths.

**Dependencies:** U1, U2

**Files:**
- `scripts/agentic-patterns-audit.py` (checkers section — bulk of the file)

**Approach:** Each checker is a function `check_NN(repo: Path) -> PatternResult`. The mapping from patterns to signals:

**Part I: Verification Is the Job**
- `check_01` (Calibrated Distrust): Test dirs exist + CI config present + agent instructions mention verification/testing. Score 2 if all three, 1 if two, 0 if fewer.
- `check_02` (Source of Truth Outside Agent's Reach): Agent instructions reference external systems (URLs, APIs, billing, logs, running binary). Score 2 if multiple external refs, 1 if any.
- `check_03` (Premise Auditor): Agent instructions contain pushback/disconfirmation language ("push back", "don't be a sycophant", "challenge", "disagree"). Score 2 if explicit disconfirmation reward, 1 if any pushback language.
- `check_04` (Read-Only Turn): "Do not edit files" or read-only patterns in agent configs/prompts/skills. Score 2 if explicit read-only turn patterns, 1 if diagnostic-only references.

**Part II: Steering, Not Typing**
- `check_05` (Interrupt Is the Keyboard): Pre-commit hooks + PreToolUse hooks + permission gates. Score 2 if layered (hooks + gates), 1 if any hooks.
- `check_06` (Assert Ground Truth): Agent instructions reference runtime behavior, running binary, or system state the agent can't observe. Score 2 if explicit ground-truth assertions, 1 if runtime refs.
- `check_07` (Steer by Reference): "look at", "behave like", "@path" exemplar references in agent instructions pointing to existing code. Score 2 if multiple exemplar refs, 1 if any.
- `check_08` (Human Is Runtime Sensor): Manual test steps, screenshot workflows, "run the app" instructions. Score 2 if structured artifact-based verification, 1 if any manual test refs.
- `check_09` (License Agent to Ask): "ask me first", "clarifying questions", "if unclear" clauses in agent instructions. Score 2 if explicit licensing clause, 1 if any ask-before-act language.

**Part III: The Brief Is the Work**
- `check_10` (Prompt Is Engineered Brief): Labeled fields (Context:, Focus files:, Live evidence:) in agent templates or skills. Score 2 if structured brief templates found, 1 if any labeled fields.
- `check_11` (Structure Compounds): Structured templates, scope fences ("Do not edit"), allowlists, or focus-file patterns in agent configs. Score 2 if multiple structural elements, 1 if any.
- `check_12` (Pin Work to SHA): Full 40-char hex SHAs in docs, specs, or agent templates. Score 2 if multiple SHA references, 1 if any.
- `check_13` (Self-Grading Spec): Shell predicates, verification commands, or exit-condition checks in specs/goals. Score 2 if self-grading patterns with shell checks, 1 if verification sections.
- `check_14` (Commit at Phase Boundaries): Push restrictions in hooks or agent instructions ("never push", commit-at-boundary). Score 2 if push gated by hooks + instruction, 1 if either.

**Part IV: Docs Are the API Between Turns**
- `check_15` (Write for Reader Who Remembers Nothing): file:line citations in docs, source-of-truth tables. Score 2 if file:line citations found, 1 if structured references.
- `check_16` (Incident Doc Programs Next Agent): Postmortem/incident docs with structured format (dead ends, rejected approaches). Score 2 if structured incident docs, 1 if any postmortem-like docs.
- `check_17` (AS-BUILT Map): Per-subsystem architecture docs, dated or SHA-stamped. Score 2 if dated/stamped arch docs, 1 if any architecture docs.

**Part V: Code Is Cheap, Understanding Is Dear**
- `check_18` (Delete and Regenerate): Manual review recommended. Light check: look for "regenerate", "clean base", "start fresh" in docs/commit messages. Score 1 max with manual_review flag.
- `check_19` (Band-Aid Is a Verdict): TODO/FIXME/HACK density (lower is better) + explicit band-aid verdict language in docs. Score 2 if low density + explicit verdicts, 1 if either signal.
- `check_20` (Fix Generator, Defend Shape): Defensive parsers (try/except around JSON/YAML parsing with fallback) + prompt engineering patterns. Score 2 if both, 1 if either.

**Part VI: You Run an Org, Not a Pair**
- `check_21` (Model Is Swapped Dependency): Co-Authored-By trailers in recent git commits. Score 2 if >20% of recent commits have trailers, 1 if any.
- `check_22` (Human Is Message Bus): Cross-agent handoff patterns in docs/instructions. Manual review recommended. Score 1 max with manual_review flag.
- `check_23` (Agents Self-Assign): Task/backlog systems (beads, issue tracking) + agent self-assignment patterns. Score 2 if structured backlog, 1 if any task tracking.
- `check_24` (Fan Out Non-Overlapping Seams): Parallel agent configs, worktree infrastructure, lane-scoped prompts. Score 2 if worktree + parallel patterns, 1 if any.
- `check_25` (Make Agent Show Its Rails): Destructive-op guardrails, backup/scope-check requirements, --force-with-lease patterns in instructions. Score 2 if explicit rails requirements, 1 if any guardrails.

**Test scenarios:**
- Each checker returns a valid PatternResult with score in {0, 1, 2}
- Checkers return score 0 on a bare/empty repo without crashing
- Evidence list contains human-readable strings explaining what was found

---

### U4. Report formatters (human, json, tsv)

**Goal:** Format the 25 pattern results into the three output modes.

**Requirements:** Human output must be scannable with color. JSON must be machine-parseable. TSV must be agent-friendly. All must include the overall score and grade.

**Dependencies:** U1, U3

**Files:**
- `scripts/agentic-patterns-audit.py` (formatters section)

**Approach:**
- **Human format:** Grouped by Part (I-VI). Each pattern shows: `[score_indicator] Pattern NN: Name — evidence_summary`. Part subtotals. Overall score with letter grade. Color: green for 2, yellow for 1, dim for 0. Use ANSI codes only when stdout is a TTY.
- **JSON format:** `{"repo": path, "timestamp": iso, "score": N, "max_score": 50, "grade": "B", "parts": [{"name": "...", "patterns": [...]}]}`. Each pattern: `{"id": N, "name": "...", "score": N, "evidence": [...], "manual_review": bool}`.
- **TSV format:** Header row: `pattern_id\tname\tpart\tscore\tevidence`. One row per pattern. Summary row at end.

**Patterns to follow:** `scripts/benchmarks/component-bench.sh` output format conventions.

**Test scenarios:**
- Human format includes ANSI color codes when TTY, omits them when piped
- JSON output is valid JSON parseable by `json.loads()`
- TSV output has correct column count in every row
- All formats include the same total score and grade

---

### U5. CLI entry point and Makefile integration

**Goal:** Wire up argparse CLI and add Makefile target.

**Requirements:** `--format`, `--part`, `--verbose` flags. Positional repo-path defaulting to `.`. Exit code 0 always (this is an audit, not a gate).

**Dependencies:** U1-U4

**Files:**
- `scripts/agentic-patterns-audit.py` (main section)
- `Makefile` (add target)

**Approach:**
- `argparse` with: positional `repo` (default `.`), `--format` (choices: human/json/tsv, default auto-detect), `--part` (int 1-6, filter to one part), `--verbose` (show all evidence in human mode).
- Auto-detect format: human if `sys.stdout.isatty()`, else tsv.
- Makefile target: `audit-patterns` with `FORMAT=` and `PART=` optional vars, following the benchmark target pattern.
- Shebang `#!/usr/bin/env python3`, chmod +x.

**Patterns to follow:** Makefile target style from `benchmark:` target.

**Test scenarios:**
- Script runs successfully with no args from repo root
- `--format json` produces valid JSON
- `--part 1` filters to only Part I patterns (4 results)
- Exit code is always 0

---

## Verification

1. Run against ACP platform repo: `python3 scripts/agentic-patterns-audit.py . --format human --verbose`
2. Run against bare repo: `git init /tmp/bare-audit-test && python3 scripts/agentic-patterns-audit.py /tmp/bare-audit-test` — should produce all zeros without errors
3. Pipe test: `python3 scripts/agentic-patterns-audit.py . | head` — should produce TSV
4. JSON validation: `python3 scripts/agentic-patterns-audit.py . --format json | python3 -m json.tool`
5. Makefile: `make audit-patterns` runs successfully
6. Part filter: `python3 scripts/agentic-patterns-audit.py . --part 6` shows only Part VI patterns
