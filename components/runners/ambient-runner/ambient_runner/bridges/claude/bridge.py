"""
ClaudeBridge — full-lifecycle PlatformBridge for the Claude Agent SDK.

Owns the entire Claude session lifecycle:
- Platform setup (auth, workspace, MCP, observability)
- Adapter creation and caching
- Session worker management (persistent SDK clients)
- Tracing middleware integration
- Interrupt and graceful shutdown
"""

import asyncio
import json
import logging
import os
import time
from collections import defaultdict
from pathlib import Path
from typing import Any, AsyncIterator, Optional

from ag_ui.core import BaseEvent, RunAgentInput
from ag_ui_claude_sdk import ClaudeAgentAdapter

from ambient_runner.bridge import (
    FrameworkCapabilities,
    PlatformBridge,
    _async_safe_manager_shutdown,
    setup_bridge_observability,
)
from ambient_runner.bridges.claude.session import SessionManager
from ambient_runner.platform.context import RunnerContext

logger = logging.getLogger(__name__)

# Maximum stderr lines kept in ring buffer for error reporting
_MAX_STDERR_LINES = 50


def _git_auth_env() -> dict[str, str]:
    """Build env dict with GIT_ASKPASS credential helper for tokens.

    Uses GIT_ASKPASS to provide credentials without embedding them in URLs
    or writing them to .git/config on disk.
    """
    env = {**os.environ, "GIT_TERMINAL_PROMPT": "0"}
    github_token = os.getenv("GITHUB_TOKEN", "").strip()
    gitlab_token = os.getenv("GITLAB_TOKEN", "").strip()
    if github_token or gitlab_token:
        # GIT_ASKPASS script that echoes the token for password prompts
        token = github_token or gitlab_token
        env["GIT_ASKPASS"] = "/bin/echo"
        if github_token:
            env["GIT_USERNAME"] = "x-access-token"
            env["GIT_PASSWORD"] = github_token
            # Configure git to use token via credential helper
            env["GIT_CONFIG_COUNT"] = "1"
            env["GIT_CONFIG_KEY_0"] = "credential.helper"
            env["GIT_CONFIG_VALUE_0"] = (
                f"!f() {{ echo username=x-access-token; echo password={token}; }}; f"
            )
        elif gitlab_token:
            env["GIT_CONFIG_COUNT"] = "1"
            env["GIT_CONFIG_KEY_0"] = "credential.helper"
            env["GIT_CONFIG_VALUE_0"] = (
                f"!f() {{ echo username=oauth2; echo password={token}; }}; f"
            )
    return env


async def _run_git(
    *args: str, cwd: str | None = None
) -> tuple[int, str, str]:
    """Run a git command, returning (returncode, stdout, stderr)."""
    env = _git_auth_env()
    proc = await asyncio.create_subprocess_exec(
        "git",
        *args,
        stdout=asyncio.subprocess.PIPE,
        stderr=asyncio.subprocess.PIPE,
        cwd=cwd,
        env=env,
    )
    stdout, stderr = await proc.communicate()
    stderr_str = stderr.decode()
    # Redact tokens from error output
    for tok_key in ("GITHUB_TOKEN", "GITLAB_TOKEN"):
        tok = os.getenv(tok_key, "")
        if tok:
            stderr_str = stderr_str.replace(tok, "***REDACTED***")
    return proc.returncode, stdout.decode(), stderr_str


async def _clone_marketplace_items(workspace_path: str) -> None:
    """Clone installed marketplace items to /workspace/marketplace/.

    Reads INSTALLED_ITEMS_JSON env var, groups items by source repo,
    and clones them using shallow/sparse checkout as appropriate.
    """
    raw = os.getenv("INSTALLED_ITEMS_JSON", "").strip()
    if not raw:
        return

    try:
        items = json.loads(raw)
    except json.JSONDecodeError as e:
        logger.warning(f"Failed to parse INSTALLED_ITEMS_JSON: {e}")
        return

    if not items:
        return

    marketplace_base = Path(workspace_path) / "marketplace"
    marketplace_base.mkdir(parents=True, exist_ok=True)

    # Group items by (sourceUrl, sourceBranch)
    groups: dict[tuple[str, str], list[dict]] = defaultdict(list)
    for item in items:
        source_url = item.get("sourceUrl", "").strip()
        source_branch = item.get("sourceBranch", "main").strip() or "main"
        if source_url:
            groups[(source_url, source_branch)].append(item)

    _TYPE_SUBDIRS = {
        "skill": "skills",
        "command": "commands",
        "agent": "agents",
    }

    async def _clone_source(
        source_url: str, branch: str, group_items: list[dict],
    ) -> None:
        has_workflow = any(
            it.get("itemType") == "workflow" for it in group_items
        )
        repo_name = source_url.split("/")[-1].removesuffix(".git")
        # Use repo_name + branch to avoid collisions between branches of same repo
        dir_name = f"{repo_name}-{branch}" if branch != "main" else repo_name
        clone_dir = marketplace_base / dir_name
        clone_url = source_url

        if clone_dir.exists():
            logger.info(f"Marketplace: {repo_name} already exists, skipping")
            return

        if has_workflow:
            rc, _, err = await _run_git(
                "clone", "--depth", "1", "-b", branch,
                clone_url, str(clone_dir),
            )
            if rc != 0:
                logger.warning(
                    f"Marketplace: failed to clone {repo_name}: {err}"
                )
                return
        else:
            rc, _, err = await _run_git(
                "clone", "--filter=blob:none", "--no-checkout",
                "--depth", "1", "-b", branch,
                clone_url, str(clone_dir),
            )
            if rc != 0:
                logger.warning(
                    f"Marketplace: failed to clone {repo_name}: {err}"
                )
                return

            file_paths = [
                it["filePath"]
                for it in group_items
                if it.get("filePath")
            ]
            if file_paths:
                rc, _, err = await _run_git(
                    "sparse-checkout", "set", *file_paths,
                    cwd=str(clone_dir),
                )
                if rc != 0:
                    logger.warning(
                        f"Marketplace: sparse-checkout failed for "
                        f"{repo_name}: {err}"
                    )

            rc, _, err = await _run_git(
                "checkout", cwd=str(clone_dir),
            )
            if rc != 0:
                logger.warning(
                    f"Marketplace: checkout failed for {repo_name}: {err}"
                )

        # Build .claude/ directory structure with symlinks
        claude_dir = clone_dir / ".claude"
        for item in group_items:
            item_type = item.get("itemType", "")
            item_id = item.get("itemId", "")
            file_path = item.get("filePath", "")

            if item_type == "workflow" or not item_id or not file_path:
                continue

            subdir_name = _TYPE_SUBDIRS.get(item_type)
            if not subdir_name:
                continue

            source_file = clone_dir / file_path
            if not source_file.exists():
                logger.warning(
                    f"Marketplace: source file not found: {source_file}"
                )
                continue

            if item_type == "skill":
                target_dir = claude_dir / subdir_name / item_id
                target_dir.mkdir(parents=True, exist_ok=True)
                target = target_dir / "SKILL.md"
            elif item_type == "command":
                target_parent = claude_dir / subdir_name
                target_parent.mkdir(parents=True, exist_ok=True)
                target = target_parent / f"{item_id}.md"
            elif item_type == "agent":
                target_parent = claude_dir / subdir_name
                target_parent.mkdir(parents=True, exist_ok=True)
                target = target_parent / f"{item_id}.md"
            else:
                continue

            if not target.exists():
                try:
                    target.symlink_to(source_file.resolve())
                except OSError as e:
                    logger.warning(
                        f"Marketplace: symlink failed for {item_id}: {e}"
                    )

        logger.info(
            f"Marketplace: {repo_name} ready at {clone_dir} "
            f"({len(group_items)} items)"
        )

    # Clone all sources in parallel
    tasks = []
    for (source_url, branch), group_items in groups.items():
        tasks.append(_clone_source(source_url, branch, group_items))

    results = await asyncio.gather(*tasks, return_exceptions=True)
    for i, result in enumerate(results):
        if isinstance(result, Exception):
            source_url = list(groups.keys())[i][0]
            logger.warning(
                f"Marketplace: failed to process {source_url}: {result}"
            )


class ClaudeBridge(PlatformBridge):
    """Bridge between the Ambient platform and the Claude Agent SDK.

    Handles lazy platform initialisation on first ``run()`` call, builds
    and caches the ``ClaudeAgentAdapter``, manages persistent
    ``SessionWorker`` instances, and wraps the event stream with
    Langfuse tracing.
    """

    def __init__(self) -> None:
        super().__init__()
        self._adapter: ClaudeAgentAdapter | None = None
        self._session_manager: SessionManager | None = None
        self._obs: Any = None

        # Platform state (populated by _setup_platform)
        self._first_run: bool = True
        self._configured_model: str = ""
        self._cwd_path: str = ""
        self._add_dirs: list[str] = []
        self._mcp_servers: dict = {}
        self._allowed_tools: list[str] = []
        self._system_prompt: dict = {}
        self._stderr_lines: list[str] = []
        # Preserved session IDs across adapter rebuilds (e.g. repo additions)
        self._saved_session_ids: dict[str, str] = {}
        # Per-thread halt tracking to avoid race conditions on shared adapter
        self._halted_by_thread: dict[str, bool] = {}

    # ------------------------------------------------------------------
    # PlatformBridge interface
    # ------------------------------------------------------------------

    def capabilities(self) -> FrameworkCapabilities:
        has_tracing = (
            self._obs is not None
            and hasattr(self._obs, "langfuse_client")
            and self._obs.langfuse_client is not None
        )
        return FrameworkCapabilities(
            framework="claude-agent-sdk",
            agent_features=[
                "agentic_chat",
                "backend_tool_rendering",
                "shared_state",
                "human_in_the_loop",
                "thinking",
            ],
            file_system=True,
            mcp=True,
            tracing="langfuse" if has_tracing else None,
            session_persistence=True,
        )

    async def _initialize_run(
        self,
        thread_id: str,
        current_user_id: str,
        current_user_name: str,
        caller_token: str,
    ) -> None:
        """Prepare the runtime for a new run.

        Sets user context, refreshes credentials, and restarts the Claude
        client if the user changed (so MCP servers pick up new creds).
        """
        from ambient_runner.platform.auth import (
            clear_runtime_credentials,
            populate_mcp_server_credentials,
            populate_runtime_credentials,
        )

        prev_user = self._context.current_user_id if self._context else ""
        if self._context:
            self._context.set_current_user(current_user_id, current_user_name, caller_token)

        await self._ensure_ready()

        # Fresh credentials for this user on every run
        clear_runtime_credentials()
        await populate_runtime_credentials(self._context)
        await populate_mcp_server_credentials(self._context)
        self._last_creds_refresh = time.monotonic()

        # If the caller changed, destroy the worker and rebuild MCP servers +
        # adapter so the new ClaudeSDKClient gets fresh mcp_servers config.
        # The session ID is preserved — --resume works because each SDK client
        # is a new CLI subprocess that spawns fresh MCP servers from os.environ.
        user_changed = current_user_id != prev_user
        if user_changed and self._session_manager.get_existing(thread_id):
            logger.info(
                f"User changed for thread={thread_id}, "
                "rebuilding MCP servers and adapter with new credentials"
            )
            await self._session_manager.destroy(thread_id)
            self._rebuild_mcp_servers()
            # Force adapter rebuild so ClaudeAgentOptions uses new mcp_servers
            self._adapter = None

        self._ensure_adapter()

    async def run(
        self,
        input_data: RunAgentInput,
        current_user_id: str = "",
        current_user_name: str = "",
        caller_token: str = "",
    ) -> AsyncIterator[BaseEvent]:
        """Full run lifecycle: initialize → session worker → tracing."""
        thread_id = input_data.thread_id or (self._context.session_id if self._context else "")

        await self._initialize_run(thread_id, current_user_id, current_user_name, caller_token)

        from ag_ui_claude_sdk.utils import process_messages

        user_msg, _ = process_messages(input_data)

        api_key = os.getenv("ANTHROPIC_API_KEY", "")
        saved_session_id = self._saved_session_ids.pop(
            thread_id, None
        ) or self._session_manager.get_session_id(thread_id)
        sdk_options = self._adapter.build_options(
            input_data, thread_id=thread_id, resume_from=saved_session_id
        )
        worker = await self._session_manager.get_or_create(
            thread_id, sdk_options, api_key
        )

        # 5. Run adapter with message stream, wrapped in tracing
        session_label = self._session_manager.get_session_id(thread_id) or thread_id
        async with self._session_manager.get_lock(thread_id):
            try:
                message_stream = worker.query(user_msg, session_id=session_label)

                from ambient_runner.middleware import tracing_middleware

                wrapped_stream = tracing_middleware(
                    self._adapter.run(input_data, message_stream=message_stream),
                    obs=self._obs,
                    model=self._configured_model,
                    prompt=user_msg,
                )

                async for event in wrapped_stream:
                    yield event

                # Persist session ID after turn completes (for --resume on pod restart)
                if worker.session_id:
                    self._session_manager._session_ids[thread_id] = worker.session_id
                    self._session_manager._persist_session_ids()

                # Capture halt state for this thread to avoid race conditions
                # with concurrent runs modifying the shared adapter's halted flag
                self._halted_by_thread[thread_id] = self._adapter.halted

                # If the adapter halted (frontend tool or built-in HITL tool like
                # AskUserQuestion), interrupt the worker to prevent the SDK from
                # auto-approving the tool call with a placeholder result.
                if self._halted_by_thread.get(thread_id, False):
                    logger.info(
                        f"Adapter halted for thread={thread_id}, "
                        "interrupting worker to await user input"
                    )
                    await worker.interrupt()
                    # Clear the halt flag for this thread
                    self._halted_by_thread.pop(thread_id, None)
            finally:
                # Clear caller token immediately — never persist between turns.
                if self._context:
                    self._context.caller_token = ""

                # Clear credentials after turn completes (shared session security).
                # In finally to ensure cleanup even on errors/cancellation.
                if (self._context.get_env("KEEP_CREDENTIALS_PERSISTENT") or "").lower() != "true":
                    from ambient_runner.platform.auth import clear_runtime_credentials

                    clear_runtime_credentials()

        self._first_run = False

    async def interrupt(self, thread_id: Optional[str] = None) -> None:
        """Interrupt the running session for a given thread."""
        if not self._session_manager:
            raise RuntimeError("No active session manager")

        tid = thread_id or (self._context.session_id if self._context else None)
        if not tid:
            raise RuntimeError("No thread_id available")

        worker = self._session_manager.get_existing(tid)
        if not worker:
            raise RuntimeError(f"No active session for thread {tid}")

        logger.info(f"Interrupt request for thread={tid}")
        await worker.interrupt()

        # Record interrupt in observability metrics
        if self._obs:
            self._obs.record_interrupt()

    # ------------------------------------------------------------------
    # Lifecycle methods
    # ------------------------------------------------------------------

    async def shutdown(self) -> None:
        """Graceful shutdown: persist sessions, finalise tracing."""
        if self._session_manager:
            await self._session_manager.shutdown()
        if self._obs:
            await self._obs.finalize()
        logger.info("ClaudeBridge: shutdown complete")

    def mark_dirty(self) -> None:
        """Signal adapter rebuild on next run (repo/workflow change).

        Destroys existing session workers so the new MCP server
        configuration (e.g. updated correction tool targets) is applied
        to the CLI process on the next run.  Conversation state is
        preserved via the CLI's ``--resume`` mechanism.
        """
        self._ready = False
        self._first_run = True
        self._adapter = None
        self._halted_by_thread.clear()
        if self._session_manager:
            # Preserve session IDs so --resume works after adapter rebuild.
            # Must be captured synchronously before the async shutdown task runs.
            self._saved_session_ids.update(self._session_manager.get_all_session_ids())
            manager = self._session_manager
            self._session_manager = None
            _async_safe_manager_shutdown(manager)
        logger.info("ClaudeBridge: marked dirty — will reinitialise on next run")

    def get_error_context(self) -> str:
        """Return recent Claude CLI stderr lines for error reporting."""
        if self._stderr_lines:
            recent = self._stderr_lines[-10:]
            return "Claude CLI stderr:\n" + "\n".join(recent)
        return ""

    async def get_mcp_status(self) -> dict:
        """Get MCP server status via an ephemeral SDK client."""
        if not self._context:
            return {
                "servers": [],
                "totalCount": 0,
                "message": "Context not initialized",
            }

        try:
            from claude_agent_sdk import ClaudeAgentOptions, ClaudeSDKClient

            from ambient_runner.platform.config import load_mcp_config
            from ambient_runner.platform.workspace import resolve_workspace_paths

            cwd_path, _ = resolve_workspace_paths(self._context)
            mcp_servers = load_mcp_config(self._context, cwd_path) or {}

            options = ClaudeAgentOptions(
                cwd=cwd_path,
                permission_mode="acceptEdits",
                mcp_servers=mcp_servers,
            )

            client = ClaudeSDKClient(options=options)
            try:
                logger.info("MCP Status: Connecting ephemeral SDK client...")
                await client.connect()

                sdk_status = await client.get_mcp_status()

                raw_servers = []
                if isinstance(sdk_status, dict):
                    raw_servers = sdk_status.get("mcpServers", [])
                elif isinstance(sdk_status, list):
                    raw_servers = sdk_status

                servers_list = []
                for srv in raw_servers:
                    if not isinstance(srv, dict):
                        continue
                    server_info = srv.get("serverInfo") or {}
                    raw_tools = srv.get("tools") or []
                    tools = [
                        {
                            "name": t.get("name", ""),
                            "annotations": {
                                k: v for k, v in (t.get("annotations") or {}).items()
                            },
                        }
                        for t in raw_tools
                        if isinstance(t, dict)
                    ]
                    servers_list.append(
                        {
                            "name": srv.get("name", ""),
                            "displayName": server_info.get("name", srv.get("name", "")),
                            "status": srv.get("status", "unknown"),
                            "version": server_info.get("version", ""),
                            "tools": tools,
                        }
                    )

                return {"servers": servers_list, "totalCount": len(servers_list)}
            finally:
                logger.info("MCP Status: Disconnecting ephemeral SDK client...")
                await client.disconnect()

        except Exception as e:
            logger.error(f"Failed to get MCP status: {e}", exc_info=True)
            return {"servers": [], "totalCount": 0, "error": str(e)}

    # ------------------------------------------------------------------
    # Properties
    # ------------------------------------------------------------------

    @property
    def context(self) -> RunnerContext | None:
        return self._context

    @property
    def configured_model(self) -> str:
        return self._configured_model

    @property
    def obs(self) -> Any:
        return self._obs

    @property
    def session_manager(self) -> SessionManager | None:
        return self._session_manager

    # ------------------------------------------------------------------
    # Private: platform setup (lazy, called on first run)
    # ------------------------------------------------------------------

    async def _setup_platform(self) -> None:
        """Full platform setup: auth, workspace, MCP, observability."""
        # Session manager
        if self._session_manager is None:
            state_dir = os.path.join(
                os.getenv("WORKSPACE_PATH", "/workspace"),
                os.getenv("RUNNER_STATE_DIR", ".claude"),
            )
            self._session_manager = SessionManager(state_dir=state_dir)

        # Claude-specific auth
        from ambient_runner.bridges.claude.auth import setup_sdk_authentication
        from ambient_runner.platform.auth import (
            populate_mcp_server_credentials,
            populate_runtime_credentials,
        )
        from ambient_runner.platform.workspace import (
            resolve_workspace_paths,
            validate_prerequisites,
        )

        await validate_prerequisites(self._context)
        _api_key, _use_vertex, configured_model = await setup_sdk_authentication(
            self._context
        )

        # Populate credentials before building system prompt (prompt checks env vars)
        await populate_runtime_credentials(self._context)
        await populate_mcp_server_credentials(self._context)
        self._last_creds_refresh = time.monotonic()

        # Clone installed marketplace items before resolving workspace paths
        await _clone_marketplace_items(
            os.getenv("WORKSPACE_PATH", "/workspace")
        )

        # Workspace paths
        cwd_path, add_dirs = resolve_workspace_paths(self._context)
        if add_dirs:
            os.environ["CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD"] = "1"

        # Observability (shared helper, before MCP so rubric tool can access it)
        self._obs = await setup_bridge_observability(self._context, configured_model)

        # MCP servers
        from ambient_runner.bridges.claude.mcp import (
            build_allowed_tools,
            build_mcp_servers,
            log_auth_status,
        )

        mcp_servers = build_mcp_servers(self._context, cwd_path, self._obs)
        log_auth_status(mcp_servers)
        allowed_tools = build_allowed_tools(mcp_servers)

        # System prompt
        from ambient_runner.bridges.claude.prompts import build_sdk_system_prompt

        system_prompt = build_sdk_system_prompt(self._context.workspace_path, cwd_path)

        # Store results
        self._configured_model = configured_model
        self._cwd_path = cwd_path
        self._add_dirs = add_dirs
        self._mcp_servers = mcp_servers
        self._allowed_tools = allowed_tools
        self._system_prompt = system_prompt

    def _rebuild_mcp_servers(self) -> None:
        """Rebuild MCP server config with current env vars.

        Called when the user changes so .mcp.json env blocks (e.g.,
        ${JIRA_API_TOKEN}) are re-expanded with the new user's credentials.
        """
        from ambient_runner.bridges.claude.mcp import (
            build_allowed_tools,
            build_mcp_servers,
        )

        self._mcp_servers = build_mcp_servers(
            self._context, self._cwd_path, self._obs
        )
        self._allowed_tools = build_allowed_tools(self._mcp_servers)
        logger.info("Rebuilt MCP servers with updated credentials")

    # ------------------------------------------------------------------
    # Private: adapter lifecycle
    # ------------------------------------------------------------------

    def _ensure_adapter(self) -> None:
        """Build or reuse the ClaudeAgentAdapter."""
        if self._adapter is not None:
            return

        self._stderr_lines.clear()

        def _stderr_handler(line: str) -> None:
            stripped = line.rstrip()
            logger.warning(f"[SDK stderr] {stripped}")
            self._stderr_lines.append(stripped)
            if len(self._stderr_lines) > _MAX_STDERR_LINES:
                self._stderr_lines.pop(0)

        options: dict[str, Any] = {
            "cwd": self._cwd_path,
            "permission_mode": "acceptEdits",
            "allowed_tools": self._allowed_tools,
            "mcp_servers": self._mcp_servers,
            "setting_sources": ["project"],
            "system_prompt": self._system_prompt,
            "include_partial_messages": True,
            "stderr": _stderr_handler,
        }

        if self._add_dirs:
            options["add_dirs"] = self._add_dirs
        if self._configured_model:
            options["model"] = self._configured_model

        adapter = ClaudeAgentAdapter(
            name="claude_code_runner",
            description="Ambient Code Platform Claude session",
            options=options,
        )
        # Attach stderr buffer so error handler can read it
        adapter._stderr_lines = self._stderr_lines  # type: ignore[attr-defined]
        self._adapter = adapter
        logger.info("Adapter built (persistent, will be reused across runs)")
