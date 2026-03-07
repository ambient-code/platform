"""Write Gemini CLI custom commands for ambient platform features.

Custom commands are TOML files discovered from:
  - ~/.gemini/commands/   (global)
  - <project-root>/.gemini/commands/  (project-level, takes precedence)

We write platform commands to the project-level commands dir so they are
scoped to the session workspace and don't pollute the container's home.

Commands written:
  /ambient:evaluate-rubric  — evaluate output against .ambient/rubric.md
  /ambient:log-correction   — log a correction to the feedback loop
"""

import logging
from pathlib import Path

logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Command definitions
# ---------------------------------------------------------------------------

_EVALUATE_RUBRIC_TOML = '''\
description = "Evaluate session output against the workflow rubric in .ambient/rubric.md"
prompt = """
@{.ambient/rubric.md}

---

Evaluate the completed session output against each criterion in the rubric above.

Then call the `evaluate_rubric` tool (from the ambient-feedback MCP server) with:
- score: your overall numeric evaluation score
- comment: your evaluation reasoning, covering each criterion

Provide honest, calibrated scores with clear reasoning. Always read the rubric
before scoring — the file is embedded above.

{{args}}
"""
'''

_LOG_CORRECTION_TOML = '''\
description = "Log a correction to the improvement feedback loop before fixing the issue"
prompt = """
The user has corrected something you did or assumed. BEFORE fixing the issue,
call the `log_correction` tool (from the ambient-feedback MCP server) with:

- correction_type: one of:
    "incomplete"   — missed something that should have been done
    "incorrect"    — did the wrong thing
    "out_of_scope" — worked on wrong files or area
    "style"        — right result but wrong approach or pattern
- agent_action: what you did or assumed (be specific and honest)
- user_correction: exactly what the user said should have happened instead
- source: "human"

Use broad judgment — if the user is steering you away from something you already
did or decided, that is a correction. When in doubt, log it first, then fix.

{{args}}
"""
'''

# ---------------------------------------------------------------------------
# Public API
# ---------------------------------------------------------------------------


def write_gemini_commands(cwd_path: str) -> None:
    """Write ambient custom commands to the project-level .gemini/commands/ dir.

    Args:
        cwd_path: The session working directory (project root for Gemini CLI).
    """
    commands_dir = Path(cwd_path) / ".gemini" / "commands" / "ambient"
    try:
        commands_dir.mkdir(parents=True, exist_ok=True)
    except OSError as exc:
        logger.warning("Could not create commands dir %s: %s", commands_dir, exc)
        return

    _write_command(commands_dir / "evaluate-rubric.toml", _EVALUATE_RUBRIC_TOML)
    _write_command(commands_dir / "log-correction.toml", _LOG_CORRECTION_TOML)

    logger.info("Wrote Gemini CLI custom commands to %s", commands_dir)


def _write_command(path: Path, content: str) -> None:
    try:
        path.write_text(content, encoding="utf-8")
        path.chmod(0o644)
    except OSError as exc:
        logger.warning("Could not write command file %s: %s", path, exc)
