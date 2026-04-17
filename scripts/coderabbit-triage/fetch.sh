#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DATA_DIR="${SCRIPT_DIR}/data"

# Defaults
REPO="ambient-code/platform"
RELEASE=""
SINCE=""
ALL_MODE=false

usage() {
    cat <<EOF
Usage: $(basename "$0") [OPTIONS]

Extract CodeRabbit review comments from GitHub for a given release window.

Options:
  --release TAG    Release tag to analyze (default: latest tag matching v*.*.*)
  --since TAG      Previous release tag (default: tag before --release)
  --all            Fetch ALL comments across all time (initial backfill)
  --repo OWNER/REPO  Repository to query (default: ambient-code/platform)
  --help           Show this help message

Examples:
  $(basename "$0") --release v0.2.0 --since v0.1.4
  $(basename "$0") --all
  $(basename "$0") --repo ambient-code/platform --release v0.3.0
EOF
    exit 0
}

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------
if [[ $# -eq 0 ]]; then
    usage
fi

while [[ $# -gt 0 ]]; do
    case "$1" in
        --release)
            RELEASE="$2"
            shift 2
            ;;
        --since)
            SINCE="$2"
            shift 2
            ;;
        --all)
            ALL_MODE=true
            shift
            ;;
        --repo)
            REPO="$2"
            shift 2
            ;;
        --help|-h)
            usage
            ;;
        *)
            echo "Error: Unknown option: $1" >&2
            usage
            ;;
    esac
done

# ---------------------------------------------------------------------------
# Preflight checks
# ---------------------------------------------------------------------------
if ! command -v gh &>/dev/null; then
    echo "Error: gh (GitHub CLI) is not installed." >&2
    exit 1
fi

if ! gh auth status &>/dev/null; then
    echo "Error: gh is not authenticated. Run 'gh auth login' first." >&2
    exit 1
fi

if ! command -v jq &>/dev/null; then
    echo "Error: jq is not installed." >&2
    exit 1
fi

# ---------------------------------------------------------------------------
# Rate-limit helper
# ---------------------------------------------------------------------------
check_rate_limit() {
    local remaining
    remaining=$(gh api rate_limit --jq '.rate.remaining' 2>/dev/null || echo "unknown")
    if [[ "$remaining" != "unknown" ]] && [[ "$remaining" -lt 50 ]]; then
        local reset_epoch
        reset_epoch=$(gh api rate_limit --jq '.rate.reset' 2>/dev/null || echo "unknown")
        local reset_time="unknown"
        if [[ "$reset_epoch" != "unknown" ]]; then
            reset_time=$(date -r "$reset_epoch" 2>/dev/null || date -d "@$reset_epoch" 2>/dev/null || echo "$reset_epoch")
        fi
        echo "WARNING: GitHub API rate limit low (${remaining} remaining). Resets at ${reset_time}." >&2
    fi
}

# ---------------------------------------------------------------------------
# Tag-to-date helper
# ---------------------------------------------------------------------------
tag_to_date() {
    local tag="$1"
    git log -1 --format=%cI "$tag" 2>/dev/null || {
        echo "Error: Could not resolve tag '${tag}' to a date. Is the tag fetched locally?" >&2
        exit 1
    }
}

# ---------------------------------------------------------------------------
# Resolve default tags
# ---------------------------------------------------------------------------
resolve_release_tag() {
    if [[ -z "$RELEASE" ]]; then
        RELEASE=$(git tag --list 'v*.*.*' --sort=-v:refname | head -n 1)
        if [[ -z "$RELEASE" ]]; then
            echo "Error: No release tags matching v*.*.* found." >&2
            exit 1
        fi
        echo "Auto-detected latest release tag: ${RELEASE}"
    fi
}

resolve_since_tag() {
    if [[ -z "$SINCE" ]]; then
        # Find the tag immediately before RELEASE in version-sorted order
        SINCE=$(git tag --list 'v*.*.*' --sort=-v:refname | grep -A1 "^${RELEASE}$" | tail -n 1)
        if [[ -z "$SINCE" || "$SINCE" == "$RELEASE" ]]; then
            echo "Error: Could not determine previous tag before '${RELEASE}'." >&2
            exit 1
        fi
        echo "Auto-detected previous release tag: ${SINCE}"
    fi
}

# ---------------------------------------------------------------------------
# CodeRabbit comment filter (jq)
# ---------------------------------------------------------------------------
CODERABBIT_JQ_FILTER='[.[] | select(.user.login == "coderabbitai[bot]") | {
    id: .id,
    body: .body,
    path: .path,
    line: .line,
    created_at: .created_at,
    pull_request_url: .pull_request_url,
    html_url: .html_url
}]'

# ---------------------------------------------------------------------------
# --all mode
# ---------------------------------------------------------------------------
fetch_all() {
    local out_dir="${DATA_DIR}/all"
    mkdir -p "$out_dir"

    echo "Fetching ALL review comments from ${REPO}..."
    check_rate_limit

    gh api --paginate "repos/${REPO}/pulls/comments" \
        --jq "[.[] | select(.user.login == \"coderabbitai[bot]\") | {
            id: .id,
            body: .body,
            path: .path,
            line: .line,
            created_at: .created_at,
            pull_request_url: .pull_request_url,
            html_url: .html_url,
            pr_number: (.pull_request_url | split(\"/\") | last | tonumber)
        }]" > "${out_dir}/raw-comments.tmp.json"

    # gh --paginate with --jq emits one JSON array per page; merge them
    jq -s 'add // []' "${out_dir}/raw-comments.tmp.json" > "${out_dir}/raw-comments.json"
    rm -f "${out_dir}/raw-comments.tmp.json"

    local count
    count=$(jq 'length' "${out_dir}/raw-comments.json")
    echo "Wrote ${count} CodeRabbit comments to ${out_dir}/raw-comments.json"
}

# ---------------------------------------------------------------------------
# Per-release mode
# ---------------------------------------------------------------------------
fetch_release() {
    resolve_release_tag
    resolve_since_tag

    local since_date release_date
    since_date=$(tag_to_date "$SINCE")
    release_date=$(tag_to_date "$RELEASE")

    # Trim to date-only for the search query (YYYY-MM-DD)
    local since_day release_day
    since_day="${since_date%%T*}"
    release_day="${release_date%%T*}"

    local out_dir="${DATA_DIR}/${RELEASE}"
    mkdir -p "$out_dir"

    echo "Fetching PRs merged between ${SINCE} (${since_day}) and ${RELEASE} (${release_day})..."
    check_rate_limit

    # Search for merged PRs in the window (use query string, not -f form field)
    local encoded_query
    encoded_query="repo:${REPO}+is:pr+is:merged+merged:${since_day}..${release_day}"
    local pr_numbers
    pr_numbers=$(gh api --paginate "search/issues?q=${encoded_query}" \
        --jq '.items[].number')

    if [[ -z "$pr_numbers" ]]; then
        echo "No PRs found merged between ${since_day} and ${release_day}."
        echo "[]" > "${out_dir}/raw-comments.json"
        echo "Wrote 0 CodeRabbit comments to ${out_dir}/raw-comments.json"
        return
    fi

    local pr_count
    pr_count=$(echo "$pr_numbers" | wc -l | tr -d ' ')
    echo "Found ${pr_count} PRs"

    # Collect all comments across PRs
    local all_comments="[]"
    for pr_num in $pr_numbers; do
        echo "Fetching comments for PR #${pr_num}..."
        check_rate_limit

        local pr_comments
        pr_comments=$(gh api --paginate "repos/${REPO}/pulls/${pr_num}/comments" \
            --jq "[.[] | select(.user.login == \"coderabbitai[bot]\") | {
                id: .id,
                body: .body,
                path: .path,
                line: .line,
                created_at: .created_at,
                pull_request_url: .pull_request_url,
                html_url: .html_url,
                pr_number: ${pr_num}
            }]" 2>/dev/null || echo "[]")

        # Merge paginated arrays, then merge into accumulator
        pr_comments=$(echo "$pr_comments" | jq -s 'add // []')
        all_comments=$(echo "${all_comments}" "${pr_comments}" | jq -s 'add // []')
    done

    echo "$all_comments" | jq '.' > "${out_dir}/raw-comments.json"

    local count
    count=$(jq 'length' "${out_dir}/raw-comments.json")
    echo "Wrote ${count} CodeRabbit comments to ${out_dir}/raw-comments.json"
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
if $ALL_MODE; then
    fetch_all
else
    fetch_release
fi
