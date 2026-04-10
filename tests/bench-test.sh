#!/usr/bin/env bash

# shellcheck shell=bash

set -euo pipefail

REPO_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$REPO_ROOT"

source "$REPO_ROOT/scripts/benchmarks/component-bench.sh"

TESTS_PASSED=0
TESTS_FAILED=0

pass() {
  printf '✓ %s\n' "$1"
  TESTS_PASSED=$((TESTS_PASSED + 1))
}

fail() {
  printf '✗ %s\n' "$1" >&2
  TESTS_FAILED=$((TESTS_FAILED + 1))
}

run_test() {
  local name=$1
  shift

  if "$@"; then
    pass "$name"
  else
    fail "$name"
  fi
}

test_bench_script_syntax() {
  bash -n scripts/benchmarks/component-bench.sh
}

test_bench_manifest_syntax() {
  bash -n scripts/benchmarks/bench-manifest.sh
}

test_bench_all_components_defined() {
  local component=""
  local key=""

  for component in "${BENCH_COMPONENTS[@]}"; do
    key=$(bench_component_key "$component")
    declare -f "bench_env_${key}" >/dev/null 2>&1
    declare -f "bench_preflight_${key}" >/dev/null 2>&1
    declare -f "bench_cold_${key}" >/dev/null 2>&1
    declare -f "bench_warm_${key}" >/dev/null 2>&1
    declare -f "bench_cleanup_${key}" >/dev/null 2>&1
  done
}

test_bench_dry_run() {
  bash scripts/benchmarks/component-bench.sh --help >/dev/null
}

make_mock_raw_file() {
  local raw_file=$1

  cat >"$raw_file" <<'EOF'
frontend	baseline	cold	1	success	42.1
frontend	baseline	cold	2	success	42.0
frontend	baseline	cold	3	success	42.2
frontend	candidate	cold	1	success	41.8
frontend	candidate	cold	2	success	41.9
frontend	candidate	cold	3	success	41.7
frontend	baseline	warm	1	success	8.2
frontend	baseline	warm	2	success	8.1
frontend	baseline	warm	3	success	8.3
frontend	candidate	warm	1	success	8.1
frontend	candidate	warm	2	success	8.0
frontend	candidate	warm	3	success	8.2
backend	baseline	cold	1	success	6.2
backend	baseline	cold	2	success	6.1
backend	baseline	cold	3	success	6.3
backend	candidate	cold	1	success	6.1
backend	candidate	cold	2	success	6.0
backend	candidate	cold	3	success	6.2
backend	baseline	warm	1	success	1.1
backend	baseline	warm	2	success	1.0
backend	baseline	warm	3	success	1.2
backend	candidate	warm	1	success	1.1
backend	candidate	warm	2	success	1.1
backend	candidate	warm	3	success	1.0
EOF
}

test_bench_report_outputs() {
  local temp_dir
  local raw_file

  temp_dir=$(mktemp -d)
  BENCH_REPORT_DIR="$temp_dir/reports"
  mkdir -p "$BENCH_REPORT_DIR/raw" "$BENCH_REPORT_DIR/logs"
  BENCH_REPEATS=3
  BENCH_MODE=both
  BASELINE_REF=abc1234
  CANDIDATE_REF=def5678
  SELECTED_COMPONENTS=(frontend backend)

  raw_file="$temp_dir/raw.tsv"
  make_mock_raw_file "$raw_file"
  bench_generate_reports "$raw_file"

  grep -q '^Results:$' "$(bench_human_report_file)"
  grep -q '✓ frontend' "$(bench_human_report_file)"
  head -n 1 "$(bench_tsv_report_file)" | grep -q $'^component\tscenario\tbaseline_s\tcandidate_s\tdelta_s\tdelta_pct\tstddev_s\tbudget_ok$'
  awk -F '\t' 'NR == 1 || NF == 8 { next } { exit 1 }' "$(bench_tsv_report_file)"

  if command -v python3 >/dev/null 2>&1; then
    python3 -m json.tool "$(bench_json_report_file)" >/dev/null
  elif command -v jq >/dev/null 2>&1; then
    jq . "$(bench_json_report_file)" >/dev/null
  else
    return 1
  fi

  rm -rf "$temp_dir"
}

test_bench_no_ansi_when_piped() {
  local temp_dir
  local raw_file

  temp_dir=$(mktemp -d)
  BENCH_REPORT_DIR="$temp_dir/reports"
  mkdir -p "$BENCH_REPORT_DIR/raw" "$BENCH_REPORT_DIR/logs"
  BENCH_REPEATS=3
  BENCH_MODE=cold
  BASELINE_REF=abc1234
  CANDIDATE_REF=def5678
  SELECTED_COMPONENTS=(frontend)
  CI_MODE=false
  BENCH_FORMAT=human

  raw_file="$temp_dir/raw.tsv"
  make_mock_raw_file "$raw_file"
  bench_generate_reports "$raw_file"

  if bench_emit_selected_format | perl -ne 'exit 1 if /\e\[/'; then
    rm -rf "$temp_dir"
    return 0
  fi

  rm -rf "$temp_dir"
  return 1
}

test_bench_report_dir_fallback() {
  local original_report_dir=$BENCH_REPORT_DIR

  BENCH_REPORT_DIR="/dev/null/benchmarks"
  bench_ensure_report_dir
  [[ -d "$BENCH_REPORT_DIR" ]]

  rm -rf "$BENCH_REPORT_DIR"
  BENCH_REPORT_DIR=$original_report_dir
}

test_bench_makefile_target() {
  make -n benchmark >/dev/null 2>&1
}

main() {
  run_test "benchmark script syntax" test_bench_script_syntax
  run_test "benchmark manifest syntax" test_bench_manifest_syntax
  run_test "benchmark component function coverage" test_bench_all_components_defined
  run_test "benchmark help output" test_bench_dry_run
  run_test "benchmark report outputs" test_bench_report_outputs
  run_test "benchmark no ANSI when piped" test_bench_no_ansi_when_piped
  run_test "benchmark report dir fallback" test_bench_report_dir_fallback
  run_test "benchmark make target syntax" test_bench_makefile_target

  printf '\nResults:\n'
  printf '  Passed: %s\n' "$TESTS_PASSED"
  printf '  Failed: %s\n' "$TESTS_FAILED"
  printf '  Total:  %s\n' $((TESTS_PASSED + TESTS_FAILED))

  [[ "$TESTS_FAILED" -eq 0 ]]
}

main "$@"
