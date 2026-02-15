"""
Platform modules — framework-agnostic, used by any bridge.

Provides the platform substrate: workspace management, authentication,
configuration, observability, security, prompts, and utilities.
"""

from ambient_runner.platform.context import RunnerContext
from ambient_runner.platform.config import (
    get_repos_config,
    load_ambient_config,
    load_mcp_config,
)
from ambient_runner.platform.workspace import (
    PrerequisiteError,
    resolve_workspace_paths,
    setup_multi_repo_paths,
    setup_workflow_paths,
    validate_prerequisites,
)
from ambient_runner.platform.auth import (
    sanitize_user_context,
    populate_runtime_credentials,
    fetch_github_token,
    fetch_google_credentials,
    fetch_jira_credentials,
    fetch_gitlab_token,
    fetch_token_for_url,
)

__all__ = [
    "RunnerContext",
    "PrerequisiteError",
    "get_repos_config",
    "load_ambient_config",
    "load_mcp_config",
    "resolve_workspace_paths",
    "setup_multi_repo_paths",
    "setup_workflow_paths",
    "validate_prerequisites",
    "sanitize_user_context",
    "populate_runtime_credentials",
    "fetch_github_token",
    "fetch_google_credentials",
    "fetch_jira_credentials",
    "fetch_gitlab_token",
    "fetch_token_for_url",
]
