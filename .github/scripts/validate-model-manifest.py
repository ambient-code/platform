#!/usr/bin/env python3
"""Validate model manifest before auto-merge.

Checks that the updated manifest is structurally valid, preserves all
existing models, keeps default models available, and has at least one
available model per provider. Exits non-zero on any failure.
"""

import json
import subprocess
import sys
from pathlib import Path

MANIFEST_PATH = (
    Path(__file__).resolve().parent.parent.parent
    / "components"
    / "manifests"
    / "base"
    / "core"
    / "models.json"
)

REQUIRED_MODEL_FIELDS = {"id", "label", "vertexId", "provider", "available"}


def fail(msg: str) -> None:
    print(f"FAIL: {msg}", file=sys.stderr)
    sys.exit(1)


def load_committed_manifest() -> dict:
    result = subprocess.run(
        ["git", "show", f"HEAD:{MANIFEST_PATH.relative_to(Path.cwd())}"],
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        return {"models": []}
    return json.loads(result.stdout)


def main() -> int:
    # Load updated manifest
    try:
        manifest = json.loads(MANIFEST_PATH.read_text())
    except (json.JSONDecodeError, FileNotFoundError) as e:
        fail(f"cannot read manifest: {e}")

    # Structure checks
    if not isinstance(manifest.get("models"), list):
        fail("'models' must be a list")
    if not manifest["models"]:
        fail("models list is empty")
    if "defaultModel" not in manifest:
        fail("missing 'defaultModel'")

    models_by_id = {m["id"]: m for m in manifest["models"]}

    # Every model has required fields
    for model in manifest["models"]:
        missing = REQUIRED_MODEL_FIELDS - set(model.keys())
        if missing:
            fail(f"model {model.get('id', '?')} missing fields: {missing}")

    # Default model must exist and be available
    default = manifest["defaultModel"]
    if default not in models_by_id:
        fail(f"defaultModel '{default}' not in models list")
    if not models_by_id[default]["available"]:
        fail(f"defaultModel '{default}' is not available")

    # Provider defaults must exist and be available
    for provider, model_id in manifest.get("providerDefaults", {}).items():
        if model_id not in models_by_id:
            fail(f"providerDefault {provider}='{model_id}' not in models list")
        if not models_by_id[model_id]["available"]:
            fail(f"providerDefault '{model_id}' is not available")

    # No models removed vs committed version
    committed = load_committed_manifest()
    committed_ids = {m["id"] for m in committed["models"]}
    current_ids = {m["id"] for m in manifest["models"]}
    removed = committed_ids - current_ids
    if removed:
        fail(f"models removed (not allowed): {sorted(removed)}")

    # At least one available model per provider
    providers = {m["provider"] for m in manifest["models"]}
    for provider in providers:
        available = [
            m for m in manifest["models"]
            if m["provider"] == provider and m["available"]
        ]
        if not available:
            fail(f"no available models for provider '{provider}'")

    added = current_ids - committed_ids
    print(
        f"Validation passed: {len(manifest['models'])} models, "
        f"{len(added)} added, 0 removed"
    )
    if added:
        for model_id in sorted(added):
            m = models_by_id[model_id]
            print(f"  + {model_id} (available={m['available']})")

    return 0


if __name__ == "__main__":
    sys.exit(main())
