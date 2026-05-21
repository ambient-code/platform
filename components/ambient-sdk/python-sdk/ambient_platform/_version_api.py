from __future__ import annotations

from dataclasses import dataclass
from typing import Optional

import httpx


@dataclass(frozen=True)
class ServerVersion:
    version: str = ""
    build_time: str = ""
    git_tag: str = ""

    @classmethod
    def from_dict(cls, data: dict) -> ServerVersion:
        return cls(
            version=data.get("version", ""),
            build_time=data.get("build_time", ""),
            git_tag=data.get("git_tag", ""),
        )


def fetch_server_version(
    base_url: str,
    *,
    timeout: float = 10.0,
    verify_ssl: bool = True,
) -> ServerVersion:
    url = base_url.rstrip("/") + "/api/ambient/v1/version"
    with httpx.Client(timeout=timeout, verify=verify_ssl) as client:
        response = client.get(url, headers={"Accept": "application/json"})
        response.raise_for_status()
        return ServerVersion.from_dict(response.json())
