"""Anthropic Messages API clients for auto-analysis.

Two implementations with the same ``create_message`` interface:

* ``VertexAnthropicClient`` — calls Vertex AI rawPredict using Google
  Application Default Credentials (requires ``ANTHROPIC_VERTEX_PROJECT_ID``).
* ``AnthropicDirectClient`` — calls the Anthropic Messages API using
  ``ANTHROPIC_API_KEY`` (works in CI and local dev without GCP).
"""

import json
import logging
import os
import urllib.request
from typing import Any

logger = logging.getLogger(__name__)


class VertexAnthropicClient:
    """Thin client for the Vertex AI Anthropic Messages API."""

    def __init__(
        self,
        project: str | None = None,
        region: str | None = None,
        model: str | None = None,
    ):
        self.project = project or os.getenv("ANTHROPIC_VERTEX_PROJECT_ID", "")
        self.region = region or os.getenv("CLOUD_ML_REGION", "us-east5")
        self.model = model or os.getenv(
            "LLM_MODEL_VERTEX_ID", "claude-sonnet-4-5@20250929"
        )
        if not self.project:
            raise ValueError("ANTHROPIC_VERTEX_PROJECT_ID is required")

        self._token: str | None = None

    def _refresh_token(self) -> str:
        import google.auth
        import google.auth.transport.requests

        creds, _ = google.auth.default(
            scopes=["https://www.googleapis.com/auth/cloud-platform"]
        )
        request = google.auth.transport.requests.Request()
        creds.refresh(request)
        self._token = creds.token
        return self._token

    def _base_url(self) -> str:
        return (
            f"https://{self.region}-aiplatform.googleapis.com/v1/"
            f"projects/{self.project}/locations/{self.region}/"
            f"publishers/anthropic/models/{self.model}"
        )

    def create_message(
        self,
        messages: list[dict[str, Any]],
        system: str = "",
        tools: list[dict[str, Any]] | None = None,
        max_tokens: int = 4096,
    ) -> dict[str, Any]:
        """Non-streaming Messages API call.  Returns the full response."""
        token = self._refresh_token()
        url = f"{self._base_url()}:rawPredict"

        payload: dict[str, Any] = {
            "anthropic_version": "vertex-2023-10-16",
            "max_tokens": max_tokens,
            "messages": messages,
        }
        if system:
            payload["system"] = system
        if tools:
            payload["tools"] = tools

        req = urllib.request.Request(
            url,
            data=json.dumps(payload).encode(),
            headers={
                "Content-Type": "application/json",
                "Authorization": f"Bearer {token}",
            },
            method="POST",
        )
        with urllib.request.urlopen(req, timeout=120) as resp:
            return json.loads(resp.read())


class AnthropicDirectClient:
    """Thin client for the Anthropic Messages API using ANTHROPIC_API_KEY."""

    def __init__(self, api_key: str | None = None, model: str | None = None):
        self.api_key = api_key or os.getenv("ANTHROPIC_API_KEY", "")
        self.model = model or os.getenv("LLM_MODEL", "claude-sonnet-4-5-20250514")
        if not self.api_key:
            raise ValueError("ANTHROPIC_API_KEY is required")

    def create_message(
        self,
        messages: list[dict[str, Any]],
        system: str = "",
        tools: list[dict[str, Any]] | None = None,
        max_tokens: int = 4096,
    ) -> dict[str, Any]:
        """Non-streaming Messages API call. Returns the full response dict."""
        url = "https://api.anthropic.com/v1/messages"
        payload: dict[str, Any] = {
            "model": self.model,
            "max_tokens": max_tokens,
            "messages": messages,
        }
        if system:
            payload["system"] = system
        if tools:
            payload["tools"] = tools

        req = urllib.request.Request(
            url,
            data=json.dumps(payload).encode(),
            headers={
                "Content-Type": "application/json",
                "x-api-key": self.api_key,
                "anthropic-version": "2023-06-01",
            },
            method="POST",
        )
        with urllib.request.urlopen(req, timeout=120) as resp:
            return json.loads(resp.read())
