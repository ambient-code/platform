"""
Ambient Platform Python SDK

Simple HTTP client for the Ambient Code Platform - Create and manage AI agent sessions without Kubernetes complexity.
"""

from .client import AmbientClient
from ._base import APIError, ListMeta, ListOptions, ObjectReference
from .session import Session, SessionBuilder, SessionList, SessionPatch, SessionStatusPatch
from .agent import Agent, AgentBuilder, AgentList, AgentPatch
from .task import Task, TaskBuilder, TaskList, TaskPatch
from .skill import Skill, SkillBuilder, SkillList, SkillPatch
from .workflow import Workflow, WorkflowBuilder, WorkflowList, WorkflowPatch
from .user import User, UserBuilder, UserList, UserPatch
from .workflow_skill import WorkflowSkill, WorkflowSkillBuilder, WorkflowSkillList, WorkflowSkillPatch
from .workflow_task import WorkflowTask, WorkflowTaskBuilder, WorkflowTaskList, WorkflowTaskPatch
from .project import Project, ProjectBuilder, ProjectList, ProjectPatch
from .project_settings import ProjectSettings, ProjectSettingsBuilder, ProjectSettingsList, ProjectSettingsPatch
from .permission import Permission, PermissionBuilder, PermissionList, PermissionPatch
from .repository_ref import RepositoryRef, RepositoryRefBuilder, RepositoryRefList, RepositoryRefPatch
from .project_key import ProjectKey, ProjectKeyBuilder, ProjectKeyList, ProjectKeyPatch
from .exceptions import (
    AmbientAPIError,
    AmbientConnectionError,
    SessionNotFoundError,
    AuthenticationError,
)

__version__ = "2.0.0"
__author__ = "Ambient Code Platform"
__email__ = "hello@ambient-code.io"

__all__ = [
    "AmbientClient",
    "APIError",
    "ListMeta",
    "ListOptions",
    "ObjectReference",
    "Session",
    "SessionBuilder",
    "SessionList",
    "SessionPatch",
    "SessionStatusPatch",
    "Agent",
    "AgentBuilder",
    "AgentList",
    "AgentPatch",
    "Task",
    "TaskBuilder",
    "TaskList",
    "TaskPatch",
    "Skill",
    "SkillBuilder",
    "SkillList",
    "SkillPatch",
    "Workflow",
    "WorkflowBuilder",
    "WorkflowList",
    "WorkflowPatch",
    "User",
    "UserBuilder",
    "UserList",
    "UserPatch",
    "WorkflowSkill",
    "WorkflowSkillBuilder",
    "WorkflowSkillList",
    "WorkflowSkillPatch",
    "WorkflowTask",
    "WorkflowTaskBuilder",
    "WorkflowTaskList",
    "WorkflowTaskPatch",
    "Project",
    "ProjectBuilder",
    "ProjectList",
    "ProjectPatch",
    "ProjectSettings",
    "ProjectSettingsBuilder",
    "ProjectSettingsList",
    "ProjectSettingsPatch",
    "Permission",
    "PermissionBuilder",
    "PermissionList",
    "PermissionPatch",
    "RepositoryRef",
    "RepositoryRefBuilder",
    "RepositoryRefList",
    "RepositoryRefPatch",
    "ProjectKey",
    "ProjectKeyBuilder",
    "ProjectKeyList",
    "ProjectKeyPatch",
    "AmbientAPIError",
    "AmbientConnectionError",
    "SessionNotFoundError",
    "AuthenticationError",
]
