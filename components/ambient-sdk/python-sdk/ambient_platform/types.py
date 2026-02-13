"""
HTTP API types for the Ambient Platform Public API.
"""

from typing import List, Optional
from dataclasses import dataclass

# Session status constants
StatusPending = "pending"
StatusRunning = "running"
StatusCompleted = "completed"
StatusFailed = "failed"


@dataclass
class RepoHTTP:
    """Repository configuration for HTTP API."""
    url: str
    branch: Optional[str] = None

    def to_dict(self) -> dict:
        result = {"url": self.url}
        if self.branch:
            result["branch"] = self.branch
        return result


@dataclass
class CreateSessionRequest:
    """Request body for creating a session."""
    task: str
    model: Optional[str] = None
    repos: Optional[List[RepoHTTP]] = None

    def to_dict(self) -> dict:
        result = {"task": self.task}
        if self.model:
            result["model"] = self.model
        if self.repos:
            result["repos"] = [repo.to_dict() for repo in self.repos]
        return result


@dataclass
class CreateSessionResponse:
    """Response from creating a session."""
    id: str
    message: str

    @classmethod
    def from_dict(cls, data: dict) -> "CreateSessionResponse":
        return cls(
            id=data["id"],
            message=data["message"]
        )


@dataclass
class SessionResponse:
    """Simplified session response from the public API."""
    id: str
    status: str  # "pending", "running", "completed", "failed"
    task: str
    model: Optional[str] = None
    created_at: Optional[str] = None
    completed_at: Optional[str] = None
    result: Optional[str] = None
    error: Optional[str] = None

    @classmethod
    def from_dict(cls, data: dict) -> "SessionResponse":
        return cls(
            id=data["id"],
            status=data["status"],
            task=data["task"],
            model=data.get("model"),
            created_at=data.get("createdAt"),
            completed_at=data.get("completedAt"),
            result=data.get("result"),
            error=data.get("error")
        )


@dataclass
class SessionListResponse:
    """Response for listing sessions."""
    items: List[SessionResponse]
    total: int

    @classmethod
    def from_dict(cls, data: dict) -> "SessionListResponse":
        return cls(
            items=[SessionResponse.from_dict(item) for item in data["items"]],
            total=data["total"]
        )


@dataclass
class ErrorResponse:
    """Standard error response."""
    error: str
    message: Optional[str] = None

    @classmethod
    def from_dict(cls, data: dict) -> "ErrorResponse":
        return cls(
            error=data["error"],
            message=data.get("message")
        )