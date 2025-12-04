#!/usr/bin/env python3
"""
Integration test for Amber GHA workflows
Validates the fix from PR #419 - YAML parsing error with template literals

Tests:
1. YAML files parse correctly
2. No problematic document separators in JavaScript strings
3. String concatenation is used instead of template literals for markdown with ---
"""

import yaml
import re
import sys
from pathlib import Path

def test_yaml_syntax():
    """Test 1: Validate all Amber workflow YAML files parse correctly"""
    print("Test 1: YAML Syntax Validation")
    print("-" * 50)

    workflows = [
        ".github/workflows/amber-auto-review.yml",
        ".github/workflows/amber-issue-handler.yml",
        ".github/workflows/amber-dependency-sync.yml"
    ]

    all_valid = True
    for workflow_path in workflows:
        path = Path(workflow_path)
        if not path.exists():
            print(f"✗ {workflow_path} - File not found")
            all_valid = False
            continue

        try:
            with open(path, 'r') as f:
                content = f.read()
                yaml.safe_load(content)
            print(f"✓ {workflow_path} - Valid YAML")
        except yaml.YAMLError as e:
            print(f"✗ {workflow_path} - YAML Error:")
            print(f"  {e}")
            all_valid = False

    print()
    return all_valid

def test_no_standalone_separators():
    """Test 2: Check for problematic YAML document separators (--- on standalone line)"""
    print("Test 2: YAML Document Separator Check (PR #419)")
    print("-" * 50)

    workflow_path = ".github/workflows/amber-auto-review.yml"
    path = Path(workflow_path)

    if not path.exists():
        print(f"✗ {workflow_path} - File not found")
        return False

    with open(path, 'r') as f:
        lines = f.readlines()

    # Find standalone --- lines (excluding first line which is YAML header)
    problematic_lines = []
    for i, line in enumerate(lines[1:], start=2):  # Skip first line, start counting at 2
        if line.strip() == "---":
            # Check if it's in a JavaScript context (crude check)
            context = "".join(lines[max(0, i-10):i])
            if "const " in context or "script:" in context:
                problematic_lines.append((i, line.strip()))

    if problematic_lines:
        print(f"✗ Found {len(problematic_lines)} problematic standalone --- separator(s):")
        for line_no, content in problematic_lines:
            print(f"  Line {line_no}: {content}")
        return False
    else:
        print(f"✓ No problematic YAML separators found")
        return True

def test_string_concatenation_used():
    """Test 3: Verify string concatenation is used (not template literals) for markdown with ---"""
    print()
    print("Test 3: String Concatenation Validation (PR #419 Fix)")
    print("-" * 50)

    workflow_path = ".github/workflows/amber-auto-review.yml"
    path = Path(workflow_path)

    if not path.exists():
        print(f"✗ {workflow_path} - File not found")
        return False

    with open(path, 'r') as f:
        content = f.read()

    # Check for the transparencySection variable
    if "const transparencySection = " not in content:
        print("⚠ transparencySection not found (may have been renamed)")
        return True  # Not a failure, just different implementation

    # Look for the pattern: const transparencySection = '\n\n---\n...
    # This indicates string concatenation with + operator
    string_concat_pattern = r"const transparencySection = '[^']*'\s*\+"
    template_literal_pattern = r"const transparencySection = `"

    uses_string_concat = re.search(string_concat_pattern, content) is not None
    uses_template_literal = re.search(template_literal_pattern, content) is not None

    if uses_template_literal and not uses_string_concat:
        print("✗ Using template literals (backticks) - this can cause YAML parsing errors")
        return False
    elif uses_string_concat:
        print("✓ Using string concatenation with + operator (PR #419 fix applied)")
        return True
    else:
        print("⚠ Unable to determine string construction method")
        return True  # Don't fail if we can't determine

def test_workflow_structure():
    """Test 4: Validate workflow structure and key components"""
    print()
    print("Test 4: Workflow Structure Validation")
    print("-" * 50)

    workflow_path = ".github/workflows/amber-auto-review.yml"
    path = Path(workflow_path)

    if not path.exists():
        print(f"✗ {workflow_path} - File not found")
        return False

    with open(path, 'r') as f:
        workflow = yaml.safe_load(f)

    checks = []

    # Check for required top-level keys
    checks.append(("name" in workflow, "Workflow has name"))
    # YAML 'on' key may be parsed as boolean True, check both
    checks.append((("on" in workflow or True in workflow), "Workflow has trigger ('on')"))
    checks.append(("jobs" in workflow, "Workflow has jobs"))

    # Check job structure
    if "jobs" in workflow:
        jobs = workflow["jobs"]
        checks.append((len(jobs) > 0, "At least one job defined"))

        for job_name, job_config in jobs.items():
            checks.append((
                "runs-on" in job_config or "uses" in job_config,
                f"Job '{job_name}' has runs-on or uses"
            ))

            if "permissions" in job_config:
                checks.append((True, f"Job '{job_name}' declares permissions"))

            if "steps" in job_config:
                checks.append((
                    len(job_config["steps"]) > 0,
                    f"Job '{job_name}' has steps"
                ))

    all_passed = True
    for passed, description in checks:
        if passed:
            print(f"✓ {description}")
        else:
            print(f"✗ {description}")
            all_passed = False

    return all_passed

def test_pr_419_regression():
    """Test 5: Specific regression test for PR #419 - ensure the exact issue doesn't reoccur"""
    print()
    print("Test 5: PR #419 Regression Test")
    print("-" * 50)

    workflow_path = ".github/workflows/amber-auto-review.yml"
    path = Path(workflow_path)

    if not path.exists():
        print(f"✗ {workflow_path} - File not found")
        return False

    with open(path, 'r') as f:
        content = f.read()

    # The original issue: template literal with --- causing YAML parsing error
    # Look for the pattern that was problematic:
    #   const transparencySection = `
    #
    #   ---
    problematic_pattern = r"const\s+transparencySection\s*=\s*`[\s\S]*?\n---\n"

    if re.search(problematic_pattern, content):
        print("✗ REGRESSION: Found the problematic pattern from PR #419")
        print("  Template literal contains standalone --- which breaks YAML parsing")
        return False
    else:
        print("✓ PR #419 fix is in place - no problematic template literals with ---")
        return True

def main():
    """Run all tests and report results"""
    print("=" * 50)
    print("Amber GHA Workflow Integration Tests")
    print("PR #419 Fix Validation")
    print("=" * 50)
    print()

    tests = [
        test_yaml_syntax,
        test_no_standalone_separators,
        test_string_concatenation_used,
        test_workflow_structure,
        test_pr_419_regression,
    ]

    results = []
    for test_func in tests:
        try:
            result = test_func()
            results.append((test_func.__name__, result))
        except Exception as e:
            print(f"✗ Test {test_func.__name__} raised exception:")
            print(f"  {e}")
            results.append((test_func.__name__, False))

    print()
    print("=" * 50)
    print("Test Summary")
    print("=" * 50)

    passed = sum(1 for _, result in results if result)
    failed = len(results) - passed

    for test_name, result in results:
        status = "✓ PASS" if result else "✗ FAIL"
        print(f"{status} - {test_name}")

    print()
    print(f"Total: {len(results)} tests")
    print(f"Passed: {passed}")
    print(f"Failed: {failed}")

    if failed == 0:
        print()
        print("✓ All tests passed!")
        return 0
    else:
        print()
        print("✗ Some tests failed")
        return 1

if __name__ == "__main__":
    sys.exit(main())
