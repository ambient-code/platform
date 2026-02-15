"""
HTTP API types for the Ambient Platform Public API.
"""

from typing import List, Optional
from dataclasses import dataclass
from urllib.parse import urlparse

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
    
    def validate(self) -> None:
        """Validate repository configuration."""
        if not self.url or not self.url.strip():
            raise ValueError("Repository URL cannot be empty")
        
        # Parse and validate URL
        try:
            parsed = urlparse(self.url)
        except Exception as e:
            raise ValueError(f"Invalid URL format: {e}")
        
        # Check for supported schemes
        if parsed.scheme not in ("https", "http"):
            raise ValueError(f"Unsupported URL scheme: {parsed.scheme} (must be http or https)")
        
        # Check for common invalid URLs
        if "example.com" in self.url or "localhost" in self.url:
            raise ValueError("URL appears to be a placeholder or localhost")


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
    
    def validate(self) -> None:
        """Validate the session creation request."""
        if not self.task or not self.task.strip():
            raise ValueError("Task cannot be empty")
        
        if len(self.task) > 10000:
            raise ValueError("Task exceeds maximum length of 10,000 characters")
        
        # Validate model if provided
        if self.model and not self._is_valid_model(self.model):
            raise ValueError(f"Invalid model: {self.model}")
        
        # Validate repositories
        if self.repos:
            for i, repo in enumerate(self.repos):
                try:
                    repo.validate()
                except ValueError as e:
                    raise ValueError(f"Repository {i}: {e}")
    
    def _is_valid_model(self, model: str) -> bool:
        """Check if the model name is valid."""
        valid_models = [
            "claude-3.5-sonnet",
            "claude-3.5-haiku", 
            "claude-3-opus",
            "claude-3-sonnet",
            "claude-3-haiku",
        ]
        return model in valid_models


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