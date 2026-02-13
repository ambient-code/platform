"""
Ambient Platform Python SDK

Simple HTTP client for the Ambient Code Platform - Create and manage AI agent sessions without Kubernetes complexity.
"""

from .client import AmbientClient
from .types import (
    SessionResponse,
    SessionListResponse,
    CreateSessionRequest,
    CreateSessionResponse,
    RepoHTTP,
    ErrorResponse,
    StatusPending,
    StatusRunning,
    StatusCompleted,
    StatusFailed,
)
from .exceptions import (
    AmbientAPIError,
    AmbientConnectionError,
    SessionNotFoundError,
    AuthenticationError,
)

__version__ = "1.0.0"
__author__ = "Ambient Code Platform"
__email__ = "hello@ambient-code.io"

__all__ = [
    # Client
    "AmbientClient",
    
    # Types
    "SessionResponse",
    "SessionListResponse", 
    "CreateSessionRequest",
    "CreateSessionResponse",
    "RepoHTTP",
    "ErrorResponse",
    
    # Status constants
    "StatusPending",
    "StatusRunning", 
    "StatusCompleted",
    "StatusFailed",
    
    # Exceptions
    "AmbientAPIError",
    "AmbientConnectionError", 
    "SessionNotFoundError",
    "AuthenticationError",
]