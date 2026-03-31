#!/usr/bin/env bash

# shellcheck shell=bash

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "$SCRIPT_DIR/../.." && pwd)

COLOR_RESET=""
COLOR_BOLD=""
COLOR_GREEN=""
COLOR_YELLOW=""
COLOR_BLUE=""
COLOR_RED=""

CI_MODE=false
BENCH_FORMAT=""
BENCH_MODE=both
BENCH_REPEATS=""
BENCH_REPORT_DIR=""
BASELINE_REF=""
CANDIDATE_REF="HEAD"
BASELINE_LABEL="baseline"
CANDIDATE_LABEL="candidate"
BENCH_TMPDIR=""
SELECTED_COMPONENTS=()
READY_COMPONENTS=()
WORKTREE_PATHS=()
BENCH_BG_PIDS=()
BENCH_INTERRUPTED=false

bench_supports_color() {
  [[ -t 1 ]] && [[ "$CI_MODE" != true ]]
}

bench_init_colors() {
  if bench_supports_color; then
    COLOR_RESET=$(tput sgr0 2>/dev/null || printf '\033[0m')
    COLOR_BOLD=$(tput bold 2>/dev/null || printf '\033[1m')
    COLOR_GREEN=$(tput setaf 2 2>/dev/null || printf '\033[32m')
    COLOR_YELLOW=$(tput setaf 3 2>/dev/null || printf '\033[33m')
    COLOR_BLUE=$(tput setaf 4 2>/dev/null || printf '\033[34m')
    COLOR_RED=$(tput setaf 1 2>/dev/null || printf '\033[31m')
  fi
}

bench_timestamp() {
  date -u +%Y-%m-%dT%H:%M:%SZ
}

bench_now_seconds() {
  perl -MTime::HiRes=time -e 'printf "%.6f\n", time' 2>/dev/null || \
    python3 -c 'import time; print(f"{time.time():.6f}")' 2>/dev/null || \
    date +%s
}

bench_elapsed_seconds() {
  local start=$1
  local end=$2
  awk -v start="$start" -v end="$end" 'BEGIN { printf "%.3f", (end - start) }'
}

bench_format_seconds() {
  local value=${1:-0}
  awk -v value="$value" 'BEGIN { printf "%.1f", value }'
}

bench_log_info() {
  if [[ "$CI_MODE" == true ]]; then
    printf '[%s] %sℹ%s %s\n' "$(bench_timestamp)" "$COLOR_BLUE" "$COLOR_RESET" "$*" >&2
  else
    printf '%sℹ%s %s\n' "$COLOR_BLUE" "$COLOR_RESET" "$*" >&2
  fi
}

bench_log_start() {
  local component=$1
  local scenario=$2
  local run_index=$3
  local total_runs=$4
  local message=$5

  if [[ "$CI_MODE" == true ]]; then
    printf '[%s] %s▶%s %-14s %s run %s/%s %s\n' \
      "$(bench_timestamp)" "$COLOR_BLUE" "$COLOR_RESET" \
      "$component" "$scenario" "$run_index" "$total_runs" "$message" >&2
  else
    printf '%s▶%s %-14s %s run %s/%s' \
      "$COLOR_BLUE" "$COLOR_RESET" \
      "$component" "$scenario" "$run_index" "$total_runs" >&2
  fi
}

bench_log_dot() {
  if [[ "$CI_MODE" != true ]]; then
    printf '.' >&2
  fi
}

bench_log_success() {
  local component=$1
  local scenario=$2
  local run_index=$3
  local total_runs=$4
  local elapsed=$5

  if [[ "$CI_MODE" == true ]]; then
    printf '[%s] %s✓%s %-14s %s run %s/%s  %ss\n' \
      "$(bench_timestamp)" "$COLOR_GREEN" "$COLOR_RESET" \
      "$component" "$scenario" "$run_index" "$total_runs" "$(bench_format_seconds "$elapsed")" >&2
  else
    printf ' %s✓%s %ss\n' \
      "$COLOR_GREEN" "$COLOR_RESET" \
      "$(bench_format_seconds "$elapsed")" >&2
  fi
}

bench_log_error() {
  local message=$1

  if [[ "$CI_MODE" == true ]]; then
    printf '[%s] %s✗%s %s\n' "$(bench_timestamp)" "$COLOR_RED" "$COLOR_RESET" "$message" >&2
  else
    printf '%s✗%s %s\n' "$COLOR_RED" "$COLOR_RESET" "$message" >&2
  fi
}

bench_log_error_inline() {
  local message=$1

  if [[ "$CI_MODE" == true ]]; then
    printf '[%s] %s✗%s %s\n' "$(bench_timestamp)" "$COLOR_RED" "$COLOR_RESET" "$message" >&2
  else
    printf ' %s✗%s %s\n' "$COLOR_RED" "$COLOR_RESET" "$message" >&2
  fi
}

bench_log_warning() {
  local message=$1

  if [[ "$CI_MODE" == true ]]; then
    printf '[%s] %s⚠%s %s\n' "$(bench_timestamp)" "$COLOR_YELLOW" "$COLOR_RESET" "$message" >&2
  else
    printf '%s⚠%s %s\n' "$COLOR_YELLOW" "$COLOR_RESET" "$message" >&2
  fi
}

bench_component_key() {
  echo "${1//-/_}"
}

bench_pick_port() {
  local component=$1
  local run_id=$2
  local ref_name=$3
  local index=0
  local current
  local ref_offset=0

  for current in "${BENCH_COMPONENTS[@]}"; do
    if [[ "$current" == "$component" ]]; then
      break
    fi
    index=$((index + 1))
  done

  if [[ "$ref_name" == "candidate" ]]; then
    ref_offset=100
  fi

  echo $((43000 + (index * 20) + ref_offset + run_id))
}

bench_wait_for_pattern() {
  local log_file=$1
  local pattern=$2
  local timeout_seconds=$3
  local pid=$4
  local start
  local now
  local elapsed

  start=$(bench_now_seconds)

  while true; do
    if [[ -f "$log_file" ]] && grep -Eq "$pattern" "$log_file"; then
      return 0
    fi

    if ! kill -0 "$pid" >/dev/null 2>&1; then
      break
    fi

    now=$(bench_now_seconds)
    elapsed=$(bench_elapsed_seconds "$start" "$now")
    if awk -v elapsed="$elapsed" -v timeout="$timeout_seconds" 'BEGIN { exit !(elapsed >= timeout) }'; then
      return 1
    fi

    sleep 1
  done

  [[ -f "$log_file" ]] && grep -Eq "$pattern" "$log_file"
}

bench_kill_pid() {
  local pid=$1

  if kill -0 "$pid" >/dev/null 2>&1; then
    kill "$pid" >/dev/null 2>&1 || true
    wait "$pid" >/dev/null 2>&1 || true
  fi
}

bench_usage() {
  cat <<'EOF'
scripts/benchmarks/component-bench.sh [OPTIONS]

  --baseline-ref REF      Git ref for baseline (default: merge-base with origin/main)
  --candidate-ref REF     Git ref for candidate (default: HEAD)
  --components LIST       Comma-separated component list (default: all)
  --mode cold|warm|both   Which scenarios to benchmark (default: both)
  --repeats N             Runs per scenario (default: 3 locally, 5 in CI)
  --format human|tsv|json Output format (default: human if TTY, tsv if piped)
  --report-dir DIR        Output directory (default: reports/benchmarks)
  --ci                    CI mode: plain progress logs, 5 repeats by default
  --help                  Show usage

Recommended usage:

  Human local summary:
    make benchmark

  Agent / automation fast path:
    make benchmark FORMAT=tsv COMPONENT=backend MODE=warm REPEATS=1

  First-pass exploratory run:
    benchmark one component first; avoid full-suite warm or cold sweeps
    unless you explicitly need the entire matrix

  Output guidance:
    human  -> interactive terminal
    tsv    -> agents, pipes, automation
    json   -> downstream tooling / archival
EOF
}

bench_default_baseline_ref() {
  local merge_base=""

  if git -C "$REPO_ROOT" rev-parse --verify origin/main >/dev/null 2>&1; then
    merge_base=$(git -C "$REPO_ROOT" merge-base HEAD origin/main 2>/dev/null || true)
    if [[ -n "$merge_base" ]]; then
      echo "$merge_base"
    else
      echo "origin/main"
    fi
    return
  fi

  echo "main"
}

bench_validate_format() {
  case "$BENCH_FORMAT" in
    human|tsv|json) ;;
    *)
      bench_log_error "Invalid format '$BENCH_FORMAT' (use human, tsv, or json)"
      exit 1
      ;;
  esac
}

bench_validate_mode() {
  case "$BENCH_MODE" in
    cold|warm|both) ;;
    *)
      bench_log_error "Invalid mode '$BENCH_MODE' (use cold, warm, or both)"
      exit 1
      ;;
  esac
}

bench_validate_ref() {
  local ref=$1

  if ! git -C "$REPO_ROOT" rev-parse --verify "$ref^{commit}" >/dev/null 2>&1; then
    bench_log_error "Git ref '$ref' does not resolve to a commit"
    exit 1
  fi
}

bench_component_exists() {
  local wanted=$1
  local component

  for component in "${BENCH_COMPONENTS[@]}"; do
    if [[ "$component" == "$wanted" ]]; then
      return 0
    fi
  done

  return 1
}

bench_require_command() {
  local command_name=$1

  if ! command -v "$command_name" >/dev/null 2>&1; then
    printf '%s\n' "required command '$command_name' is not installed"
    return 1
  fi
}

bench_require_node_version() {
  local min_major=$1
  local min_minor=$2
  local version major minor

  version=$(node -v 2>/dev/null | sed 's/^v//')
  major=$(printf '%s' "$version" | cut -d. -f1)
  minor=$(printf '%s' "$version" | cut -d. -f2)

  if ! awk -v major="$major" -v minor="$minor" -v min_major="$min_major" -v min_minor="$min_minor" \
    'BEGIN { exit !((major > min_major) || (major == min_major && minor >= min_minor)) }'; then
    printf '%s\n' "node v$version found; need >= ${min_major}.${min_minor}"
    return 1
  fi
}

bench_require_go_version() {
  local min_major=$1
  local min_minor=$2
  local version major minor

  version=$(go env GOVERSION 2>/dev/null | sed 's/^go//')
  major=$(printf '%s' "$version" | cut -d. -f1)
  minor=$(printf '%s' "$version" | cut -d. -f2)

  if ! awk -v major="$major" -v minor="$minor" -v min_major="$min_major" -v min_minor="$min_minor" \
    'BEGIN { exit !((major > min_major) || (major == min_major && minor >= min_minor)) }'; then
    printf '%s\n' "go $version found; need >= ${min_major}.${min_minor}"
    return 1
  fi
}

bench_require_python3() {
  bench_require_command python3
}

bench_require_python_version() {
  local min_major=$1
  local min_minor=$2
  local version major minor

  version=$(python3 - <<'EOF'
import sys
print(f"{sys.version_info.major}.{sys.version_info.minor}.{sys.version_info.micro}")
EOF
)
  major=$(printf '%s' "$version" | cut -d. -f1)
  minor=$(printf '%s' "$version" | cut -d. -f2)

  if ! awk -v major="$major" -v minor="$minor" -v min_major="$min_major" -v min_minor="$min_minor" \
    'BEGIN { exit !((major > min_major) || (major == min_major && minor >= min_minor)) }'; then
    printf '%s\n' "python $version found; need >= ${min_major}.${min_minor}"
    return 1
  fi
}

bench_prepare_component_env() {
  local component=$1
  local cache_root=$2
  local function_name="bench_env_$(bench_component_key "$component")"

  if declare -f "$function_name" >/dev/null 2>&1; then
    "$function_name" "$cache_root"
  fi
}

bench_preflight_component() {
  local component=$1
  local function_name="bench_preflight_$(bench_component_key "$component")"

  if declare -f "$function_name" >/dev/null 2>&1; then
    "$function_name"
  fi
}

bench_parse_components() {
  local raw=${1:-all}
  local item=""
  local trimmed=""

  SELECTED_COMPONENTS=()

  if [[ "$raw" == "all" || -z "$raw" ]]; then
    for item in "${BENCH_COMPONENTS[@]}"; do
      SELECTED_COMPONENTS+=("$item")
    done
    return
  fi

  OLD_IFS=$IFS
  IFS=,
  for item in $raw; do
    trimmed=$(printf '%s' "$item" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
    [[ -z "$trimmed" ]] && continue
    if ! bench_component_exists "$trimmed"; then
      bench_log_error "Unknown component '$trimmed' (valid: ${BENCH_COMPONENTS[*]})"
      exit 1
    fi
    SELECTED_COMPONENTS+=("$trimmed")
  done
  IFS=$OLD_IFS

  if [[ ${#SELECTED_COMPONENTS[@]} -eq 0 ]]; then
    bench_log_error "No components selected"
    exit 1
  fi
}

bench_scenarios_for_mode() {
  case "$BENCH_MODE" in
    cold) echo "cold" ;;
    warm) echo "warm" ;;
    both)
      echo "cold"
      echo "warm"
      ;;
  esac
}

bench_has_scenario() {
  local wanted=$1
  local scenario

  while IFS= read -r scenario; do
    if [[ "$scenario" == "$wanted" ]]; then
      return 0
    fi
  done < <(bench_scenarios_for_mode)

  return 1
}

bench_result_file_for_component() {
  local component=$1
  echo "$BENCH_REPORT_DIR/raw/${component}.tsv"
}

bench_combined_raw_file() {
  echo "$BENCH_REPORT_DIR/results.raw.tsv"
}

bench_human_report_file() {
  echo "$BENCH_REPORT_DIR/results.human.txt"
}

bench_tsv_report_file() {
  echo "$BENCH_REPORT_DIR/results.tsv"
}

bench_json_report_file() {
  echo "$BENCH_REPORT_DIR/results.json"
}

bench_safe_append() {
  local file=$1
  shift

  if ! printf '%s\n' "$*" >>"$file" 2>/dev/null; then
    bench_log_warning "Could not write to $file"
  fi
}

bench_safe_redirect() {
  local file=$1

  if ! touch "$file" 2>/dev/null; then
    file="$BENCH_TMPDIR/fallback-$(basename "$file")"
    touch "$file" 2>/dev/null || file="/dev/null"
    bench_log_warning "Log redirect failed; using $file"
  fi
  echo "$file"
}

bench_record_result() {
  local component=$1
  local ref_name=$2
  local scenario=$3
  local run_index=$4
  local status=$5
  local elapsed=$6
  local message=${7:-}
  local file

  message=$(printf '%s' "$message" | tr '\t\r\n' '   ')
  file=$(bench_result_file_for_component "$component")
  bench_safe_append "$file" "$(printf '%s\t%s\t%s\t%s\t%s\t%s\t%s' \
    "$component" "$ref_name" "$scenario" "$run_index" "$status" "$elapsed" "$message")"
}

bench_invoke_function() {
  local component=$1
  local stage=$2
  local worktree_dir=$3
  local cache_root=$4
  local run_index=$5
  local ref_name=$6
  local function_name

  function_name="bench_${stage}_$(bench_component_key "$component")"
  if ! declare -f "$function_name" >/dev/null 2>&1; then
    bench_log_error "Missing function '$function_name'"
    return 1
  fi

  "$function_name" "$worktree_dir" "$cache_root" "$run_index" "$ref_name"
}

bench_prepare_cache_root() {
  local component=$1
  local ref_name=$2
  local run_index=$3
  local cache_root="$BENCH_TMPDIR/cache/$(bench_component_key "$component")/$ref_name/run-$run_index"

  rm -rf "$cache_root"
  mkdir -p "$cache_root"
  echo "$cache_root"
}

bench_run_timed_step() {
  local component=$1
  local ref_name=$2
  local scenario=$3
  local run_index=$4
  local total_runs=$5
  local worktree_dir=$6
  local cache_root=$7
  local log_file=$8
  local start end elapsed

  local dot_pid=""

  bench_prepare_component_env "$component" "$cache_root"
  log_file=$(bench_safe_redirect "$log_file")
  bench_log_start "$component" "$scenario/$ref_name" "$run_index" "$total_runs" ""

  if [[ "$CI_MODE" != true ]]; then
    ( while true; do sleep 3; bench_log_dot; done ) &
    dot_pid=$!
  fi

  start=$(bench_now_seconds)
  if bench_invoke_function "$component" "$scenario" "$worktree_dir" "$cache_root" "$run_index" "$ref_name" >"$log_file" 2>&1; then
    end=$(bench_now_seconds)
    elapsed=$(bench_elapsed_seconds "$start" "$end")
    [[ -n "$dot_pid" ]] && kill "$dot_pid" 2>/dev/null; wait "$dot_pid" 2>/dev/null || true
    bench_log_success "$component" "$scenario/$ref_name" "$run_index" "$total_runs" "$elapsed"
    printf '%s\n' "$elapsed"
    return 0
  fi

  [[ -n "$dot_pid" ]] && kill "$dot_pid" 2>/dev/null; wait "$dot_pid" 2>/dev/null || true
  bench_log_error_inline "$component $scenario/$ref_name run $run_index/$total_runs failed"
  return 1
}

bench_run_component_ref() {
  local component=$1
  local ref_name=$2
  local worktree_dir=$3
  local repeat_index
  local cache_root
  local log_file
  local elapsed

  for repeat_index in $(seq 1 "$BENCH_REPEATS"); do
    cache_root=$(bench_prepare_cache_root "$component" "$ref_name" "$repeat_index")

    bench_invoke_function "$component" cleanup "$worktree_dir" "$cache_root" "$repeat_index" "$ref_name" >/dev/null 2>&1 || true

    if [[ "$BENCH_MODE" == "warm" ]]; then
      log_file=$(bench_safe_redirect "$BENCH_REPORT_DIR/logs/${component}-${ref_name}-setup-${repeat_index}.log")
      bench_prepare_component_env "$component" "$cache_root"
      if ! bench_invoke_function "$component" cold "$worktree_dir" "$cache_root" "$repeat_index" "$ref_name" >"$log_file" 2>&1; then
        bench_record_result "$component" "$ref_name" "warm" "$repeat_index" "error" "0" "warm setup failed ($(basename "$log_file"))"
        return 1
      fi
    fi

    if bench_has_scenario cold; then
      log_file="$BENCH_REPORT_DIR/logs/${component}-${ref_name}-cold-${repeat_index}.log"
      if elapsed=$(bench_run_timed_step "$component" "$ref_name" "cold" "$repeat_index" "$BENCH_REPEATS" "$worktree_dir" "$cache_root" "$log_file"); then
        bench_record_result "$component" "$ref_name" "cold" "$repeat_index" "success" "$elapsed" ""
      else
        bench_record_result "$component" "$ref_name" "cold" "$repeat_index" "error" "0" "cold failed ($(basename "$log_file"))"
        return 1
      fi
    fi

    if bench_has_scenario warm; then
      log_file="$BENCH_REPORT_DIR/logs/${component}-${ref_name}-warm-${repeat_index}.log"
      if elapsed=$(bench_run_timed_step "$component" "$ref_name" "warm" "$repeat_index" "$BENCH_REPEATS" "$worktree_dir" "$cache_root" "$log_file"); then
        bench_record_result "$component" "$ref_name" "warm" "$repeat_index" "success" "$elapsed" ""
      else
        bench_record_result "$component" "$ref_name" "warm" "$repeat_index" "error" "0" "warm failed ($(basename "$log_file"))"
        return 1
      fi
    fi
  done

  return 0
}

bench_component_worktree_dir() {
  local component=$1
  local ref_name=$2

  echo "$BENCH_TMPDIR/worktrees/$(bench_component_key "$component")-$ref_name"
}

bench_ensure_report_dir() {
  if mkdir -p "$BENCH_REPORT_DIR/raw" "$BENCH_REPORT_DIR/logs" >/dev/null 2>&1; then
    return
  fi

  local fallback_dir
  fallback_dir=$(mktemp -d)
  bench_log_warning "Could not write to '$BENCH_REPORT_DIR'; using '$fallback_dir' instead"
  BENCH_REPORT_DIR="$fallback_dir"
  mkdir -p "$BENCH_REPORT_DIR/raw" "$BENCH_REPORT_DIR/logs"
}

bench_setup_component_worktrees() {
  local component=$1
  local baseline_dir
  local candidate_dir

  baseline_dir=$(bench_component_worktree_dir "$component" "$BASELINE_LABEL")
  candidate_dir=$(bench_component_worktree_dir "$component" "$CANDIDATE_LABEL")

  mkdir -p "$BENCH_TMPDIR/worktrees"

  git -C "$REPO_ROOT" worktree add --detach "$baseline_dir" "$BASELINE_REF" >/dev/null 2>&1
  git -C "$REPO_ROOT" worktree add --detach "$candidate_dir" "$CANDIDATE_REF" >/dev/null 2>&1

  WORKTREE_PATHS+=("$baseline_dir" "$candidate_dir")
}

bench_run_component_job() {
  local component=$1
  local baseline_dir
  local candidate_dir

  baseline_dir=$(bench_component_worktree_dir "$component" "$BASELINE_LABEL")
  candidate_dir=$(bench_component_worktree_dir "$component" "$CANDIDATE_LABEL")

  bench_run_component_ref "$component" "$BASELINE_LABEL" "$baseline_dir"
  bench_run_component_ref "$component" "$CANDIDATE_LABEL" "$candidate_dir"
}

bench_record_component_preflight_error() {
  local component=$1
  local message=$2

  bench_log_error "$component preflight failed: $message"
  if bench_has_scenario cold; then
    bench_record_result "$component" "$BASELINE_LABEL" "cold" "1" "error" "0" "$message"
    bench_record_result "$component" "$CANDIDATE_LABEL" "cold" "1" "error" "0" "$message"
  fi
  if bench_has_scenario warm; then
    bench_record_result "$component" "$BASELINE_LABEL" "warm" "1" "error" "0" "$message"
    bench_record_result "$component" "$CANDIDATE_LABEL" "warm" "1" "error" "0" "$message"
  fi
}

bench_preflight_selected_components() {
  local component
  local message
  local status=0

  READY_COMPONENTS=()

  for component in "${SELECTED_COMPONENTS[@]}"; do
    : >"$(bench_result_file_for_component "$component")"

    if bench_preflight_component "$component" >"$BENCH_REPORT_DIR/logs/${component}-preflight.log" 2>&1; then
      READY_COMPONENTS+=("$component")
    else
      message=$(tr '\n' ' ' <"$BENCH_REPORT_DIR/logs/${component}-preflight.log" | sed 's/[[:space:]]*$//')
      bench_record_component_preflight_error "$component" "$message"
      status=1
    fi
  done

  return "$status"
}

bench_gather_runs() {
  local raw_file=$1
  local component=$2
  local ref_name=$3
  local scenario=$4

  awk -F '\t' -v component="$component" -v ref_name="$ref_name" -v scenario="$scenario" \
    '$1 == component && $2 == ref_name && $3 == scenario && $5 == "success" { print $6 }' "$raw_file"
}

bench_has_errors() {
  local raw_file=$1
  local component=$2
  local scenario=$3

  awk -F '\t' -v component="$component" -v scenario="$scenario" \
    '$1 == component && $3 == scenario && $5 != "success" { found = 1 } END { exit !found }' "$raw_file"
}

bench_success_count() {
  local raw_file=$1
  local component=$2
  local ref_name=$3
  local scenario=$4

  awk -F '\t' -v component="$component" -v ref_name="$ref_name" -v scenario="$scenario" \
    '$1 == component && $2 == ref_name && $3 == scenario && $5 == "success" { count++ } END { print count + 0 }' "$raw_file"
}

bench_calc_median() {
  if [[ $# -eq 0 ]]; then
    echo ""
    return
  fi

  printf '%s\n' "$@" | sort -n | awk '
    {
      values[NR] = $1
    }
    END {
      if (NR == 0) {
        exit
      }
      if (NR % 2 == 1) {
        printf "%.1f", values[(NR + 1) / 2]
      } else {
        printf "%.1f", (values[NR / 2] + values[(NR / 2) + 1]) / 2
      }
    }'
}

bench_calc_stddev() {
  if [[ $# -eq 0 ]]; then
    echo ""
    return
  fi

  printf '%s\n' "$@" | awk '
    {
      values[NR] = $1
      sum += $1
    }
    END {
      if (NR == 0) {
        exit
      }
      mean = sum / NR
      for (i = 1; i <= NR; i++) {
        delta = values[i] - mean
        variance += delta * delta
      }
      printf "%.1f", sqrt(variance / NR)
    }'
}

bench_join_json_runs() {
  local first=true
  local value

  for value in "$@"; do
    if [[ "$first" == true ]]; then
      first=false
    else
      printf ', '
    fi
    printf '%s' "$value"
  done
}

bench_component_complete() {
  local raw_file=$1
  local component=$2
  local scenario
  local ref_name
  local count

  while IFS= read -r scenario; do
    for ref_name in "$BASELINE_LABEL" "$CANDIDATE_LABEL"; do
      count=$(bench_success_count "$raw_file" "$component" "$ref_name" "$scenario")
      if [[ "$count" -ne "$BENCH_REPEATS" ]]; then
        return 1
      fi
    done
  done < <(bench_scenarios_for_mode)

  return 0
}

bench_generate_human_report() {
  local raw_file=$1
  local human_file
  local scenario
  local component
  local baseline_runs=()
  local candidate_runs=()
  local line
  local baseline_median
  local candidate_median
  local candidate_stddev
  local delta_s
  local delta_pct
  local budget_ok
  local passed=0
  local failed=0

  human_file=$(bench_human_report_file)
  : >"$human_file"

  {
    printf '═══════════════════════════════════════════\n'
    printf '  Component Benchmark Summary\n'
    printf '═══════════════════════════════════════════\n'
    printf '\n'
    printf 'Baseline: %s  Candidate: %s\n' "$BASELINE_REF" "$CANDIDATE_REF"
    printf 'Platform: %s/%s    Repeats: %s   Date: %s\n' "$(uname -s | tr '[:upper:]' '[:lower:]')" "$(uname -m)" "$BENCH_REPEATS" "$(bench_timestamp)"
    printf '\n'
  } >>"$human_file"

  while IFS= read -r scenario; do
    if [[ "$scenario" == "cold" ]]; then
      printf 'Cold Install (new contributor path)\n' >>"$human_file"
    else
      printf 'Warm Rebuild (incremental, after source touch)\n' >>"$human_file"
    fi

    for component in "${SELECTED_COMPONENTS[@]}"; do
      if bench_has_errors "$raw_file" "$component" "$scenario"; then
        printf '  ✗ %-14s failed\n' "$component" >>"$human_file"
        continue
      fi

      baseline_runs=()
      candidate_runs=()
      while IFS= read -r line; do
        [[ -n "$line" ]] && baseline_runs+=("$line")
      done < <(bench_gather_runs "$raw_file" "$component" "$BASELINE_LABEL" "$scenario")
      while IFS= read -r line; do
        [[ -n "$line" ]] && candidate_runs+=("$line")
      done < <(bench_gather_runs "$raw_file" "$component" "$CANDIDATE_LABEL" "$scenario")

      if [[ ${#baseline_runs[@]} -eq 0 || ${#candidate_runs[@]} -eq 0 ]]; then
        printf '  ✗ %-14s incomplete\n' "$component" >>"$human_file"
        continue
      fi

      baseline_median=$(bench_calc_median "${baseline_runs[@]}")
      candidate_median=$(bench_calc_median "${candidate_runs[@]}")
      candidate_stddev=$(bench_calc_stddev "${candidate_runs[@]}")
      delta_s=$(awk -v base="$baseline_median" -v cand="$candidate_median" 'BEGIN { printf "%.1f", cand - base }')
      delta_pct=$(awk -v base="$baseline_median" -v cand="$candidate_median" 'BEGIN { if (base == 0) printf "0.0"; else printf "%.1f", ((cand - base) / base) * 100 }')
      printf '  ✓ %-14s %5ss → %5ss  %5ss (%5s%%)  stddev %ss\n' \
        "$component" "$baseline_median" "$candidate_median" "$delta_s" "$delta_pct" "$candidate_stddev" >>"$human_file"
    done

    printf '\n' >>"$human_file"
  done < <(bench_scenarios_for_mode)

  if bench_has_scenario cold; then
    printf '60s Budget (cold install)\n' >>"$human_file"
    for component in "${SELECTED_COMPONENTS[@]}"; do
      baseline_runs=()
      candidate_runs=()
      while IFS= read -r line; do
        [[ -n "$line" ]] && candidate_runs+=("$line")
      done < <(bench_gather_runs "$raw_file" "$component" "$CANDIDATE_LABEL" "cold")
      if [[ ${#candidate_runs[@]} -eq 0 ]]; then
        printf '  ✗ %-9s unavailable\n' "$component" >>"$human_file"
        continue
      fi

      candidate_median=$(bench_calc_median "${candidate_runs[@]}")
      budget_ok=$(awk -v value="$candidate_median" 'BEGIN { if (value <= 60.0) print "true"; else print "false" }')
      if [[ "$budget_ok" == "true" ]]; then
        printf '  ✓ %-9s %5ss (%ss headroom)\n' \
          "$component" "$candidate_median" "$(awk -v value="$candidate_median" 'BEGIN { printf "%.1f", 60.0 - value }')" >>"$human_file"
      else
        printf '  ✗ %-9s %5ss (%ss over budget)\n' \
          "$component" "$candidate_median" "$(awk -v value="$candidate_median" 'BEGIN { printf "%.1f", value - 60.0 }')" >>"$human_file"
      fi
    done
    printf '\n' >>"$human_file"
  fi

  if bench_has_scenario cold; then
    printf 'Cold Install (visual, 60s budget)\n' >>"$human_file"
    for component in "${SELECTED_COMPONENTS[@]}"; do
      candidate_runs=()
      while IFS= read -r line; do
        [[ -n "$line" ]] && candidate_runs+=("$line")
      done < <(bench_gather_runs "$raw_file" "$component" "$CANDIDATE_LABEL" "cold")
      if [[ ${#candidate_runs[@]} -eq 0 ]]; then
        printf '  %-14s  --\n' "$component" >>"$human_file"
        continue
      fi
      candidate_median=$(bench_calc_median "${candidate_runs[@]}")
      local bar_len
      bar_len=$(awk -v value="$candidate_median" 'BEGIN { v = int(value / 2); if (v < 1) v = 1; if (v > 30) v = 30; printf "%d", v }')
      local bar=""
      local i
      for (( i=0; i<bar_len; i++ )); do bar="${bar}#"; done
      local marker=""
      if awk -v value="$candidate_median" 'BEGIN { exit !(value > 60.0) }'; then
        marker=" OVER"
      fi
      printf '  %-14s %s %ss%s\n' "$component" "$bar" "$candidate_median" "$marker" >>"$human_file"
    done
    printf '  %s\n' "|----|----|----|----|----|----| 60s" >>"$human_file"
    printf '\n' >>"$human_file"
  fi

  for component in "${SELECTED_COMPONENTS[@]}"; do
    if bench_component_complete "$raw_file" "$component"; then
      passed=$((passed + 1))
    else
      failed=$((failed + 1))
    fi
  done

  {
    printf 'Results:\n'
    printf '  Passed: %s\n' "$passed"
    printf '  Failed: %s\n' "$failed"
    printf '  Total:  %s\n' $((passed + failed))
  } >>"$human_file"
}

bench_generate_tsv_report() {
  local raw_file=$1
  local tsv_file
  local scenario
  local component
  local baseline_runs=()
  local candidate_runs=()
  local line
  local baseline_median
  local candidate_median
  local candidate_stddev
  local delta_s
  local delta_pct
  local budget_ok="-"

  tsv_file=$(bench_tsv_report_file)
  printf 'component\tscenario\tbaseline_s\tcandidate_s\tdelta_s\tdelta_pct\tstddev_s\tbudget_ok\n' >"$tsv_file"

  while IFS= read -r scenario; do
    for component in "${SELECTED_COMPONENTS[@]}"; do
      baseline_runs=()
      candidate_runs=()
      while IFS= read -r line; do
        [[ -n "$line" ]] && baseline_runs+=("$line")
      done < <(bench_gather_runs "$raw_file" "$component" "$BASELINE_LABEL" "$scenario")
      while IFS= read -r line; do
        [[ -n "$line" ]] && candidate_runs+=("$line")
      done < <(bench_gather_runs "$raw_file" "$component" "$CANDIDATE_LABEL" "$scenario")

      if [[ ${#baseline_runs[@]} -eq 0 || ${#candidate_runs[@]} -eq 0 ]]; then
        continue
      fi

      baseline_median=$(bench_calc_median "${baseline_runs[@]}")
      candidate_median=$(bench_calc_median "${candidate_runs[@]}")
      candidate_stddev=$(bench_calc_stddev "${candidate_runs[@]}")
      delta_s=$(awk -v base="$baseline_median" -v cand="$candidate_median" 'BEGIN { printf "%.1f", cand - base }')
      delta_pct=$(awk -v base="$baseline_median" -v cand="$candidate_median" 'BEGIN { if (base == 0) printf "0.0"; else printf "%.1f", ((cand - base) / base) * 100 }')

      if [[ "$scenario" == "cold" ]]; then
        budget_ok=$(awk -v value="$candidate_median" 'BEGIN { if (value <= 60.0) print "true"; else print "false" }')
      else
        budget_ok="-"
      fi

      printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' \
        "$component" "$scenario" "$baseline_median" "$candidate_median" "$delta_s" "$delta_pct" "$candidate_stddev" "$budget_ok" >>"$tsv_file"
    done
  done < <(bench_scenarios_for_mode)
}

bench_generate_json_report() {
  local raw_file=$1
  local json_file
  local component
  local scenario
  local first_component=true
  local first_scenario
  local baseline_runs=()
  local candidate_runs=()
  local line
  local baseline_median
  local candidate_median
  local baseline_stddev
  local candidate_stddev
  local delta_s
  local delta_pct
  local budget_ok

  json_file=$(bench_json_report_file)
  : >"$json_file"

  {
    printf '{\n'
    printf '  "metadata": {\n'
    printf '    "baseline_ref": "%s",\n' "$BASELINE_REF"
    printf '    "candidate_ref": "%s",\n' "$CANDIDATE_REF"
    printf '    "platform": "%s/%s",\n' "$(uname -s | tr '[:upper:]' '[:lower:]')" "$(uname -m)"
    printf '    "repeats": %s,\n' "$BENCH_REPEATS"
    printf '    "date": "%s"\n' "$(bench_timestamp)"
    printf '  },\n'
    printf '  "components": {\n'
  } >>"$json_file"

  for component in "${SELECTED_COMPONENTS[@]}"; do
    if [[ "$first_component" == true ]]; then
      first_component=false
    else
      printf ',\n' >>"$json_file"
    fi

    printf '    "%s": {' "$component" >>"$json_file"
    first_scenario=true

    while IFS= read -r scenario; do
      baseline_runs=()
      candidate_runs=()
      while IFS= read -r line; do
        [[ -n "$line" ]] && baseline_runs+=("$line")
      done < <(bench_gather_runs "$raw_file" "$component" "$BASELINE_LABEL" "$scenario")
      while IFS= read -r line; do
        [[ -n "$line" ]] && candidate_runs+=("$line")
      done < <(bench_gather_runs "$raw_file" "$component" "$CANDIDATE_LABEL" "$scenario")

      if [[ ${#baseline_runs[@]} -eq 0 || ${#candidate_runs[@]} -eq 0 ]]; then
        continue
      fi

      baseline_median=$(bench_calc_median "${baseline_runs[@]}")
      candidate_median=$(bench_calc_median "${candidate_runs[@]}")
      baseline_stddev=$(bench_calc_stddev "${baseline_runs[@]}")
      candidate_stddev=$(bench_calc_stddev "${candidate_runs[@]}")
      delta_s=$(awk -v base="$baseline_median" -v cand="$candidate_median" 'BEGIN { printf "%.1f", cand - base }')
      delta_pct=$(awk -v base="$baseline_median" -v cand="$candidate_median" 'BEGIN { if (base == 0) printf "0.0"; else printf "%.1f", ((cand - base) / base) * 100 }')
      budget_ok="null"
      if [[ "$scenario" == "cold" ]]; then
        if awk -v value="$candidate_median" 'BEGIN { exit !(value <= 60.0) }'; then
          budget_ok="true"
        else
          budget_ok="false"
        fi
      fi

      if [[ "$first_scenario" == true ]]; then
        printf '\n' >>"$json_file"
        first_scenario=false
      else
        printf ',\n' >>"$json_file"
      fi

      {
        printf '      "%s": {\n' "$scenario"
        printf '        "baseline": { "median": %s, "stddev": %s, "runs": [%s] },\n' \
          "$baseline_median" "$baseline_stddev" "$(bench_join_json_runs "${baseline_runs[@]}")"
        printf '        "candidate": { "median": %s, "stddev": %s, "runs": [%s] },\n' \
          "$candidate_median" "$candidate_stddev" "$(bench_join_json_runs "${candidate_runs[@]}")"
        printf '        "delta_s": %s,\n' "$delta_s"
        printf '        "delta_pct": %s' "$delta_pct"
        if [[ "$budget_ok" != "null" ]]; then
          printf ',\n        "budget_ok": %s\n' "$budget_ok"
        else
          printf '\n'
        fi
        printf '      }'
      } >>"$json_file"
    done < <(bench_scenarios_for_mode)

    if [[ "$first_scenario" == true ]]; then
      printf ' }' >>"$json_file"
    else
      printf '\n    }' >>"$json_file"
    fi
  done

  {
    printf '\n'
    printf '  }\n'
    printf '}\n'
  } >>"$json_file"
}

bench_colorize_human_stream() {
  while IFS= read -r line; do
    case "$line" in
      "═══════════════════════════════════════════")
        printf '%s%s%s\n' "$COLOR_BOLD" "$line" "$COLOR_RESET"
        ;;
      "  Component Benchmark Summary")
        printf '%s%s%s\n' "$COLOR_BOLD" "$line" "$COLOR_RESET"
        ;;
      "Cold Install"*|"Warm Rebuild"*|"60s Budget"*|"Results:")
        printf '%s%s%s\n' "$COLOR_BOLD" "$line" "$COLOR_RESET"
        ;;
      "  ✓ "*)
        printf '  %s✓%s%s\n' "$COLOR_GREEN" "$COLOR_RESET" "${line#  ✓}"
        ;;
      "  ✗ "*)
        printf '  %s✗%s%s\n' "$COLOR_RED" "$COLOR_RESET" "${line#  ✗}"
        ;;
      *"OVER"*)
        printf '%s%s%s\n' "$COLOR_RED" "$line" "$COLOR_RESET"
        ;;
      "  |"*)
        printf '%s%s%s\n' "$COLOR_YELLOW" "$line" "$COLOR_RESET"
        ;;
      *)
        printf '%s\n' "$line"
        ;;
    esac
  done <"$(bench_human_report_file)"
}

bench_generate_reports() {
  local raw_file=$1

  bench_generate_human_report "$raw_file"
  bench_generate_tsv_report "$raw_file"
  bench_generate_json_report "$raw_file"
}

bench_emit_selected_format() {
  case "$BENCH_FORMAT" in
    human)
      if bench_supports_color; then
        bench_colorize_human_stream
      else
        cat "$(bench_human_report_file)"
      fi
      ;;
    tsv)
      cat "$(bench_tsv_report_file)"
      ;;
    json)
      cat "$(bench_json_report_file)"
      ;;
  esac
}

bench_kill_tree() {
  local pid=$1
  local children
  children=$(pgrep -P "$pid" 2>/dev/null) || true
  local child
  for child in $children; do
    bench_kill_tree "$child"
  done
  kill -TERM "$pid" 2>/dev/null || true
}

bench_kill_children() {
  local pid
  for pid in "${BENCH_BG_PIDS[@]}"; do
    if kill -0 "$pid" 2>/dev/null; then
      bench_kill_tree "$pid"
    fi
  done
  for pid in "${BENCH_BG_PIDS[@]}"; do
    wait "$pid" 2>/dev/null || true
  done
  BENCH_BG_PIDS=()
}

bench_cleanup() {
  bench_kill_children

  local path

  if [[ ${#WORKTREE_PATHS[@]} -gt 0 ]]; then
    for path in "${WORKTREE_PATHS[@]}"; do
      if [[ -d "$path" ]]; then
        git -C "$REPO_ROOT" worktree remove --force "$path" >/dev/null 2>&1 || true
      fi
    done
  fi

  if [[ -n "$BENCH_TMPDIR" && -d "$BENCH_TMPDIR" ]]; then
    chmod -R u+w "$BENCH_TMPDIR" >/dev/null 2>&1 || true
    rm -rf "$BENCH_TMPDIR"
  fi
}

bench_handle_signal() {
  if [[ "$BENCH_INTERRUPTED" == true ]]; then
    return
  fi
  BENCH_INTERRUPTED=true
  printf '\n' >&2
  bench_log_error "Interrupted — cleaning up…"
  bench_cleanup
  trap - EXIT INT TERM
  kill -INT $$
}

bench_parse_args() {
  local components_arg="all"

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --baseline-ref)
        BASELINE_REF=$2
        shift 2
        ;;
      --candidate-ref)
        CANDIDATE_REF=$2
        shift 2
        ;;
      --components)
        components_arg=$2
        shift 2
        ;;
      --mode)
        BENCH_MODE=$2
        shift 2
        ;;
      --repeats)
        BENCH_REPEATS=$2
        shift 2
        ;;
      --format)
        BENCH_FORMAT=$2
        shift 2
        ;;
      --report-dir)
        BENCH_REPORT_DIR=$2
        shift 2
        ;;
      --ci)
        CI_MODE=true
        shift
        ;;
      --help)
        bench_usage
        exit 0
        ;;
      *)
        bench_log_error "Unknown argument: $1"
        bench_usage
        exit 1
        ;;
    esac
  done

  if [[ "${CI:-false}" == "true" ]]; then
    CI_MODE=true
  fi

  if [[ -z "$BENCH_REPEATS" ]]; then
    if [[ "$CI_MODE" == true ]]; then
      BENCH_REPEATS=5
    else
      BENCH_REPEATS=3
    fi
  fi

  if [[ -z "$BASELINE_REF" ]]; then
    BASELINE_REF=$(bench_default_baseline_ref)
  fi

  if [[ -z "$BENCH_REPORT_DIR" ]]; then
    BENCH_REPORT_DIR="$REPO_ROOT/reports/benchmarks"
  fi

  if [[ -z "$BENCH_FORMAT" ]]; then
    if [[ -t 1 && "$CI_MODE" != true ]]; then
      BENCH_FORMAT=human
    else
      BENCH_FORMAT=tsv
    fi
  fi

  bench_validate_mode
  bench_validate_format
  bench_validate_ref "$BASELINE_REF"
  bench_validate_ref "$CANDIDATE_REF"
  bench_parse_components "$components_arg"
}

bench_print_intro() {
  local baseline_short candidate_short component_count scenario_list

  baseline_short=$(git -C "$REPO_ROOT" rev-parse --short "$BASELINE_REF" 2>/dev/null || echo "$BASELINE_REF")
  candidate_short=$(git -C "$REPO_ROOT" rev-parse --short "$CANDIDATE_REF" 2>/dev/null || echo "$CANDIDATE_REF")
  component_count=${#SELECTED_COMPONENTS[@]}
  scenario_list=$(bench_scenarios_for_mode | tr '\n' '+' | sed 's/+$//')

  {
    printf '%s═══════════════════════════════════════════%s\n' "$COLOR_BOLD" "$COLOR_RESET"
    printf '%s  Ambient Code Platform — Component Bench%s\n' "$COLOR_BOLD" "$COLOR_RESET"
    printf '%s═══════════════════════════════════════════%s\n' "$COLOR_BOLD" "$COLOR_RESET"
    printf '  Baseline:   %s\n' "$baseline_short"
    printf '  Candidate:  %s\n' "$candidate_short"
    printf '  Components: %s (%s)\n' "$component_count" "${SELECTED_COMPONENTS[*]}"
    printf '  Mode:       %s   Repeats: %s\n' "$scenario_list" "$BENCH_REPEATS"
    printf '  Platform:   %s/%s\n' "$(uname -s)" "$(uname -m)"
    printf '\n'
  } >&2
}

bench_run() {
  local component
  local pid
  local status=0
  local raw_file

  bench_ensure_report_dir
  BENCH_TMPDIR=$(mktemp -d)
  trap bench_cleanup EXIT
  trap bench_handle_signal INT TERM

  bench_print_intro

  if ! bench_preflight_selected_components; then
    status=1
  fi

  if [[ ${#READY_COMPONENTS[@]} -gt 0 ]]; then
    for component in "${READY_COMPONENTS[@]}"; do
      bench_setup_component_worktrees "$component"
    done

    for component in "${READY_COMPONENTS[@]}"; do
      bench_run_component_job "$component" &
      BENCH_BG_PIDS+=($!)
    done
  fi

  for pid in "${BENCH_BG_PIDS[@]}"; do
    if ! wait "$pid" 2>/dev/null; then
      status=1
    fi
    if [[ "$BENCH_INTERRUPTED" == true ]]; then
      status=1
      break
    fi
  done

  raw_file=$(bench_combined_raw_file)
  : >"$raw_file"
  for component in "${SELECTED_COMPONENTS[@]}"; do
    cat "$(bench_result_file_for_component "$component")" >>"$raw_file"
  done

  bench_generate_reports "$raw_file"
  bench_emit_selected_format
  return "$status"
}

source "$SCRIPT_DIR/bench-manifest.sh"

main() {
  cd "$REPO_ROOT"
  bench_parse_args "$@"
  bench_init_colors
  bench_run
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
