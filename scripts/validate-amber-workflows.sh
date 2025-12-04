#!/bin/bash
# Amber GHA Workflow Validation Script
# Tests Amber workflow files for syntax, security, and functionality

set -e

echo "🔍 Amber GHA Workflow Validation Suite"
echo "======================================"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Helper function to report test results
report_test() {
    TESTS_RUN=$((TESTS_RUN + 1))
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $2"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "${RED}✗${NC} $2"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

# Test 1: Validate YAML syntax for all Amber workflows
echo "Test 1: YAML Syntax Validation"
echo "-------------------------------"

WORKFLOW_FILES=(
    ".github/workflows/amber-auto-review.yml"
    ".github/workflows/amber-issue-handler.yml"
    ".github/workflows/amber-dependency-sync.yml"
)

for workflow in "${WORKFLOW_FILES[@]}"; do
    if command -v yamllint &> /dev/null; then
        yamllint -d "{extends: default, rules: {line-length: {max: 200}}}" "$workflow" 2>&1
        report_test $? "YAML syntax valid: $workflow"
    else
        # Fallback to basic validation with Python
        python3 -c "import yaml; yaml.safe_load(open('$workflow'))" 2>&1
        report_test $? "YAML syntax valid (Python): $workflow"
    fi
done
echo ""

# Test 2: Check for YAML document separator issues (the bug from PR #419)
echo "Test 2: YAML Document Separator Check"
echo "--------------------------------------"

for workflow in "${WORKFLOW_FILES[@]}"; do
    # Look for standalone --- in JavaScript string literals
    if grep -n "^---$" "$workflow" | grep -v "^1:---$" > /dev/null; then
        echo -e "${YELLOW}Warning${NC}: Found standalone --- on non-header lines in $workflow"
        grep -n "^---$" "$workflow" | grep -v "^1:---$"
        report_test 1 "No problematic YAML separators: $workflow"
    else
        report_test 0 "No problematic YAML separators: $workflow"
    fi
done
echo ""

# Test 3: Verify required permissions are declared
echo "Test 3: Permissions Declaration Check"
echo "--------------------------------------"

check_permission() {
    local file=$1
    local permission=$2
    if grep -q "^permissions:" "$file" && grep -A 10 "^permissions:" "$file" | grep -q "$permission"; then
        report_test 0 "$permission permission declared: $(basename $file)"
    else
        report_test 1 "$permission permission missing: $(basename $file)"
    fi
}

check_permission ".github/workflows/amber-auto-review.yml" "pull-requests: write"
check_permission ".github/workflows/amber-issue-handler.yml" "contents: write"
check_permission ".github/workflows/amber-issue-handler.yml" "issues: write"
check_permission ".github/workflows/amber-dependency-sync.yml" "contents: write"
echo ""

# Test 4: Verify workflow trigger conditions
echo "Test 4: Workflow Trigger Validation"
echo "------------------------------------"

# amber-auto-review should trigger on PR events
if grep -q "pull_request_target:" ".github/workflows/amber-auto-review.yml"; then
    report_test 0 "amber-auto-review triggers on pull_request_target"
else
    report_test 1 "amber-auto-review missing pull_request_target trigger"
fi

# amber-issue-handler should have proper label/comment filtering
if grep -q "amber:auto-fix\|amber:refactor\|amber:test-coverage" ".github/workflows/amber-issue-handler.yml"; then
    report_test 0 "amber-issue-handler has label filters"
else
    report_test 1 "amber-issue-handler missing label filters"
fi

# amber-dependency-sync should have schedule
if grep -q "schedule:" ".github/workflows/amber-dependency-sync.yml"; then
    report_test 0 "amber-dependency-sync has scheduled trigger"
else
    report_test 1 "amber-dependency-sync missing schedule"
fi
echo ""

# Test 5: Security - Check for secret exposure
echo "Test 5: Security Validation"
echo "----------------------------"

for workflow in "${WORKFLOW_FILES[@]}"; do
    # Check that secrets are properly referenced
    if grep -q "secrets\." "$workflow"; then
        report_test 0 "Uses GitHub secrets properly: $(basename $workflow)"
    else
        echo -e "${YELLOW}Info${NC}: No secrets used in $(basename $workflow)"
    fi

    # Check for hardcoded tokens (security issue)
    if grep -iE "(token|password|key):\s*['\"][a-zA-Z0-9]{20,}" "$workflow"; then
        report_test 1 "SECURITY: Hardcoded credentials found in $(basename $workflow)"
    else
        report_test 0 "No hardcoded credentials: $(basename $workflow)"
    fi
done
echo ""

# Test 6: Verify string concatenation fix (PR #419)
echo "Test 6: Template Literal Validation"
echo "------------------------------------"

# Check amber-auto-review.yml for proper string concatenation (not template literals with ---)
if grep -q "const transparencySection = " ".github/workflows/amber-auto-review.yml"; then
    # Count lines with template literals vs string concatenation
    template_literal_count=$(grep -c "const transparencySection = \`" ".github/workflows/amber-auto-review.yml" || echo 0)
    string_concat_count=$(grep -c "const transparencySection = '.*' +" ".github/workflows/amber-auto-review.yml" || echo 0)

    if [ "$string_concat_count" -gt 0 ] && [ "$template_literal_count" -eq 0 ]; then
        report_test 0 "Uses string concatenation (not template literals with ---)"
    else
        report_test 1 "Should use string concatenation to avoid YAML separator issues"
    fi
else
    echo -e "${YELLOW}Info${NC}: transparencySection not found in amber-auto-review.yml"
fi
echo ""

# Test 7: Verify workflow file references exist
echo "Test 7: File Reference Validation"
echo "----------------------------------"

# Check if referenced files in workflows exist
check_file_exists() {
    local ref_file=$1
    if [ -f "$ref_file" ]; then
        report_test 0 "Referenced file exists: $ref_file"
    else
        report_test 1 "Referenced file missing: $ref_file"
    fi
}

# Files referenced in amber workflows
check_file_exists "CLAUDE.md"
check_file_exists ".claude/context/backend-development.md"
check_file_exists ".claude/context/frontend-development.md"
check_file_exists ".claude/context/security-standards.md"
check_file_exists "agents/amber.md"
check_file_exists "scripts/sync-amber-dependencies.py"
echo ""

# Test 8: Validate GitHub Actions syntax
echo "Test 8: GitHub Actions Syntax Check"
echo "------------------------------------"

for workflow in "${WORKFLOW_FILES[@]}"; do
    # Check for common syntax patterns
    if grep -q "uses: actions/" "$workflow"; then
        report_test 0 "Uses standard GitHub actions: $(basename $workflow)"
    fi

    # Validate step structure (must have name and run/uses)
    step_count=$(grep -c "- name:" "$workflow")
    run_or_uses_count=$(grep -cE "(run:|uses:)" "$workflow")

    if [ "$run_or_uses_count" -ge "$step_count" ]; then
        report_test 0 "All steps have run/uses commands: $(basename $workflow)"
    else
        report_test 1 "Some steps missing run/uses: $(basename $workflow)"
    fi
done
echo ""

# Test 9: Verify error handling in workflows
echo "Test 9: Error Handling Validation"
echo "----------------------------------"

# Check for continue-on-error and proper failure handling
if grep -q "continue-on-error: true" ".github/workflows/amber-auto-review.yml"; then
    report_test 0 "amber-auto-review has error handling for non-critical steps"
fi

if grep -q "if: failure()" ".github/workflows/amber-issue-handler.yml"; then
    report_test 0 "amber-issue-handler has failure reporting"
else
    report_test 1 "amber-issue-handler missing failure reporting"
fi
echo ""

# Test 10: Validate constitution compliance checks
echo "Test 10: Constitution Compliance Check"
echo "---------------------------------------"

if grep -q "Validate constitution compliance" ".github/workflows/amber-dependency-sync.yml"; then
    report_test 0 "amber-dependency-sync validates constitution compliance"
else
    report_test 1 "amber-dependency-sync missing constitution validation"
fi
echo ""

# Summary
echo "======================================"
echo "Test Summary"
echo "======================================"
echo -e "Total tests run:    ${TESTS_RUN}"
echo -e "Tests passed:       ${GREEN}${TESTS_PASSED}${NC}"
echo -e "Tests failed:       ${RED}${TESTS_FAILED}${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ All Amber workflow validations passed!${NC}"
    exit 0
else
    echo -e "${RED}✗ Some validations failed. Review output above.${NC}"
    exit 1
fi
