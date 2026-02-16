"""
Exceptions for the Ambient Platform Python SDK.
"""


class AmbientAPIError(Exception):
    """Base exception for Ambient Platform API errors."""
    pass


class AmbientConnectionError(AmbientAPIError):
    """Raised when connection to the API fails."""
    pass


class SessionNotFoundError(AmbientAPIError):
    """Raised when a session is not found or not accessible."""
    pass


class AuthenticationError(AmbientAPIError):
    """Raised when authentication fails."""
    pass