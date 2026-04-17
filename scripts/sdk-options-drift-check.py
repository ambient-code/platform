#!/usr/bin/env python3
"""Detect drift between the installed claude-agent-sdk and the committed manifest.

Exit codes:
    0 - No drift detected
    1 - Drift detected (manifest updated in-place)
    2 - Error (import failure, missing file, etc.)
"""

from __future__ import annotations

import dataclasses
import json
import sys
from datetime import datetime, timezone
from pathlib import Path

MANIFEST_PATH = (
    Path(__file__).resolve().parent.parent
    / "components"
    / "runners"
    / "ambient-runner"
    / "sdk-options-manifest.json"
)


def get_current_fields() -> dict[str, dict[str, object]]:
    """Introspect ClaudeAgentOptions and return a dict of field name -> metadata."""
    # Pydantic v2
    if hasattr(ClaudeAgentOptions, "model_fields"):
        fields_map = ClaudeAgentOptions.model_fields
        result = {}
        for name, field_info in fields_map.items():
            annotation = field_info.annotation
            type_str = str(annotation) if annotation else "Any"
            required = field_info.is_required()
            result[name] = {"type": type_str, "required": required}
        return result

    # Pydantic v1
    if hasattr(ClaudeAgentOptions, "__fields__"):
        fields_map = ClaudeAgentOptions.__fields__
        result = {}
        for name, field_info in fields_map.items():
            type_str = (
                str(field_info.outer_type_)
                if hasattr(field_info, "outer_type_")
                else str(field_info.type_)
            )
            required = field_info.required
            result[name] = {"type": type_str, "required": required}
        return result

    # dataclass
    if dataclasses.is_dataclass(ClaudeAgentOptions):
        result = {}
        for f in dataclasses.fields(ClaudeAgentOptions):
            has_default = f.default is not dataclasses.MISSING
            has_factory = f.default_factory is not dataclasses.MISSING
            required = not has_default and not has_factory
            type_str = str(f.type) if f.type else "Any"
            result[f.name] = {"type": type_str, "required": required}
        return result

    print(
        "ERROR: ClaudeAgentOptions is not a Pydantic model or dataclass — cannot introspect fields",
        file=sys.stderr,
    )
    sys.exit(2)


def load_manifest() -> dict:
    """Load the existing manifest from disk."""
    if not MANIFEST_PATH.exists():
        print(f"ERROR: Manifest not found at {MANIFEST_PATH}", file=sys.stderr)
        sys.exit(2)
    with open(MANIFEST_PATH, encoding="utf-8") as fh:
        return json.load(fh)


def write_manifest(
    current_fields: dict[str, dict[str, object]], sdk_version: str
) -> None:
    """Write an updated manifest to disk."""
    manifest = {
        "description": "Canonical list of Claude Agent SDK ClaudeAgentOptions fields",
        "generatedFrom": "claude-agent-sdk (PyPI)",
        "generatedAt": datetime.now(timezone.utc).isoformat(),
        "sdkVersion": sdk_version,
        "options": current_fields,
    }
    with open(MANIFEST_PATH, "w", encoding="utf-8") as fh:
        json.dump(manifest, fh, indent=2)
        fh.write("\n")
    print(f"Updated manifest written to {MANIFEST_PATH}")


def main(sdk_version: str) -> int:
    current_fields = get_current_fields()
    manifest = load_manifest()
    manifest_options = manifest.get("options", {})

    current_names = set(current_fields.keys())
    manifest_names = set(manifest_options.keys())

    added = sorted(current_names - manifest_names)
    removed = sorted(manifest_names - current_names)

    # Check type changes for fields present in both
    changed: list[tuple[str, str, str]] = []
    for name in sorted(current_names & manifest_names):
        old_type = manifest_options[name].get("type", "")
        new_type = current_fields[name].get("type", "")
        if old_type != new_type:
            changed.append((name, old_type, new_type))

    if not added and not removed and not changed:
        print(
            f"No drift detected (SDK {sdk_version}, manifest {manifest.get('sdkVersion', 'unknown')})"
        )
        return 0

    # Drift found
    print("SDK options drift detected!")
    print(
        f"  SDK version: {sdk_version} (manifest: {manifest.get('sdkVersion', 'unknown')})"
    )
    if added:
        print(f"\n  Added fields ({len(added)}):")
        for name in added:
            print(f"    + {name}: {current_fields[name]['type']}")
    if removed:
        print(f"\n  Removed fields ({len(removed)}):")
        for name in removed:
            print(f"    - {name}: {manifest_options[name]['type']}")
    if changed:
        print(f"\n  Changed types ({len(changed)}):")
        for name, old_type, new_type in changed:
            print(f"    ~ {name}: {old_type} -> {new_type}")

    write_manifest(current_fields, sdk_version)
    return 1


if __name__ == "__main__":
    try:
        import importlib.metadata

        from claude_agent_sdk import ClaudeAgentOptions

        sdk_version = importlib.metadata.version("claude-agent-sdk")
    except ImportError as exc:
        print(f"ERROR: Cannot import claude_agent_sdk: {exc}", file=sys.stderr)
        print("Install it: pip install claude-agent-sdk", file=sys.stderr)
        sys.exit(2)
    except Exception as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        sys.exit(2)

    sys.exit(main(sdk_version))
