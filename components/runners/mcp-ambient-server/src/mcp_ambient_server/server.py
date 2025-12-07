"""MCP server for Ambient Code Platform API.

This server provides read-only access to the ACP backend API through MCP tools.
It runs as a sidecar in Claude runner pods and uses the user's bearer token for authentication.
"""

import logging
import json
import sys
from mcp.server.fastmcp import FastMCP
from mcp_ambient_server.client import APIClient


# Configure logging to stderr (stdout is used for MCP protocol)
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
    stream=sys.stderr,
)

logger = logging.getLogger(__name__)

# Initialize FastMCP server
mcp = FastMCP("ambient-code")

# Initialize API client (will be set in main())
api_client: APIClient = None


def format_json(data: dict) -> str:
    """Format dict as pretty JSON string.

    Args:
        data: Dictionary to format

    Returns:
        Formatted JSON string
    """
    return json.dumps(data, indent=2)


# ============================================================================
# Project Management Tools
# ============================================================================


@mcp.tool()
async def create_project(
    name: str, display_name: str = "", description: str = ""
) -> str:
    """Create a new project in the Ambient Code Platform.

    Args:
        name: Project name (required, must be valid Kubernetes namespace name)
        display_name: Display name (optional, used on OpenShift)
        description: Project description (optional)

    Returns:
        JSON object with created project details (name, namespace, status)
    """
    try:
        project = await api_client.create_project(name, display_name, description)
        return format_json(project)
    except Exception as e:
        logger.error(f"Failed to create project {name}: {e}")
        return f"Error creating project: {str(e)}"


@mcp.tool()
async def list_projects() -> str:
    """List all projects accessible by the user.

    Returns:
        JSON array of project objects with name, displayName, description, status
    """
    try:
        projects = await api_client.list_projects()
        return format_json({"projects": projects, "count": len(projects)})
    except Exception as e:
        logger.error(f"Failed to list projects: {e}")
        return f"Error listing projects: {str(e)}"


@mcp.tool()
async def get_project(project_name: str) -> str:
    """Get detailed information about a specific project.

    Args:
        project_name: Name of the project to retrieve

    Returns:
        JSON object with project details
    """
    try:
        project = await api_client.get_project(project_name)
        return format_json(project)
    except Exception as e:
        logger.error(f"Failed to get project {project_name}: {e}")
        return f"Error getting project: {str(e)}"


@mcp.tool()
async def check_project_access(project_name: str) -> str:
    """Check user's access permissions for a project.

    Args:
        project_name: Name of the project to check access for

    Returns:
        JSON object with access information and permissions
    """
    try:
        access_info = await api_client.check_project_access(project_name)
        return format_json(access_info)
    except Exception as e:
        logger.error(f"Failed to check access for project {project_name}: {e}")
        return f"Error checking project access: {str(e)}"


# ============================================================================
# Session Browsing Tools
# ============================================================================


@mcp.tool()
async def list_sessions(project_name: str) -> str:
    """List all agentic sessions in a project.

    Args:
        project_name: Name of the project

    Returns:
        JSON array of session objects with name, status, phase, timestamps
    """
    try:
        sessions = await api_client.list_sessions(project_name)
        return format_json({"sessions": sessions, "count": len(sessions)})
    except Exception as e:
        logger.error(f"Failed to list sessions in project {project_name}: {e}")
        return f"Error listing sessions: {str(e)}"


@mcp.tool()
async def get_session(project_name: str, session_name: str) -> str:
    """Get detailed information about a specific agentic session.

    Args:
        project_name: Name of the project
        session_name: Name of the session

    Returns:
        JSON object with session details including spec, status, repos, workflow
    """
    try:
        session = await api_client.get_session(project_name, session_name)
        return format_json(session)
    except Exception as e:
        logger.error(f"Failed to get session {project_name}/{session_name}: {e}")
        return f"Error getting session: {str(e)}"


@mcp.tool()
async def get_session_k8s_resources(project_name: str, session_name: str) -> str:
    """Get Kubernetes resources associated with a session (pods, jobs, services).

    Args:
        project_name: Name of the project
        session_name: Name of the session

    Returns:
        JSON object with Kubernetes resource information
    """
    try:
        resources = await api_client.get_session_k8s_resources(
            project_name, session_name
        )
        return format_json(resources)
    except Exception as e:
        logger.error(
            f"Failed to get k8s resources for {project_name}/{session_name}: {e}"
        )
        return f"Error getting Kubernetes resources: {str(e)}"


@mcp.tool()
async def list_session_workspace(project_name: str, session_name: str) -> str:
    """List all files in a session's workspace.

    Args:
        project_name: Name of the project
        session_name: Name of the session

    Returns:
        JSON array of file objects with name, size, type, modified time
    """
    try:
        files = await api_client.list_session_workspace(project_name, session_name)
        return format_json({"files": files, "count": len(files)})
    except Exception as e:
        logger.error(f"Failed to list workspace for {project_name}/{session_name}: {e}")
        return f"Error listing workspace: {str(e)}"


# ============================================================================
# Workspace File Access Tools
# ============================================================================


@mcp.tool()
async def get_workspace_file(project_name: str, session_name: str, path: str) -> str:
    """Get contents of a specific file from a session's workspace.

    Args:
        project_name: Name of the project
        session_name: Name of the session
        path: Path to the file (relative to workspace root, no '..' allowed)

    Returns:
        File contents or JSON object with file metadata and content
    """
    try:
        file_data = await api_client.get_workspace_file(
            project_name, session_name, path
        )
        return format_json(file_data)
    except ValueError as e:
        return f"Invalid path: {str(e)}"
    except Exception as e:
        logger.error(
            f"Failed to get workspace file {project_name}/{session_name}/{path}: {e}"
        )
        return f"Error getting workspace file: {str(e)}"


# ============================================================================
# Workflow & Cluster Info Tools
# ============================================================================


@mcp.tool()
async def list_ootb_workflows() -> str:
    """List all out-of-the-box (OOTB) workflows available in the platform.

    Returns:
        JSON array of workflow objects with name, description, agents
    """
    try:
        workflows = await api_client.list_ootb_workflows()
        return format_json({"workflows": workflows, "count": len(workflows)})
    except Exception as e:
        logger.error(f"Failed to list OOTB workflows: {e}")
        return f"Error listing workflows: {str(e)}"


@mcp.tool()
async def get_workflow_metadata(project_name: str, session_name: str) -> str:
    """Get metadata about the workflow assigned to a session.

    Args:
        project_name: Name of the project
        session_name: Name of the session

    Returns:
        JSON object with workflow metadata including name, agents, status
    """
    try:
        metadata = await api_client.get_workflow_metadata(project_name, session_name)
        return format_json(metadata)
    except Exception as e:
        logger.error(
            f"Failed to get workflow metadata for {project_name}/{session_name}: {e}"
        )
        return f"Error getting workflow metadata: {str(e)}"


@mcp.tool()
async def get_cluster_info() -> str:
    """Get information about the Kubernetes/OpenShift cluster.

    Returns:
        JSON object with cluster details like version, platform, capabilities
    """
    try:
        cluster_info = await api_client.get_cluster_info()
        return format_json(cluster_info)
    except Exception as e:
        logger.error(f"Failed to get cluster info: {e}")
        return f"Error getting cluster info: {str(e)}"


@mcp.tool()
async def get_health() -> str:
    """Get health status of the backend API.

    Returns:
        JSON object with health status information
    """
    try:
        health = await api_client.get_health()
        return format_json(health)
    except Exception as e:
        logger.error(f"Failed to get health status: {e}")
        return f"Error getting health status: {str(e)}"


# ============================================================================
# Main Entry Point
# ============================================================================


def main():
    """Initialize the MCP server and start listening on stdio."""
    global api_client

    try:
        # Initialize API client
        api_client = APIClient()
        logger.info("API client initialized successfully")

        # Run MCP server on stdio
        logger.info("Starting MCP server for Ambient Code Platform...")
        mcp.run(transport="stdio")

    except ValueError as e:
        logger.error(f"Failed to initialize API client: {e}")
        sys.exit(1)
    except Exception as e:
        logger.error(f"Unexpected error: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
