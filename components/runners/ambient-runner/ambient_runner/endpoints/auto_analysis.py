"""Automatic repo intelligence analysis.

Runs a multi-round LLM conversation against the Vertex AI API to
analyze repository structure, frameworks, caveats, and recent changes.
Results are stored in the intelligence DB and injected into the agent's
system prompt on subsequent sessions.

This is a platform feature — it works regardless of which runner
bridge (Claude, Gemini, LangGraph) the session uses.
"""

import asyncio
import json
import logging
import os
from pathlib import Path

logger = logging.getLogger(__name__)

# Track repos currently being analyzed (keyed by repo name).
_analyzing_repos: set[str] = set()


def is_analyzing(repo_name: str) -> bool:
    """Return True if auto-analysis is currently running for this repo."""
    return repo_name in _analyzing_repos


def is_intelligence_disabled() -> bool:
    """Check if intelligence features are disabled via env var."""
    return os.getenv("AMBIENT_DISABLE_INTELLIGENCE", "").lower() in ("true", "1", "yes")


async def run_auto_analysis(repo_name: str, repo_url: str, bridge=None):
    """Analyze a repo by calling the Vertex API directly — no bridge dependency.

    This is a platform feature that works regardless of which runner
    bridge (Claude, Gemini, LangGraph) the session uses. It:
    1. Checks if intelligence already exists (skip if so)
    2. Reads the repo directory tree and key files
    3. Calls the Vertex AI API with tool use (read_file, create_intelligence, memory_store)
    4. Stores the results in the intelligence DB

    Disabled when ``AMBIENT_DISABLE_INTELLIGENCE=true``.
    """
    if is_intelligence_disabled():
        logger.info(
            "Intelligence disabled (AMBIENT_DISABLE_INTELLIGENCE), skipping auto-analysis"
        )
        return

    # Wait for the repo notification to settle
    await asyncio.sleep(3)

    _analyzing_repos.add(repo_name)
    try:
        from ambient_runner.tools.intelligence_api import IntelligenceAPIClient

        try:
            intel_client = IntelligenceAPIClient()
        except ValueError:
            logger.debug(
                "Intelligence API client not available, skipping auto-analysis"
            )
            return

        if intel_client.intelligence_exists(repo_url):
            logger.info(
                f"Intelligence already exists for '{repo_name}', skipping auto-analysis"
            )
            return

        logger.info(f"No intelligence found for '{repo_name}', running auto-analysis")

        from ambient_runner.tools.vertex_client import (
            VertexAnthropicClient,
            AnthropicDirectClient,
        )

        llm_client = None
        try:
            llm_client = VertexAnthropicClient()
            logger.info("Auto-analysis using Vertex AI")
        except ValueError:
            pass

        if llm_client is None:
            try:
                llm_client = AnthropicDirectClient()
                logger.info("Auto-analysis using Anthropic API key")
            except ValueError:
                pass

        if llm_client is None:
            logger.debug(
                "No LLM client available (need ANTHROPIC_VERTEX_PROJECT_ID or ANTHROPIC_API_KEY), skipping auto-analysis"
            )
            return

        repo_context = _read_repo_context(repo_name)
        prompt = (
            f"Analyze the repository '{repo_name}' ({repo_url}) and store intelligence.\n\n"
            f"Here is the directory tree and key files:\n\n"
            f"{repo_context}\n\n"
            f"You can use `read_file` to read any file from the repo by path "
            f"(relative to the repo root). Read 3-5 key source files to understand "
            f"the architecture before storing your analysis.\n\n"
            f"Steps:\n"
            f"1. Use `read_file` to read the most important source files based on the tree above\n"
            f"2. Call `create_intelligence` with repo_url='{repo_url}' and accurate details "
            f"from the actual code you read\n"
            f"3. Call `memory_store` for 3-5 files with specific, accurate findings "
            f"(caveats, conventions, known issues you found in the code)\n"
            f"4. Be precise — cite actual function names, patterns, and code details"
        )

        # Build tools and handlers
        tools, handlers = _build_analysis_tools(intel_client, repo_name)

        # Run the LLM tool-call loop
        history = [{"role": "user", "content": prompt}]
        max_rounds = 10

        for _round in range(max_rounds):
            response = await asyncio.to_thread(
                llm_client.create_message,
                messages=history,
                system="You are a code analysis assistant. Analyze repositories and store findings using the provided tools.",
                tools=tools,
                max_tokens=16384,
            )

            stop_reason = response.get("stop_reason", "end_turn")
            content_blocks = response.get("content", [])
            history.append({"role": "assistant", "content": content_blocks})

            if stop_reason != "tool_use":
                break

            tool_results = []
            for block in content_blocks:
                if block["type"] == "tool_use":
                    handler = handlers.get(block["name"])
                    if handler:
                        result = handler(block.get("input", {}))
                    else:
                        result = json.dumps({"error": f"Unknown tool: {block['name']}"})
                    logger.info(
                        "Auto-analysis tool: %s → %s", block["name"], result[:80]
                    )
                    tool_results.append(
                        {
                            "type": "tool_result",
                            "tool_use_id": block["id"],
                            "content": result,
                        }
                    )

            history.append({"role": "user", "content": tool_results})

        logger.info(f"Auto-analysis complete for '{repo_name}' ({_round + 1} rounds)")

        # Signal the bridge to rebuild its system prompt so the newly
        # stored intelligence is included on the next user turn.
        if bridge is not None:
            bridge.mark_dirty()
            logger.info(
                "Marked bridge dirty — system prompt will rebuild with fresh intelligence"
            )

    except Exception as e:
        logger.warning(f"Auto-analysis failed for '{repo_name}' (non-critical): {e}")
    finally:
        _analyzing_repos.discard(repo_name)


def _build_analysis_tools(intel_client, repo_name: str):
    """Build the tool definitions and handlers for auto-analysis."""
    workspace = os.getenv("WORKSPACE_PATH", "/workspace")
    session_id = os.getenv("AGENTIC_SESSION_NAME", "")

    tools = [
        {
            "name": "read_file",
            "description": "Read a file from the repository by relative path.",
            "input_schema": {
                "type": "object",
                "properties": {
                    "path": {
                        "type": "string",
                        "description": "File path relative to repo root",
                    }
                },
                "required": ["path"],
            },
        },
        {
            "name": "create_intelligence",
            "description": "Create the repo intelligence record. Call this FIRST before memory_store.",
            "input_schema": {
                "type": "object",
                "properties": {
                    "repo_url": {"type": "string"},
                    "summary": {"type": "string"},
                    "language": {"type": "string"},
                    "framework": {"type": "string"},
                    "build_system": {"type": "string"},
                    "test_strategy": {"type": "string"},
                    "architecture": {"type": "string"},
                    "conventions": {"type": "string"},
                    "caveats": {"type": "string"},
                },
                "required": ["repo_url", "summary", "language"],
            },
        },
        {
            "name": "memory_store",
            "description": "Store a file-level finding (call AFTER create_intelligence).",
            "input_schema": {
                "type": "object",
                "properties": {
                    "repo_url": {"type": "string"},
                    "file_path": {"type": "string"},
                    "category": {
                        "type": "string",
                        "enum": ["investigation", "caveat", "review", "convention"],
                    },
                    "title": {"type": "string"},
                    "body": {"type": "string"},
                    "severity": {
                        "type": "string",
                        "enum": ["info", "warning", "critical"],
                    },
                    "confidence": {"type": "number"},
                },
                "required": ["repo_url", "file_path", "category", "title", "body"],
            },
        },
    ]

    def _read_file(args):
        path = args.get("path", "")
        repos_root = os.path.realpath(os.path.join(workspace, "repos"))
        if not os.path.isdir(repos_root):
            return json.dumps({"error": f"File not found: {path}"})
        for repo_dir in os.listdir(repos_root):
            candidate = os.path.realpath(os.path.join(repos_root, repo_dir, path))
            if not candidate.startswith(repos_root + os.sep):
                return json.dumps({"error": "Invalid file path"})
            if os.path.isfile(candidate):
                with open(candidate, errors="replace") as f:
                    content = f.read()
                return (
                    content[:4000] + "\n... (truncated)"
                    if len(content) > 4000
                    else content
                )
        return json.dumps({"error": f"File not found: {path}"})

    def _create_intelligence(args):
        data = {
            k: v
            for k, v in {
                "repo_url": args["repo_url"],
                "summary": args["summary"],
                "language": args["language"],
                "framework": args.get("framework"),
                "build_system": args.get("build_system"),
                "test_strategy": args.get("test_strategy"),
                "architecture": args.get("architecture"),
                "conventions": args.get("conventions"),
                "caveats": args.get("caveats"),
                "analyzed_by_session_id": session_id or None,
                "confidence": 0.9,
            }.items()
            if v is not None
        }
        try:
            result = intel_client.create_intelligence(data)
            return json.dumps({"created": True, "id": result.get("id")})
        except Exception as e:
            return json.dumps({"created": False, "error": str(e)})

    def _memory_store(args):
        intel = intel_client.lookup_intelligence(args["repo_url"])
        if not intel:
            return json.dumps(
                {
                    "stored": False,
                    "message": "No intelligence record — call create_intelligence first",
                }
            )
        data = {
            "intelligence_id": intel["id"],
            "file_path": args["file_path"],
            "category": args["category"],
            "title": args["title"],
            "body": args["body"],
            "severity": args.get("severity", "info"),
            "source_type": "agent_analysis",
            "confidence": args.get("confidence"),
        }
        if session_id:
            data["session_id"] = session_id
        finding = intel_client.create_finding(data)
        return json.dumps({"stored": True, "finding_id": finding.get("id")})

    handlers = {
        "read_file": _read_file,
        "create_intelligence": _create_intelligence,
        "memory_store": _memory_store,
    }
    return tools, handlers


def _read_repo_context(repo_name: str) -> str:
    """Read directory tree and key files from a cloned repo for analysis.

    Returns the directory tree plus README and build config. The LLM
    picks additional files to read via the read_file tool during analysis.
    """
    repo_path = Path(os.getenv("WORKSPACE_PATH", "/workspace")) / "repos" / repo_name
    if not repo_path.exists():
        return f"(Repository {repo_name} not found at {repo_path})"

    parts = []
    skip_dirs = {
        ".git",
        "node_modules",
        "__pycache__",
        "venv",
        ".venv",
        ".tox",
        "dist",
        "build",
        ".eggs",
        ".mypy_cache",
        ".pytest_cache",
    }

    # 1. Full directory tree (depth 3, first 100 files)
    try:
        tree_lines = []
        file_count = 0
        for root, dirs, files in os.walk(repo_path):
            dirs[:] = sorted(
                d for d in dirs if d not in skip_dirs and not d.startswith(".")
            )
            depth = str(root).replace(str(repo_path), "").count(os.sep)
            if depth > 3:
                dirs.clear()
                continue
            indent = "  " * depth
            dir_name = os.path.basename(root) if depth > 0 else repo_name
            tree_lines.append(f"{indent}{dir_name}/")
            for f in sorted(files):
                if file_count >= 100:
                    break
                tree_lines.append(f"{indent}  {f}")
                file_count += 1
        parts.append(
            "## Directory Structure\n```\n" + "\n".join(tree_lines) + "\n```\n"
        )
    except Exception as e:
        parts.append(f"(Could not read directory: {e})\n")

    # 2. Always include README and build config (if they exist)
    always_read = [
        "README.md",
        "README.rst",
        "README",
        "pyproject.toml",
        "setup.py",
        "setup.cfg",
        "package.json",
        "go.mod",
        "Cargo.toml",
    ]

    for filename in always_read:
        filepath = repo_path / filename
        if not filepath.exists() or not filepath.is_file():
            continue
        try:
            content = filepath.read_text(errors="replace")
            if len(content) > 3000:
                content = content[:3000] + "\n... (truncated)"
            parts.append(f"## {filename}\n```\n{content}\n```\n")
        except Exception as e:
            logger.debug("Failed to read %s: %s", filename, e)
            continue

    return "\n".join(parts)
