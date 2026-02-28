#!/usr/bin/env python3
"""Automated Vertex AI model discovery.

Maintains a curated list of Anthropic model base names, resolves their
latest Vertex AI version via the Model Garden API, probes each to confirm
availability, and updates the model manifest. Never removes models — only
adds new ones or updates the ``available`` / ``vertexId`` fields.

Required env vars:
    CLOUD_ML_REGION            - GCP region (e.g. us-east5)
    ANTHROPIC_VERTEX_PROJECT_ID - GCP project ID

Optional env vars:
    GOOGLE_APPLICATION_CREDENTIALS - Path to SA key (uses ADC otherwise)
    MANIFEST_PATH              - Override default manifest location
"""

import json
import os
import subprocess
import sys
import urllib.error
import urllib.request
from pathlib import Path

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

DEFAULT_MANIFEST = (
    Path(__file__).resolve().parent.parent
    / "components"
    / "manifests"
    / "base"
    / "models.json"
)

# Known Anthropic model base names. Add new models here as they are released.
# Version resolution and availability probing are automatic.
KNOWN_MODELS = [
    "claude-sonnet-4-6",
    "claude-sonnet-4-5",
    "claude-opus-4-6",
    "claude-opus-4-5",
    "claude-opus-4-1",
    "claude-haiku-4-5",
]

# Tier classification: models matching these patterns are "standard", others "premium"
STANDARD_TIER_PATTERNS = ["sonnet", "haiku"]


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def get_access_token() -> str:
    """Get a GCP access token via gcloud."""
    result = subprocess.run(
        ["gcloud", "auth", "print-access-token"],
        capture_output=True,
        text=True,
        check=True,
    )
    return result.stdout.strip()


def resolve_version(region: str, model_id: str, token: str) -> str | None:
    """Resolve the latest version for a model via the Model Garden get API.

    Returns the version string (e.g. "20250929") or None if the API call
    fails (permissions, model not found, etc.).
    """
    url = (
        f"https://{region}-aiplatform.googleapis.com/v1/"
        f"publishers/anthropic/models/{model_id}"
    )

    req = urllib.request.Request(
        url,
        headers={"Authorization": f"Bearer {token}"},
        method="GET",
    )

    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            data = json.loads(resp.read().decode())
    except (urllib.error.HTTPError, Exception) as e:
        print(f"  {model_id}: version resolution unavailable ({e})", file=sys.stderr)
        return None

    # The response name field is "publishers/anthropic/models/claude-sonnet-4-5@20250929"
    name = data.get("name", "")
    if "@" in name:
        return name.split("@", 1)[1]

    return data.get("versionId")


def model_id_to_label(model_id: str) -> str:
    """Convert a model ID like 'claude-opus-4-6' to 'Claude Opus 4.6'."""
    parts = model_id.split("-")
    result = []
    for part in parts:
        if part[0].isdigit():
            if result and result[-1][-1].isdigit():
                result[-1] += f".{part}"
            else:
                result.append(part)
        else:
            result.append(part.capitalize())
    return " ".join(result)


def classify_tier(model_id: str) -> str:
    """Classify a model as 'standard' or 'premium' based on its name."""
    lower = model_id.lower()
    for pattern in STANDARD_TIER_PATTERNS:
        if pattern in lower:
            return "standard"
    return "premium"


def probe_model(region: str, project_id: str, vertex_id: str, token: str) -> str:
    """Probe a Vertex AI model endpoint.

    Returns:
        "available"   - 200 or 400 (model exists, endpoint responds)
        "unavailable" - 404 (model not found)
        "unknown"     - any other status (transient error, leave unchanged)
    """
    url = (
        f"https://{region}-aiplatform.googleapis.com/v1/"
        f"projects/{project_id}/locations/{region}/"
        f"publishers/anthropic/models/{vertex_id}:rawPredict"
    )

    body = json.dumps({
        "anthropic_version": "vertex-2023-10-16",
        "max_tokens": 1,
        "messages": [{"role": "user", "content": "hi"}],
    }).encode()

    req = urllib.request.Request(
        url,
        data=body,
        headers={
            "Authorization": f"Bearer {token}",
            "Content-Type": "application/json",
        },
        method="POST",
    )

    try:
        with urllib.request.urlopen(req, timeout=30):
            return "available"
    except urllib.error.HTTPError as e:
        if e.code == 400:
            return "available"
        if e.code == 404:
            return "unavailable"
        print(f"  WARNING: unexpected HTTP {e.code} for {vertex_id}", file=sys.stderr)
        return "unknown"
    except Exception as e:
        print(f"  WARNING: probe error for {vertex_id}: {e}", file=sys.stderr)
        return "unknown"


def load_manifest(path: Path) -> dict:
    """Load the model manifest JSON."""
    with open(path) as f:
        return json.load(f)


def save_manifest(path: Path, manifest: dict) -> None:
    """Save the model manifest JSON with consistent formatting."""
    with open(path, "w") as f:
        json.dump(manifest, f, indent=2)
        f.write("\n")


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------


def main() -> int:
    region = os.environ.get("CLOUD_ML_REGION", "").strip()
    project_id = os.environ.get("ANTHROPIC_VERTEX_PROJECT_ID", "").strip()

    if not region or not project_id:
        print(
            "ERROR: CLOUD_ML_REGION and ANTHROPIC_VERTEX_PROJECT_ID must be set",
            file=sys.stderr,
        )
        return 1

    manifest_path = Path(os.environ.get("MANIFEST_PATH", str(DEFAULT_MANIFEST)))
    if not manifest_path.exists():
        print(f"ERROR: manifest not found at {manifest_path}", file=sys.stderr)
        return 1

    manifest = load_manifest(manifest_path)
    token = get_access_token()

    print(f"Processing {len(KNOWN_MODELS)} known model(s) in {region}/{project_id}...")

    changes = []

    for model_id in KNOWN_MODELS:
        # Step 1: Try to resolve the latest version via Model Garden API
        resolved_version = resolve_version(region, model_id, token)

        # Find existing entry in manifest
        existing = next(
            (m for m in manifest["models"] if m["id"] == model_id), None
        )

        # Determine the vertex ID to probe
        if resolved_version:
            vertex_id = f"{model_id}@{resolved_version}"
        elif existing and existing.get("vertexId"):
            # Fall back to whatever the manifest already has
            vertex_id = existing["vertexId"]
        else:
            # Last resort: probe with @default
            vertex_id = f"{model_id}@default"

        # Step 2: Probe availability
        status = probe_model(region, project_id, vertex_id, token)
        is_available = status == "available"

        if existing:
            # Update vertexId if version resolution found a newer one
            if existing.get("vertexId") != vertex_id and resolved_version:
                old_vid = existing.get("vertexId", "")
                existing["vertexId"] = vertex_id
                changes.append(
                    f"  {model_id}: vertexId updated {old_vid} -> {vertex_id}"
                )
                print(f"  {model_id}: vertexId updated -> {vertex_id}")

            if status == "unknown":
                print(
                    f"  {model_id}: probe inconclusive, "
                    f"leaving available={existing['available']}"
                )
                continue
            if existing["available"] != is_available:
                existing["available"] = is_available
                changes.append(
                    f"  {model_id}: available changed to {is_available}"
                )
                print(f"  {model_id}: available -> {is_available}")
            else:
                print(f"  {model_id}: unchanged (available={is_available})")
        else:
            if status == "unknown":
                print(f"  {model_id}: new model but probe inconclusive, skipping")
                continue
            new_entry = {
                "id": model_id,
                "label": model_id_to_label(model_id),
                "vertexId": vertex_id,
                "provider": "anthropic",
                "tier": classify_tier(model_id),
                "available": is_available,
            }
            manifest["models"].append(new_entry)
            changes.append(f"  {model_id}: added (available={is_available})")
            print(f"  {model_id}: NEW model added (available={is_available})")

    if changes:
        save_manifest(manifest_path, manifest)
        print(f"\n{len(changes)} change(s) written to {manifest_path}:")
        for c in changes:
            print(c)
    else:
        print("\nNo changes detected.")

    return 0


if __name__ == "__main__":
    sys.exit(main())
