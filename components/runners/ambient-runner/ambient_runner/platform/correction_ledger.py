"""Per-session in-memory correction ledger.

Stores corrections logged during a session and renders them as a markdown
block for injection into the agent's context. Lives in the platform layer
so both Claude and Gemini bridges can use the same class without
cross-bridge imports.

The ledger is intentionally not persisted -- pod restart clears it.
Corrections are session-local context, not durable state.
"""

import logging
import os
from datetime import datetime, timezone
from typing import TypedDict

logger = logging.getLogger(__name__)

# Maximum individual entries rendered in the correction block.
MAX_RENDERED_ENTRIES = 20

# Maximum characters per text field (agent_action, user_correction).
MAX_FIELD_LENGTH = 500


class CorrectionEntry(TypedDict):
    correction_type: str
    agent_action: str
    user_correction: str
    timestamp: str


def is_correction_injection_enabled() -> bool:
    """Check whether correction injection is enabled via env var.

    The ``ENABLE_CORRECTION_INJECTION`` env var is set by the operator
    when the ``learning-agent-loop`` feature flag is active for the
    workspace.
    """
    return os.getenv("ENABLE_CORRECTION_INJECTION", "").strip().lower() in (
        "1",
        "true",
        "yes",
    )


class CorrectionLedger:
    """In-memory list of corrections for a single session.

    Thread safety: the existing per-session lock in SessionManager
    serialises turns, so concurrent mutation is not possible within
    a single session. No additional locking is needed.
    """

    def __init__(self) -> None:
        self._entries: list[CorrectionEntry] = []

    def append(self, entry: dict) -> None:
        """Add a correction to the ledger.

        Fields are truncated to ``MAX_FIELD_LENGTH`` characters. A
        timestamp is auto-generated if not provided.
        """
        agent_action = str(entry.get("agent_action", "") or "")[:MAX_FIELD_LENGTH]
        user_correction = str(entry.get("user_correction", "") or "")[:MAX_FIELD_LENGTH]

        stored: CorrectionEntry = {
            "correction_type": str(entry.get("correction_type", "") or ""),
            "agent_action": agent_action,
            "user_correction": user_correction,
            "timestamp": entry.get("timestamp")
            or datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        }
        self._entries.append(stored)
        logger.debug(
            "Correction appended to ledger: type=%s (total=%d)",
            stored["correction_type"],
            len(self._entries),
        )

    def render(self) -> str:
        """Render the correction block for context injection.

        Returns a markdown block with corrections in reverse chronological
        order (most recent first). Caps at ``MAX_RENDERED_ENTRIES``
        individual entries, with a summary line for any overflow.

        Returns empty string if no corrections have been logged.
        """
        if not self._entries:
            return ""

        lines: list[str] = ["## Corrections from this session\n"]

        total = len(self._entries)
        # Reverse chronological: most recent first
        recent = list(reversed(self._entries))
        displayed = recent[:MAX_RENDERED_ENTRIES]

        for entry in displayed:
            action = entry["agent_action"] or "(not specified)"
            correction = entry["user_correction"] or "(not specified)"
            lines.append(
                f"- **[{entry['correction_type']}]** ({entry['timestamp']}): "
                f"Agent did: {action} | "
                f"Should have: {correction}"
            )

        omitted = total - len(displayed)
        if omitted > 0:
            noun = "correction" if omitted == 1 else "corrections"
            lines.append(f"\n...and {omitted} earlier {noun} omitted")

        return "\n".join(lines)

    def __len__(self) -> int:
        return len(self._entries)
