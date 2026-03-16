#!/usr/bin/env python3
"""Automated Vertex AI model discovery.

Discovers models from Vertex AI publishers via the Model Garden list API,
filters by configured prefix patterns, resolves versions, probes each to
confirm availability, and updates the model manifest. Never removes models
— only adds new ones or updates the ``available`` / ``vertexId`` fields.

New models matching a prefix are auto-discovered without code changes.
For example, if Anthropic releases ``claude-opus-4-7``, it will be picked
up automatically because it matches the ``claude-`` prefix under the
``anthropic`` publisher.

Required env vars:
    GCP_REGION                 - GCP region (e.g. us-east5)
    GCP_PROJECT                - GCP project ID

Optional env vars:
    GOOGLE_APPLICATION_CREDENTIALS - Path to SA key (uses ADC otherwise)
    MANIFEST_PATH              - Override default manifest location
"""

import json
import os
import re
import subprocess
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from collections import defaultdict
from typing import NotRequired, TypedDict
from pathlib import Path

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

DEFAULT_MANIFEST = (
    Path(__file__).resolve().parent.parent.parent
    / "components"
    / "manifests"
    / "base"
    / "models.json"
)

# Keep only the N most recent versions per model family.
# e.g. claude-opus-4-6 and claude-opus-4-5 are kept, claude-opus-4-1 is dropped.
MAX_VERSIONS_PER_FAMILY = 2

# Model Garden list API pagination settings.
LIST_PAGE_SIZE = 100
MAX_LIST_PAGES = 20


# Publisher discovery configuration.
# prefixes:  only models whose ID starts with one of these are included.
# exclude:   model IDs matching these regex patterns are skipped (embeddings,
#            image models, legacy versions, etc.).
class PublisherConfig(TypedDict):
    publisher: str
    provider: str
    prefixes: list[str]
    exclude: list[str]
    version_cutoff: NotRequired[
        tuple[int, ...]
    ]  # models with version <= this are excluded


PUBLISHERS: list[PublisherConfig] = [
    {
        "publisher": "anthropic",
        "provider": "anthropic",
        "prefixes": ["claude-"],
        "exclude": [
            r"^claude-[a-z]+-\d+$",  # base aliases without minor version (claude-opus-4)
        ],
    },
    {
        "publisher": "google",
        "provider": "google",
        "prefixes": ["gemini-"],
        "exclude": [
            r"-\d{3}$",  # pinned versions like gemini-2.5-flash-001
            r"exp",  # experimental models
            r"embedding",
            r"imagen",
            r"veo",
            r"chirp",
            r"codey",
            r"medlm",
        ],
        "version_cutoff": (2, 0),  # exclude gemini 2.0 and older
    },
]

# Fallback seed list used when the list API is unavailable.
# Once the list API works, this is only used for models it might miss.
SEED_MODELS: list[tuple[str, str, str]] = [
    ("claude-sonnet-4-6", "anthropic", "anthropic"),
    ("claude-sonnet-4-5", "anthropic", "anthropic"),
    ("claude-opus-4-6", "anthropic", "anthropic"),
    ("claude-opus-4-5", "anthropic", "anthropic"),
    ("claude-haiku-4-5", "anthropic", "anthropic"),
    ("gemini-2.5-flash", "google", "google"),
    ("gemini-2.5-pro", "google", "google"),
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
    except subprocess.TimeoutExpired as err:
        raise RuntimeError("Timed out getting GCP access token via gcloud") from err
    except subprocess.CalledProcessError as err:
        raise RuntimeError("Failed to get GCP access token via gcloud") from err
    return result.stdout.strip()


def list_publisher_models(publisher: str, token: str) -> list[tuple[str, str | None]]:
    """List models from the Model Garden for a publisher.

    Uses the v1beta1 API: GET /publishers/{publisher}/models
    Returns a list of (model_id, version_id) tuples. version_id is the
    versionId from the API response (e.g. "20250929") or None if absent.
    Returns an empty list on failure (caller falls back to seed list).
    """
    base_url = "https://aiplatform.googleapis.com/v1beta1"
    all_models: list[tuple[str, str | None]] = []
    page_token = ""

    for _ in range(MAX_LIST_PAGES):
        params = {"pageSize": str(LIST_PAGE_SIZE)}
        if page_token:
            params["pageToken"] = page_token

        url = (
            f"{base_url}/publishers/{urllib.parse.quote(publisher, safe='')}"
            f"/models?{urllib.parse.urlencode(params)}"
        )

        data = None
        last_err: Exception | None = None
        for attempt in range(3):
            req = urllib.request.Request(
                url,
                headers={"Authorization": f"Bearer {token}"},
                method="GET",
            )
            try:
                with urllib.request.urlopen(req, timeout=30) as resp:
                    data = json.loads(resp.read().decode())
                break
            except urllib.error.HTTPError as e:
                # Auth failures are fatal — don't fall back to seeds with bad credentials
                if e.code in (401, 403):
                    raise RuntimeError(
                        f"list models for {publisher} failed (HTTP {e.code}): "
                        f"check GCP credentials and IAM permissions"
                    ) from e
                # Not found — retrying won't help
                if e.code == 404:
                    print(
                        f"  WARNING: list models for {publisher} returned 404",
                        file=sys.stderr,
                    )
                    return []
                last_err = e
            except Exception as e:
                last_err = e

            if attempt < 2:
                time.sleep(2**attempt)

        if data is None:
            print(
                f"  WARNING: list models for {publisher} failed after 3 attempts ({last_err})",
                file=sys.stderr,
            )
            return []

        for model in data.get("publisherModels", []):
            # name is like "publishers/google/models/gemini-2.5-flash"
            name = model.get("name", "")
            model_id = name.rsplit("/", 1)[-1] if "/" in name else name
            version_id = model.get("versionId")
            if model_id:
                all_models.append((model_id, version_id))

        page_token = data.get("nextPageToken", "")
        if not page_token:
            break

    return all_models


def discover_models(
    token: str, manifest: dict[str, object]
) -> list[tuple[str, str, str, str | None]]:
    """Discover models from all configured publishers.

    Queries the Model Garden list API for each publisher, filters by
    prefix patterns, and excludes unwanted model types. Falls back to
    the SEED_MODELS list for any publisher where the API fails.

    Provider default models (defaultModel + providerDefaults values) are
    exempt from version limiting and always kept.

    Returns a deduplicated list of (model_id, publisher, provider, version_id)
    tuples. version_id comes from the list API response and may be None for
    seed models or when the API doesn't provide it.
    """
    seen: set[str] = set()
    result: list[tuple[str, str, str, str | None]] = []

    # Collect per-publisher: (model_id, reason) for the summary table
    publisher_log: list[tuple[str, list[tuple[str, str]]]] = []

    for pub in PUBLISHERS:
        publisher = pub["publisher"]
        provider = pub["provider"]
        prefixes = pub["prefixes"]
        excludes = [re.compile(p) for p in pub["exclude"]]
        min_ver = pub.get("version_cutoff")

        api_models = list_publisher_models(publisher, token)
        log_entries: list[tuple[str, str]] = []

        if api_models:
            for model_id, version_id in sorted(api_models, key=lambda x: x[0]):
                if not any(model_id.startswith(p) for p in prefixes):
                    log_entries.append((model_id, "SKIP (prefix)"))
                    continue
                if any(pat.search(model_id) for pat in excludes):
                    log_entries.append((model_id, "EXCLUDE"))
                    continue
                if min_ver:
                    _, parsed_ver = parse_model_family(model_id)
                    if parsed_ver and parsed_ver <= min_ver:
                        log_entries.append((model_id, "EXCLUDE (version)"))
                        continue
                log_entries.append((model_id, "KEEP"))
                if model_id not in seen:
                    seen.add(model_id)
                    result.append((model_id, publisher, provider, version_id))
        else:
            print(
                f"  {publisher}: API unavailable, using seed list",
                file=sys.stderr,
            )

        publisher_log.append((publisher, log_entries))

    # Merge in seed models that weren't discovered by the API.
    # Apply version_cutoff so seed models respect the same filtering as API models.
    pub_by_name = {p["publisher"]: p for p in PUBLISHERS}
    for model_id, publisher, provider in SEED_MODELS:
        if model_id not in seen:
            cutoff = pub_by_name.get(publisher, {}).get("version_cutoff")
            if cutoff:
                _, parsed_ver = parse_model_family(model_id)
                if parsed_ver and parsed_ver <= cutoff:
                    continue
            seen.add(model_id)
            result.append((model_id, publisher, provider, None))

    # Build the set of protected model IDs (defaults are never dropped)
    protected: set[str] = set()
    default_model = manifest.get("defaultModel", "")
    if default_model:
        protected.add(default_model)
    for model_id in manifest.get("providerDefaults", {}).values():
        if model_id:
            protected.add(model_id)

    # Keep only the N most recent versions per model family
    result = keep_latest_versions(result, MAX_VERSIONS_PER_FAMILY, protected)
    kept_ids = {entry[0] for entry in result}

    # Print the summary table with accurate final disposition
    for publisher, log_entries in publisher_log:
        if not log_entries:
            continue
        print(f"  {publisher}: {len(log_entries)} model(s) from API")
        for model_id, reason in log_entries:
            if reason == "KEEP" and model_id in protected:
                reason = "KEEP (default)"
            elif reason == "KEEP" and model_id not in kept_ids:
                reason = "SKIP (version limit)"
            print(f"    {model_id:<50s} {reason}")

    return sorted(result, key=lambda x: x[0])


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


# Temporal qualifiers stripped from model names before determining family.
# These are release stages and date stamps, not part of the model identity.
# Applied to individual dash-segments after splitting.
_QUALIFIER_PATTERNS = [
    re.compile(r"^preview$"),
    re.compile(r"^exp$"),
    re.compile(r"^\d{2}$"),  # date segments like 04, 17 (from stamps like 04-17)
]


def parse_model_family(model_id: str) -> tuple[str, tuple[int, ...]]:
    """Split a model ID into (family, version_tuple).

    Handles two naming conventions:

    1. Semver segment (e.g. "2.5" in "gemini-2.5-flash"):
       The first segment matching ``\\d+\\.\\d+`` is extracted as the version
       and removed from the family name. Temporal qualifiers (preview, exp,
       date stamps) are also stripped so that preview variants group with
       their stable counterpart.
         "gemini-2.5-flash"                      -> ("gemini-flash", (2, 5))
         "gemini-2.5-flash-lite"                 -> ("gemini-flash-lite", (2, 5))
         "gemini-2.5-flash-preview-04-17"        -> ("gemini-flash", (2, 5))
         "gemini-2.0-flash-preview-image-generation"
                                                 -> ("gemini-flash-image-generation", (2, 0))
         "gemini-3.1-flash-image-preview"        -> ("gemini-flash-image", (3, 1))

    2. Trailing digits (e.g. "claude-opus-4-6"):
       Trailing numeric dash-segments form the version.
         "claude-opus-4-6"       -> ("claude-opus", (4, 6))
         "claude-haiku-4-5"      -> ("claude-haiku", (4, 5))
    """
    parts = model_id.split("-")

    # Check for a semver segment (e.g. "2.5", "3.1")
    for i, part in enumerate(parts):
        if re.fullmatch(r"\d+\.\d+", part):
            version = tuple(int(x) for x in part.split("."))
            family_parts = parts[:i] + parts[i + 1 :]
            # Strip temporal qualifiers from family name
            family_parts = [
                p
                for p in family_parts
                if not any(q.match(p) for q in _QUALIFIER_PATTERNS)
            ]
            return "-".join(family_parts), version

    # Fall back to trailing numeric segments
    version_parts: list[int] = []
    while parts and parts[-1].isdigit():
        version_parts.insert(0, int(parts.pop()))
    family = "-".join(parts) if parts else model_id
    return family, tuple(version_parts)


def keep_latest_versions(
    models: list[tuple[str, str, str, str | None]],
    max_versions: int,
    protected: set[str] | None = None,
) -> list[tuple[str, str, str, str | None]]:
    """Keep only the N most recent versions per model family.

    Models without a parseable version (no semver or trailing digits) are always kept.
    Provider default models (from providerDefaults in the manifest) are exempt
    from version limiting and always kept.
    """
    protected = protected or set()

    # Group by family
    families: dict[
        str, list[tuple[tuple[int, ...], tuple[str, str, str, str | None]]]
    ] = defaultdict(list)
    no_version: list[tuple[str, str, str, str | None]] = []

    for entry in models:
        model_id = entry[0]
        if model_id in protected:
            no_version.append(entry)
            continue
        family, version = parse_model_family(model_id)
        if version:
            families[family].append((version, entry))
        else:
            no_version.append(entry)

    result: list[tuple[str, str, str, str | None]] = list(no_version)
    for family, versioned in sorted(families.items()):
        # Sort by version descending, keep top N
        versioned.sort(key=lambda x: x[0], reverse=True)
        kept = [entry for _, entry in versioned[:max_versions]]
        dropped = [entry[0] for _, entry in versioned[max_versions:]]
        if dropped:
            print(f"  {family}: keeping {max_versions} latest, dropping {dropped}")
        result.extend(kept)

    return sorted(result, key=lambda x: x[0])


def _build_probe_request(
    region: str, project_id: str, vertex_id: str, publisher: str, token: str
) -> urllib.request.Request:
    """Build the probe HTTP request for a given publisher."""
    safe_vid = urllib.parse.quote(vertex_id, safe="@")
    if publisher == "google":
        url = (
            f"https://{region}-aiplatform.googleapis.com/v1/"
            f"projects/{project_id}/locations/{region}/"
            f"publishers/google/models/{safe_vid}:generateContent"
        )
        body = json.dumps(
            {
                "contents": [{"parts": [{"text": "hi"}]}],
                "generationConfig": {"maxOutputTokens": 1},
            }
        ).encode()
    elif publisher == "anthropic":
        url = (
            f"https://{region}-aiplatform.googleapis.com/v1/"
            f"projects/{project_id}/locations/{region}/"
            f"publishers/anthropic/models/{safe_vid}:rawPredict"
        )
        body = json.dumps(
            {
                "anthropic_version": "vertex-2023-10-16",
                "max_tokens": 1,
                "messages": [{"role": "user", "content": "hi"}],
            }
        ).encode()
    else:
        raise ValueError(f"Unknown publisher: {publisher!r}")

    return urllib.request.Request(
        url,
        data=body,
        headers={
            "Authorization": f"Bearer {token}",
            "Content-Type": "application/json",
        },
        method="POST",
    )


def probe_model(
    region: str, project_id: str, vertex_id: str, publisher: str, token: str
) -> str:
    """Probe a Vertex AI model endpoint.

    Returns:
        "available"   - 200 or 400 (model exists, endpoint responds)
        "unavailable" - 404 (model not found)
        "unknown"     - any other status (transient error, leave unchanged)
    """
    if publisher not in ("anthropic", "google"):
        print(
            f"  {vertex_id}: unsupported publisher {publisher!r}",
            file=sys.stderr,
        )
        return "unknown"

    last_err = None
    for attempt in range(3):
        req = _build_probe_request(region, project_id, vertex_id, publisher, token)

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
    """Load the model manifest JSON, or return a blank manifest if missing.

    Raises on malformed JSON to prevent overwriting a corrupt file.
    Returns a blank manifest only when the file does not exist yet.
    """
    if not path.exists():
        return {"version": 1, "defaultModel": "claude-sonnet-4-5", "models": []}

    with open(path) as f:
        data = json.load(f)

    if not isinstance(data, dict) or "models" not in data:
        raise ValueError(
            f"manifest at {path} is missing required 'models' key — "
            f"fix the file manually or delete it to start fresh"
        )

    return data


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

    # Discover models from the Model Garden API + seed list fallback
    print("Discovering models from Vertex AI Model Garden...")
    models_to_process = discover_models(token, manifest)
    print(f"Processing {len(models_to_process)} model(s) in {region}/{project_id}...")

    changes = []

    for model_id, publisher, provider, version_id in models_to_process:
        # Find existing entry in manifest
        existing = next((m for m in manifest["models"] if m["id"] == model_id), None)

        # Determine the vertex ID to probe.
        # version_id comes from the list API; fall back to existing manifest
        # entry or @default if neither is available.
        if version_id:
            vertex_id = f"{model_id}@{version_id}"
        elif existing and existing.get("vertexId"):
            vertex_id = existing["vertexId"]
        else:
            vertex_id = f"{model_id}@default"

        # Probe availability
        status = probe_model(region, project_id, vertex_id, publisher, token)
        is_available = status == "available"

        if existing:
            # Update vertexId if version resolution found a newer one
            if existing.get("vertexId") != vertex_id and version_id:
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
                "provider": provider,
                "available": is_available,
                "featureGated": True,  # New models require explicit opt-in via feature flag
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
