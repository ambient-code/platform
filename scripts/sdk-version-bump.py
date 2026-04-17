#!/usr/bin/env python3
"""
SDK Version Bump Script

Checks PyPI for new releases of claude-agent-sdk and anthropic, and updates
the runner's pyproject.toml + uv.lock when newer versions are available.
Generates a changelog from GitHub releases for PR descriptions.

Used by .github/workflows/sdk-version-bump.yml (daily schedule).

Usage:
    python scripts/sdk-version-bump.py [--check-only] [--package PKG]

Exit codes:
    0 - Success (update applied or no update needed)
    1 - Error
    2 - Update available (--check-only mode)
"""

import argparse
import json
import re
import subprocess
import sys
import urllib.request
from dataclasses import dataclass
from pathlib import Path
from typing import Optional

sys.path.insert(0, str(Path(__file__).resolve().parent))
from sdk_report import build_report, render_report_markdown


# Packages to track: (pypi_name, github_owner/repo, pyproject_key)
TRACKED_PACKAGES = [
    {
        "pypi_name": "claude-agent-sdk",
        "github_repo": "anthropics/claude-agent-sdk-python",
        "pyproject_key": "claude-agent-sdk",
    },
    {
        "pypi_name": "anthropic",
        "github_repo": "anthropics/anthropic-sdk-python",
        "pyproject_key": "anthropic[vertex]",
    },
]

PYPROJECT_PATH = Path("components/runners/ambient-runner/pyproject.toml")


@dataclass
class VersionInfo:
    """Version state for a tracked package."""

    package_name: str
    pypi_name: str
    github_repo: str
    current_version: str
    latest_version: str
    needs_update: bool


@dataclass
class ChangelogEntry:
    """A single release changelog entry."""

    version: str
    body: str


def fetch_pypi_latest(package_name: str) -> Optional[str]:
    """Fetch the latest version of a package from PyPI.

    Args:
        package_name: The PyPI package name.

    Returns:
        Latest version string, or None on failure.
    """
    url = f"https://pypi.org/pypi/{package_name}/json"
    try:
        req = urllib.request.Request(url, headers={"Accept": "application/json"})
        with urllib.request.urlopen(req, timeout=30) as resp:
            data = json.loads(resp.read().decode())
            return data["info"]["version"]
    except Exception as e:
        print(f"  Warning: Failed to fetch PyPI info for {package_name}: {e}")
        return None


def parse_current_version(pyproject_path: Path, dep_key: str) -> Optional[str]:
    """Parse the current minimum version constraint from pyproject.toml.

    Looks for patterns like: "package>=1.2.3" or "package[extras]>=1.2.3"

    Args:
        pyproject_path: Path to pyproject.toml.
        dep_key: Dependency key as it appears in pyproject.toml (e.g. "anthropic[vertex]").

    Returns:
        Current version string (without operator), or None if not found.
    """
    content = pyproject_path.read_text()

    # Escape brackets for regex
    escaped_key = re.escape(dep_key)
    pattern = rf'"{escaped_key}>=([\d.]+)"'
    match = re.search(pattern, content)
    if match:
        return match.group(1)
    return None


def update_pyproject_version(
    pyproject_path: Path, dep_key: str, old_version: str, new_version: str
) -> bool:
    """Update a dependency version in pyproject.toml.

    Args:
        pyproject_path: Path to pyproject.toml.
        dep_key: Dependency key (e.g. "claude-agent-sdk").
        old_version: Current version string.
        new_version: New version string.

    Returns:
        True if the file was modified.
    """
    content = pyproject_path.read_text()
    old_spec = f'"{dep_key}>={old_version}"'
    new_spec = f'"{dep_key}>={new_version}"'

    if old_spec not in content:
        print(f"  Error: Could not find {old_spec} in {pyproject_path}")
        return False

    content = content.replace(old_spec, new_spec)
    pyproject_path.write_text(content)
    return True


def regenerate_lockfile(runner_dir: Path) -> bool:
    """Run `uv lock` to regenerate the lockfile.

    Args:
        runner_dir: Path to the runner component directory.

    Returns:
        True if successful.
    """
    print("  Running uv lock...")
    result = subprocess.run(
        ["uv", "lock"],
        cwd=runner_dir,
        capture_output=True,
        text=True,
        timeout=120,
        check=False,
    )
    if result.returncode != 0:
        print(f"  Error: uv lock failed:\n{result.stderr}")
        return False
    print("  Lock file regenerated successfully.")
    return True


def _fetch_all_releases(github_repo: str) -> list[dict]:
    """Fetch all releases from GitHub, paginating if necessary."""
    releases: list[dict] = []
    page = 1
    while True:
        url = (
            f"https://api.github.com/repos/{github_repo}"
            f"/releases?per_page=100&page={page}"
        )
        try:
            req = urllib.request.Request(
                url,
                headers={
                    "Accept": "application/vnd.github+json",
                    "X-GitHub-Api-Version": "2022-11-28",
                },
            )
            with urllib.request.urlopen(req, timeout=30) as resp:
                batch = json.loads(resp.read().decode())
        except Exception as e:
            print(
                f"  Warning: Failed to fetch GitHub releases "
                f"for {github_repo} (page {page}): {e}"
            )
            break

        if not batch:
            break
        releases.extend(batch)
        if len(batch) < 100:
            break
        page += 1

    return releases


def fetch_github_changelog(
    github_repo: str, from_version: str, to_version: str
) -> list[ChangelogEntry]:
    """Fetch release notes from GitHub for versions between from and to (exclusive/inclusive).

    Args:
        github_repo: GitHub owner/repo string.
        from_version: Current version (exclusive).
        to_version: Target version (inclusive).

    Returns:
        List of ChangelogEntry objects, newest first.
    """
    releases = _fetch_all_releases(github_repo)

    entries = []
    from_parts = _version_tuple(from_version)
    to_parts = _version_tuple(to_version)

    for release in releases:
        tag = release.get("tag_name", "")
        # Strip leading 'v'
        version_str = tag.lstrip("v")
        version_parts = _version_tuple(version_str)

        # Include versions where: from < version <= to
        if from_parts < version_parts <= to_parts:
            body = release.get("body", "") or ""
            entries.append(ChangelogEntry(version=version_str, body=body.strip()))

    # Newest first
    entries.sort(key=lambda e: _version_tuple(e.version), reverse=True)
    return entries


def _version_tuple(version_str: str) -> tuple[int, ...]:
    """Parse a version string into a comparable tuple of ints.

    Handles pre-release suffixes like '1.0.0rc1' by extracting leading digits.
    """
    try:
        parts = []
        for segment in version_str.split("."):
            match = re.match(r"(\d+)", segment)
            parts.append(int(match.group(1)) if match else 0)
        return tuple(parts) if parts else (0,)
    except (ValueError, AttributeError):
        return (0,)


def check_versions(repo_root: Path) -> list[VersionInfo]:
    """Check all tracked packages for available updates.

    Args:
        repo_root: Repository root path.

    Returns:
        List of VersionInfo for each tracked package.
    """
    pyproject = repo_root / PYPROJECT_PATH
    results = []

    for pkg in TRACKED_PACKAGES:
        print(f"Checking {pkg['pypi_name']}...")
        current = parse_current_version(pyproject, pkg["pyproject_key"])
        if current is None:
            print(f"  Warning: Could not find {pkg['pyproject_key']} in {pyproject}")
            continue

        latest = fetch_pypi_latest(pkg["pypi_name"])
        if latest is None:
            continue

        needs_update = _version_tuple(latest) > _version_tuple(current)
        status = "UPDATE AVAILABLE" if needs_update else "up to date"
        print(f"  Current: {current}, Latest: {latest} ({status})")

        results.append(
            VersionInfo(
                package_name=pkg["pyproject_key"],
                pypi_name=pkg["pypi_name"],
                github_repo=pkg["github_repo"],
                current_version=current,
                latest_version=latest,
                needs_update=needs_update,
            )
        )

    return results


def apply_updates(repo_root: Path, versions: list[VersionInfo]) -> tuple[bool, str]:
    """Apply version updates to pyproject.toml and regenerate lockfile.

    Args:
        repo_root: Repository root path.
        versions: Version info list (only those with needs_update=True are applied).

    Returns:
        Tuple of (success, pr_body_markdown).
    """
    pyproject = repo_root / PYPROJECT_PATH
    runner_dir = pyproject.parent
    updates = [v for v in versions if v.needs_update]

    if not updates:
        return True, ""

    # Update pyproject.toml for each package
    for v in updates:
        print(f"Updating {v.package_name}: {v.current_version} -> {v.latest_version}")
        if not update_pyproject_version(
            pyproject, v.package_name, v.current_version, v.latest_version
        ):
            return False, ""

    # Regenerate lock file
    if not regenerate_lockfile(runner_dir):
        return False, ""

    # Build feature analysis reports
    reports = []
    for v in updates:
        print(f"Fetching changelog for {v.pypi_name}...")
        entries = fetch_github_changelog(
            v.github_repo, v.current_version, v.latest_version
        )
        report = build_report(v.pypi_name, v.current_version, v.latest_version, entries)
        reports.append(report)

    pr_body = render_report_markdown(reports)
    return True, pr_body


def build_pr_title(versions: list[VersionInfo]) -> str:
    """Build a concise PR title from the updated packages.

    Args:
        versions: Version info list (only those with needs_update=True).

    Returns:
        PR title string.
    """
    updates = [v for v in versions if v.needs_update]
    if not updates:
        return "deps(runner): no updates"
    parts = []
    for v in updates:
        parts.append(f"{v.pypi_name} {v.latest_version}")
    return f"deps(runner): bump {', '.join(parts)}"


def main() -> int:
    parser = argparse.ArgumentParser(description="SDK version bump tool")
    parser.add_argument(
        "--check-only",
        action="store_true",
        help="Only check for updates, don't apply them",
    )
    parser.add_argument(
        "--package",
        choices=["claude-agent-sdk", "anthropic", "all"],
        default="all",
        help="Which package to check (default: all)",
    )
    args = parser.parse_args()

    # Determine repo root
    script_dir = Path(__file__).resolve().parent
    repo_root = script_dir.parent

    print("=" * 60)
    print("SDK Version Bump")
    print("=" * 60)
    print()

    # Check versions
    versions = check_versions(repo_root)

    if args.package != "all":
        versions = [v for v in versions if v.pypi_name == args.package]

    updates_available = any(v.needs_update for v in versions)

    if not updates_available:
        print("\nAll tracked SDKs are up to date.")
        return 0

    if args.check_only:
        print("\nUpdates available (--check-only mode, not applying).")
        return 2

    # Apply updates
    print()
    success, pr_body = apply_updates(repo_root, versions)
    if not success:
        print("\nFailed to apply updates.")
        return 1

    # Write outputs for GitHub Actions
    pr_title = build_pr_title(versions)

    # Write to files so the workflow can read them
    output_dir = repo_root / ".sdk-bump-output"
    output_dir.mkdir(exist_ok=True)
    (output_dir / "pr-title.txt").write_text(pr_title)
    (output_dir / "pr-body.md").write_text(pr_body)

    print(f"\nPR title: {pr_title}")
    print(f"PR body written to {output_dir / 'pr-body.md'}")
    print("\nDone.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
