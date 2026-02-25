"""
Inject shared platform commands into the Claude Code commands directory.

Copies command files from /app/commands/ (baked into the runner image)
into /app/.claude/commands/ so Claude Code picks them up as user-level
slash commands.
"""

import logging
import shutil
from pathlib import Path

logger = logging.getLogger(__name__)

BUNDLED_COMMANDS_DIR = Path("/app/commands")
CLAUDE_COMMANDS_DIR = Path("/app/.claude/commands")


def inject_platform_commands(
    source: Path = BUNDLED_COMMANDS_DIR,
    target: Path = CLAUDE_COMMANDS_DIR,
) -> int:
    """Copy bundled command files into the Claude Code commands directory.

    Args:
        source: Directory containing bundled command ``.md`` files.
        target: Claude Code user-level commands directory.

    Returns:
        Number of command files injected.
    """
    if not source.is_dir():
        logger.debug(f"No bundled commands directory at {source}")
        return 0

    target.mkdir(parents=True, exist_ok=True)

    count = 0
    for src_file in sorted(source.iterdir()):
        if not src_file.is_file():
            continue
        dst_file = target / src_file.name
        shutil.copy2(src_file, dst_file)
        count += 1
        logger.debug(f"Injected command: {src_file.name} -> {dst_file}")

    logger.info(f"Injected {count} platform command(s) into {target}")
    return count
