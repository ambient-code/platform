"""
Proxy authentication middleware for the Ambient Runner.

Rejects direct AG-UI connections that bypass the backend proxy by validating
the Authorization header against the RUNNER_PROXY_SECRET environment variable.

The backend proxy sets ``Authorization: Bearer {RUNNER_PROXY_SECRET}`` on
every request it forwards to the runner. The runner then validates this header
to ensure the request came through the authorised backend proxy rather than
being sent directly by an arbitrary cluster process.

When RUNNER_PROXY_SECRET is not set (local development, unit tests), the
middleware is disabled and all requests pass through unmodified.

Protected methods: POST, PUT, PATCH, DELETE (write operations).
Excluded paths: /health (liveness probes must not require auth).
"""

import logging
import os

from fastapi import Request
from fastapi.responses import JSONResponse
from starlette.middleware.base import BaseHTTPMiddleware

logger = logging.getLogger(__name__)

# Paths that are always public regardless of auth configuration.
_PUBLIC_PATHS = {"/health"}


class ProxyAuthMiddleware(BaseHTTPMiddleware):
    """Reject write requests that do not carry a valid proxy secret.

    Only active when the ``RUNNER_PROXY_SECRET`` environment variable is set.
    When the variable is absent the middleware is a no-op so that local
    development and tests work without configuration changes.
    """

    async def dispatch(self, request: Request, call_next):
        proxy_secret = os.getenv("RUNNER_PROXY_SECRET", "").strip()

        # Passthrough when no secret is configured (dev / test mode).
        if not proxy_secret:
            return await call_next(request)

        # Health endpoint is always public (required for K8s liveness probes).
        if request.url.path in _PUBLIC_PATHS:
            return await call_next(request)

        # Only validate write operations — GET/HEAD/OPTIONS are read-only.
        if request.method not in ("POST", "PUT", "PATCH", "DELETE"):
            return await call_next(request)

        auth_header = request.headers.get("Authorization", "")
        expected = f"Bearer {proxy_secret}"

        if auth_header != expected:
            logger.warning(
                "Rejected direct runner connection from %s %s "
                "(missing or invalid Authorization header)",
                request.method,
                request.url.path,
            )
            return JSONResponse(
                status_code=403,
                content={
                    "detail": (
                        "Direct runner connections are not permitted. "
                        "Route requests through the platform backend proxy."
                    )
                },
            )

        return await call_next(request)
