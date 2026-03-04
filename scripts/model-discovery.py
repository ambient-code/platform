#!/usr/bin/env python3
"""Automated Vertex AI model discovery.

Maintains a curated list of Anthropic model base names, resolves their
latest Vertex AI version via the Model Garden API, probes each to confirm
availability, and updates the model manifest. Never removes models — only
adds new ones or updates the ``available`` / ``vertexId`` fields.

Required env vars:
    GCP_REGION                 - GCP region (e.g. us-east5)
    GCP_PROJECT                - GCP project ID

Optional env vars:
    GOOGLE_APPLICATION_CREDENTIALS - Path to SA key (uses ADC otherwise)
    MANIFEST_PATH              - Override default manifest location
"""

import json
import os
import subprocess
import sys
import time
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
    "claude-haiku-4-5",
]


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def get_access_token() -> str:
    """Get a GCP access token via gcloud."""
    try:
        result = subprocess.run(
            ["gcloud", "auth", "print-access-token"],
            capture_output=True,
            text=True,
            check=True,
            timeout=30,
        )
    except subprocess.TimeoutExpired:
        raise RuntimeError("Timed out getting GCP access token via gcloud")
    except subprocess.CalledProcessError:
        raise RuntimeError("Failed to get GCP access token via gcloud")
    return result.stdout.strip()


def resolve_version(region: str, model_id: str, token: str) -> str | None:
    """Resolve the latest version for a model via the Model Garden API.

    Returns the version string (e.g. "20250929") or None if the API call
    fails (permissions, model not found, etc.).

    Note: requires ``roles/serviceusage.serviceUsageConsumer`` on the GCP
    project. Works in CI via the Workload Identity service account; may
    return None locally if the user lacks this role.
    """
    url = (
        f"https://{region}-aiplatform.googleapis.com/v1/"
        f"publishers/anthropic/models/{model_id}"
    )

    last_err = None
    for attempt in range(3):
        req = urllib.request.Request(
            url,
            headers={"Authorization": f"Bearer {token}"},
            method="GET",
        )
        try:
            with urllib.request.urlopen(req, timeout=30) as resp:
                data = json.loads(resp.read().decode())

            name = data.get("name", "")
            if "@" in name:
                return name.split("@", 1)[1]
            return data.get("versionId")

        except urllib.error.HTTPError as e:
            if e.code in (403, 404):
                # Permission denied or not found — retrying won't help
                print(
                    f"  {model_id}: version resolution unavailable (HTTP {e.code})",
                    file=sys.stderr,
                )
                return None
            last_err = e
        except Exception as e:
            last_err = e

        if attempt < 2:
            time.sleep(2**attempt)  # 1s, 2s backoff

    print(
        f"  {model_id}: version resolution failed after 3 attempts ({last_err})",
        file=sys.stderr,
    )
    return None


def model_id_to_label(model_id: str) -> str:
    """Convert a model ID like 'claude-opus-4-6' to 'Claude Opus 4.6'."""
    parts = model_id.split("-")
    result = []
    for part in parts:
        if part and part[0].isdigit():
            if result and result[-1][-1].isdigit():
                result[-1] += f".{part}"
            else:
                result.append(part)
        elif part:
            result.append(part.capitalize())
    return " ".join(result)


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

    body = json.dumps(
        {
            "anthropic_version": "vertex-2023-10-16",
            "max_tokens": 1,
            "messages": [{"role": "user", "content": "hi"}],
        }
    ).encode()

    last_err = None
    for attempt in range(3):
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
            if e.code in (429, 500, 502, 503, 504):
                last_err = e
            else:
                print(
                    f"  WARNING: unexpected HTTP {e.code} for {vertex_id}",
                    file=sys.stderr,
                )
                return "unknown"
        except Exception as e:
            last_err = e

        if attempt < 2:
            time.sleep(2**attempt)

    print(
        f"  WARNING: probe failed after 3 attempts for {vertex_id} ({last_err})",
        file=sys.stderr,
    )
    return "unknown"


def load_manifest(path: Path) -> dict:
    """Load the model manifest JSON, or return a blank manifest if missing/empty."""
    blank = {"version": 1, "defaultModel": "claude-sonnet-4-5", "models": []}
    if not path.exists():
        return blank
    try:
        with open(path) as f:
            data = json.load(f)
        if not isinstance(data, dict) or "models" not in data:
            return blank
        return data
    except (json.JSONDecodeError, ValueError) as e:
        print(
            f"WARNING: malformed manifest at {path}, starting fresh ({e})",
            file=sys.stderr,
        )
        return blank


def save_manifest(path: Path, manifest: dict) -> None:
    """Save the model manifest JSON with consistent formatting."""
    with open(path, "w") as f:
        json.dump(manifest, f, indent=2)
        f.write("\n")


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------


def main() -> int:
    region = os.environ.get("GCP_REGION", "").strip()
    project_id = os.environ.get("GCP_PROJECT", "").strip()

    if not region or not project_id:
        print(
            "ERROR: GCP_REGION and GCP_PROJECT must be set",
            file=sys.stderr,
        )
        return 1

    manifest_path = Path(os.environ.get("MANIFEST_PATH", str(DEFAULT_MANIFEST)))
    manifest = load_manifest(manifest_path)
    token = get_access_token()

    print(f"Processing {len(KNOWN_MODELS)} known model(s) in {region}/{project_id}...")

    changes = []

    for model_id in KNOWN_MODELS:
        # Try to resolve the latest version via Model Garden API
        resolved_version = resolve_version(region, model_id, token)

        # Find existing entry in manifest
        existing = next((m for m in manifest["models"] if m["id"] == model_id), None)

        # Determine the vertex ID to probe
        if resolved_version:
            vertex_id = f"{model_id}@{resolved_version}"
        elif existing and existing.get("vertexId"):
            vertex_id = existing["vertexId"]
        else:
            vertex_id = f"{model_id}@default"

        # Probe availability
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
                changes.append(f"  {model_id}: available changed to {is_available}")
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
