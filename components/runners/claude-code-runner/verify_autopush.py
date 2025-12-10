#!/usr/bin/env python3
"""
Verification script for autoPush implementation.

This script demonstrates the autoPush flag extraction from REPOS_JSON
without requiring the full wrapper environment.
"""

import json


def parse_repos_config(repos_json_str: str) -> list[dict]:
    """
    Simplified version of _get_repos_config() for verification.

    Extracts name, input, output, and autoPush from REPOS_JSON.
    """
    try:
        if not repos_json_str.strip():
            return []

        data = json.loads(repos_json_str)
        if not isinstance(data, list):
            return []

        result = []
        for item in data:
            if not isinstance(item, dict):
                continue

            name = str(item.get('name', '')).strip()
            input_obj = item.get('input') or {}
            output_obj = item.get('output')

            # Get URL from input
            url = str((input_obj or {}).get('url', '')).strip()

            if not name and url:
                # Derive name from URL (simplified)
                parts = url.rstrip('/').split('/')
                if parts:
                    name = parts[-1].removesuffix('.git').strip()

            if name and isinstance(input_obj, dict) and url:
                # Extract autoPush flag (default to False if not present)
                auto_push = item.get('autoPush', False)
                result.append({
                    'name': name,
                    'input': input_obj,
                    'output': output_obj,
                    'autoPush': auto_push
                })

        return result
    except Exception as e:
        print(f"Error parsing repos: {e}")
        return []


def should_push_repo(repo_config: dict) -> tuple[bool, str]:
    """
    Check if a repo should be pushed based on autoPush flag.

    Returns (should_push, reason)
    """
    name = repo_config.get('name', 'unknown')
    auto_push = repo_config.get('autoPush', False)

    if not auto_push:
        return False, f"autoPush disabled for {name}"

    output = repo_config.get('output')
    if not output or not output.get('url'):
        return False, f"No output URL configured for {name}"

    return True, f"Will push {name} (autoPush enabled)"


def main():
    """Run verification tests"""

    print("=== AutoPush Implementation Verification ===\n")

    # Test Case 1: Mixed autoPush flags
    print("Test 1: Mixed autoPush settings")
    repos_json = json.dumps([
        {
            "name": "repo-push",
            "input": {"url": "https://github.com/org/repo-push", "branch": "main"},
            "output": {"url": "https://github.com/user/fork-push", "branch": "feature"},
            "autoPush": True
        },
        {
            "name": "repo-no-push",
            "input": {"url": "https://github.com/org/repo-no-push", "branch": "main"},
            "output": {"url": "https://github.com/user/fork-no-push", "branch": "feature"},
            "autoPush": False
        },
        {
            "name": "repo-default",
            "input": {"url": "https://github.com/org/repo-default", "branch": "main"},
            "output": {"url": "https://github.com/user/fork-default", "branch": "feature"}
            # No autoPush field - should default to False
        }
    ])

    repos = parse_repos_config(repos_json)
    print(f"Parsed {len(repos)} repos:")

    for repo in repos:
        should_push, reason = should_push_repo(repo)
        status = "✓ PUSH" if should_push else "✗ SKIP"
        print(f"  {status}: {reason}")
        print(f"         autoPush={repo.get('autoPush', 'not set')}")

    print()

    # Test Case 2: All autoPush enabled
    print("Test 2: All repos with autoPush=true")
    repos_json = json.dumps([
        {
            "name": "repo1",
            "input": {"url": "https://github.com/org/repo1", "branch": "main"},
            "output": {"url": "https://github.com/user/fork1", "branch": "feature"},
            "autoPush": True
        },
        {
            "name": "repo2",
            "input": {"url": "https://github.com/org/repo2", "branch": "develop"},
            "output": {"url": "https://github.com/user/fork2", "branch": "feature"},
            "autoPush": True
        }
    ])

    repos = parse_repos_config(repos_json)
    for repo in repos:
        should_push, reason = should_push_repo(repo)
        status = "✓ PUSH" if should_push else "✗ SKIP"
        print(f"  {status}: {reason}")

    print()

    # Test Case 3: All autoPush disabled
    print("Test 3: All repos with autoPush=false")
    repos_json = json.dumps([
        {
            "name": "repo1",
            "input": {"url": "https://github.com/org/repo1", "branch": "main"},
            "autoPush": False
        },
        {
            "name": "repo2",
            "input": {"url": "https://github.com/org/repo2", "branch": "develop"},
            "autoPush": False
        }
    ])

    repos = parse_repos_config(repos_json)
    for repo in repos:
        should_push, reason = should_push_repo(repo)
        status = "✓ PUSH" if should_push else "✗ SKIP"
        print(f"  {status}: {reason}")

    print("\n=== Verification Complete ===")


if __name__ == '__main__':
    main()
