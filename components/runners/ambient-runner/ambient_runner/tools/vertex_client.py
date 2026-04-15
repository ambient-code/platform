"""Vertex AI Anthropic Messages API client with streaming and tool use.

Calls the Vertex AI rawPredict / streamRawPredict endpoint directly
using Google Application Default Credentials.  No Anthropic SDK or
Claude CLI required.
"""

import json
import logging
import os
import urllib.request
from typing import Any, Iterator

logger = logging.getLogger(__name__)


class VertexAnthropicClient:
    """Thin client for the Vertex AI Anthropic Messages API."""

    def __init__(
        self,
        project: str | None = None,
        region: str | None = None,
        model: str | None = None,
    ):
        self.project = project or os.getenv(
            "ANTHROPIC_VERTEX_PROJECT_ID", ""
        )
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

    def stream_message(
        self,
        messages: list[dict[str, Any]],
        system: str = "",
        tools: list[dict[str, Any]] | None = None,
        max_tokens: int = 4096,
    ) -> Iterator[dict[str, Any]]:
        """Streaming Messages API call.  Yields SSE event dicts."""
        token = self._refresh_token()
        url = f"{self._base_url()}:streamRawPredict"

        payload: dict[str, Any] = {
            "anthropic_version": "vertex-2023-10-16",
            "max_tokens": max_tokens,
            "stream": True,
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
            for raw_line in resp:
                line = raw_line.decode().strip()
                if line.startswith("data: "):
                    try:
                        yield json.loads(line[6:])
                    except json.JSONDecodeError:
                        continue
