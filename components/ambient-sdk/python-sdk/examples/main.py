#!/usr/bin/env python3
"""Ambient Platform SDK - Python Example"""

import sys

from ambient_platform import (
    AmbientClient,
    Session,
    SessionPatch,
    ListOptions,
    APIError,
)
from ambient_platform.exceptions import AmbientAPIError


def main():
    print("Ambient Platform SDK - Python Example")
    print("======================================")

    try:
        client = AmbientClient.from_env(timeout=60.0)
    except ValueError as e:
        print(f"Configuration error: {e}")
        sys.exit(1)

    with client:
        data = Session.builder().name("example-session").prompt("Analyze the repository structure").build()
        created = client.sessions.create(data)
        print(f"Created session: {created.name} (id={created.id})")

        got = client.sessions.get(created.id)
        print(f"Got session: {got.name}")

        sessions = client.sessions.list(ListOptions().size(10))
        print(f"Found {len(sessions.items)} sessions (total: {sessions.total})")

        patch = SessionPatch().prompt("Updated prompt")
        updated = client.sessions.update(created.id, patch)
        print(f"Updated session prompt: {updated.prompt}")

        print("\nIterating all sessions:")
        count = 0
        for s in client.sessions.list_all(size=100):
            count += 1
            if count <= 3:
                print(f"  {count}. {s.name} ({s.id})")
        if count > 3:
            print(f"  ... and {count - 3} more")

        print("\nDone.")


if __name__ == "__main__":
    main()
