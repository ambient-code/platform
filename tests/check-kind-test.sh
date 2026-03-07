#!/bin/bash
# Test: check-kind Makefile target offers dnf install on Fedora
#
# Validates three scenarios by manipulating PATH:
#   1. kind present    → exits 0
#   2. kind absent, dnf present → shows "sudo dnf install kind", exits 1
#   3. kind absent, dnf absent  → shows install URL, exits 1

set -uo pipefail

PASS=0
FAIL=0
MAKEFILE_DIR="$(cd "$(dirname "$0")/.." && pwd)"

run_test() {
    local name="$1" expected_exit="$2" expected_pattern="$3" path_override="$4"
    output=$(env PATH="$path_override" make -C "$MAKEFILE_DIR" check-kind 2>&1) || true
    actual_exit=${PIPESTATUS[0]:-$?}

    # Re-run to capture exit code properly (pipe eats it)
    env PATH="$path_override" make -C "$MAKEFILE_DIR" check-kind >/dev/null 2>&1
    actual_exit=$?

    local ok=true
    if [ "$actual_exit" -ne "$expected_exit" ]; then
        echo "FAIL: $name — expected exit $expected_exit, got $actual_exit"
        ok=false
    fi
    if [ -n "$expected_pattern" ] && ! echo "$output" | grep -q "$expected_pattern"; then
        echo "FAIL: $name — output missing pattern: $expected_pattern"
        echo "  got: $output"
        ok=false
    fi
    if $ok; then
        echo "PASS: $name"
        ((PASS++))
    else
        ((FAIL++))
    fi
}

# Build a minimal PATH with only coreutils (no kind, no dnf)
BARE_PATH=""
for d in /usr/bin /bin /usr/sbin /sbin; do
    [ -d "$d" ] && BARE_PATH="${BARE_PATH:+$BARE_PATH:}$d"
done

# Create temp dir with fake binaries
TMPDIR_TEST=$(mktemp -d)
trap 'rm -rf "$TMPDIR_TEST"' EXIT

# Fake kind
mkdir -p "$TMPDIR_TEST/with-kind"
printf '#!/bin/sh\nexit 0\n' > "$TMPDIR_TEST/with-kind/kind"
chmod +x "$TMPDIR_TEST/with-kind/kind"

# Fake dnf
mkdir -p "$TMPDIR_TEST/with-dnf"
printf '#!/bin/sh\nexit 0\n' > "$TMPDIR_TEST/with-dnf/dnf"
chmod +x "$TMPDIR_TEST/with-dnf/dnf"

echo "=== check-kind tests ==="

# 1. kind present → success
run_test "kind installed → exit 0" \
    0 "" \
    "$TMPDIR_TEST/with-kind:$BARE_PATH"

# 2. kind absent, dnf present → suggest dnf install (make returns 2 on recipe failure)
run_test "kind missing + dnf available → suggests dnf install" \
    2 "sudo dnf install kind" \
    "$TMPDIR_TEST/with-dnf:$BARE_PATH"

# 3. kind absent, dnf absent → show install URL (make returns 2 on recipe failure)
run_test "kind missing + no dnf → shows install URL" \
    2 "kind.sigs.k8s.io" \
    "$BARE_PATH"

echo ""
echo "Results: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ]
