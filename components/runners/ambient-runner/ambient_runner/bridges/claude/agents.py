"""Load custom agent definitions from .claude/agents/*.md files.

Scans markdown files with YAML-style frontmatter and converts them to
``AgentDefinition`` objects that the Claude Agent SDK can dispatch as
custom subagent types.
"""

import logging
from pathlib import Path
from typing import Any

from claude_agent_sdk import AgentDefinition

logger = logging.getLogger(__name__)


def _parse_agent_file(file_path: Path) -> tuple[dict[str, str], str]:
    """Parse frontmatter key-value pairs and body from a markdown file.

    Uses the same manual ``key: value`` parsing approach as
    ``ambient_runner.endpoints.content._parse_frontmatter`` to avoid
    adding a ``pyyaml`` dependency.

    Returns:
        A tuple of (frontmatter dict, body text after closing ``---``).
    """
    try:
        content = file_path.read_text(encoding="utf-8")
    except OSError:
        logger.warning("Cannot read agent file %s", file_path.name, exc_info=True)
        raise

    content = content.replace("\r\n", "\n")

    if not content.startswith("---\n"):
        return {}, content

    end_idx = content.find("\n---", 4)
    if end_idx == -1:
        return {}, content

    frontmatter_raw = content[4:end_idx]
    body = content[end_idx + 4 :].strip()  # skip past "\n---"

    metadata: dict[str, str] = {}
    for line in frontmatter_raw.split("\n"):
        if not line.strip():
            continue
        parts = line.split(":", 1)
        if len(parts) == 2:
            key = parts[0].strip()
            value = parts[1].strip().strip("\"'")
            metadata[key] = value

    return metadata, body


def _parse_string_list(value: Any) -> list[str] | None:
    """Parse a comma-separated string into a list of stripped strings.

    Returns ``None`` when *value* is falsy so that ``AgentDefinition``
    falls back to its default (inherit parent tools).
    """
    if not value:
        return None
    if isinstance(value, list):
        return [str(v).strip() for v in value if str(v).strip()]
    if isinstance(value, str):
        items = [v.strip() for v in value.split(",") if v.strip()]
        return items if items else None
    return None


def load_agents_from_directory(agents_dir: str) -> dict[str, AgentDefinition]:
    """Load ``AgentDefinition`` objects from ``.claude/agents/*.md`` files.

    Each markdown file is expected to have YAML-style frontmatter
    (``---`` delimited) followed by a body that becomes the agent's
    prompt.  Recognised frontmatter keys:

    * ``name`` – agent name (falls back to the filename stem)
    * ``description`` – when to use this agent (falls back to
      ``"Agent: {name}"``)
    * ``tools`` – comma-separated list of tool names
    * ``model`` – model alias or full model ID
    * ``skills`` – comma-separated list of skill names

    Args:
        agents_dir: Absolute path to the ``.claude/agents`` directory.

    Returns:
        Mapping of agent name → ``AgentDefinition``.  Empty dict when
        the directory does not exist or contains no valid agents.
    """
    agents: dict[str, AgentDefinition] = {}
    agents_path = Path(agents_dir)

    if not agents_path.is_dir():
        logger.debug("No agents directory at %s", agents_dir)
        return agents

    for md_file in sorted(agents_path.glob("*.md")):
        try:
            metadata, body = _parse_agent_file(md_file)

            name = metadata.get("name", md_file.stem)
            description = metadata.get("description", f"Agent: {name}")
            tools = _parse_string_list(metadata.get("tools"))
            skills = _parse_string_list(metadata.get("skills"))
            model = metadata.get("model")

            agents[name] = AgentDefinition(
                description=description,
                prompt=body,
                tools=tools,
                model=model,
                skills=skills,
            )
            logger.info("Loaded custom agent '%s' from %s", name, md_file.name)

        except Exception:
            logger.warning("Failed to load agent from %s", md_file.name, exc_info=True)

    logger.info("Loaded %d custom agent(s) from %s", len(agents), agents_dir)
    return agents
