#!/usr/bin/env bash
# demo-github.sh — acpctl end-to-end demo: GitHub credential → interactive session
#
# What this demo does:
#   1. Log in (reads from existing acpctl config, or prompts)
#   2. Create a project
#   3. Create an agent
#   4. Create a credential (GitHub PAT — prompts for token path, defaults to env var)
#   5. Bind the credential to the agent (credential:reader scope=agent)
#   6. Start an interactive session with a prompt to open a GitHub issue as a test
#   7. Stream session messages live until RUN_FINISHED
#   8. Clean up
#
# Usage:
#   ./demo-github.sh
#   GITHUB_TOKEN_FILE=/path/to/token ./demo-github.sh
#   GITHUB_REPO=org/repo ./demo-github.sh
#   PAUSE=1 ./demo-github.sh     # pause between steps
#
# Optional env:
#   GITHUB_TOKEN_FILE   — path to file containing GitHub PAT (default: prompted, falls back to $GITHUB_TOKEN)
#   GITHUB_REPO         — org/repo to open test issue in (default: prompted)
#   ACPCTL              — path to acpctl binary (default: acpctl from PATH)
#   PAUSE               — seconds between demo steps (default: 0)
#   SESSION_READY_TIMEOUT — seconds to wait for Running (default: 180)
#   MESSAGE_WAIT_TIMEOUT  — seconds to wait for RUN_FINISHED (default: 300)

set -euo pipefail

ACPCTL="${ACPCTL:-acpctl}"
PAUSE="${PAUSE:-0}"
SESSION_READY_TIMEOUT="${SESSION_READY_TIMEOUT:-180}"
MESSAGE_WAIT_TIMEOUT="${MESSAGE_WAIT_TIMEOUT:-300}"

# ── helpers ────────────────────────────────────────────────────────────────────

bold()  { printf '\033[1m%s\033[0m\n' "$*"; }
dim()   { printf '\033[2m%s\033[0m\n' "$*"; }
cyan()  { printf '\033[36m%s\033[0m\n' "$*"; }
green() { printf '\033[32m%s\033[0m\n' "$*"; }
yellow(){ printf '\033[33m%s\033[0m\n' "$*"; }
red()   { printf '\033[31m%s\033[0m\n' "$*"; }
sep()   { printf '\033[2m%s\033[0m\n' "──────────────────────────────────────────────────"; }

step() {
    local description="$1"
    shift
    echo
    sep
    bold "▶  $description"
    printf '\033[38;5;214m   $ %s\033[0m\n' "$*"
    sleep "$PAUSE"
    "$@"
    echo
}

announce() {
    echo
    sep
    cyan "━━  $*"
    sep
    sleep "$PAUSE"
}

die() { red "error: $*" >&2; exit 1; }

# ── preflight ──────────────────────────────────────────────────────────────────

command -v "$ACPCTL" &>/dev/null || die "${ACPCTL} not found. Set ACPCTL=/path/to/acpctl or add to PATH."
command -v python3   &>/dev/null || die "python3 not found."

# ── intro ──────────────────────────────────────────────────────────────────────

echo
bold "Ambient CLI Demo — GitHub Credential"
sep
echo
printf '  %s\n' "1. Create a project + agent"
printf '  %s\n' "2. Create a GitHub credential (PAT)"
printf '  %s\n' "3. Bind the credential to the agent"
printf '  %s\n' "4. Start an interactive session"
printf '  %s\n' "5. Agent opens a test GitHub issue to verify auth"
echo
printf '  \033[38;5;214m%-38s\033[0m %s\n' "Orange text like this" "= a terminal command being run"
echo
sep

# ── gather inputs ──────────────────────────────────────────────────────────────

announce "0 · Configuration"

# GitHub PAT
if [[ -n "${GITHUB_TOKEN_FILE:-}" ]]; then
    dim "   Using GITHUB_TOKEN_FILE=${GITHUB_TOKEN_FILE}"
    GITHUB_PAT_PATH="${GITHUB_TOKEN_FILE}"
else
    DEFAULT_TOKEN_PATH="${GITHUB_TOKEN:-}"
    printf '\033[1m   GitHub PAT file path\033[0m (leave blank to use \$GITHUB_TOKEN env var): '
    read -r GITHUB_PAT_PATH
    GITHUB_PAT_PATH="${GITHUB_PAT_PATH:-}"
fi

if [[ -n "${GITHUB_PAT_PATH}" && -f "${GITHUB_PAT_PATH}" ]]; then
    GITHUB_TOKEN_VALUE="$(cat "${GITHUB_PAT_PATH}")"
    dim "   Token read from file: ${GITHUB_PAT_PATH}"
elif [[ -n "${GITHUB_TOKEN:-}" ]]; then
    GITHUB_TOKEN_VALUE="${GITHUB_TOKEN}"
    dim "   Token read from \$GITHUB_TOKEN env var"
else
    die "No GitHub token found. Set GITHUB_TOKEN_FILE, pass a file path, or export GITHUB_TOKEN."
fi

[[ -z "${GITHUB_TOKEN_VALUE}" ]] && die "GitHub token is empty."

# GitHub repo
if [[ -n "${GITHUB_REPO:-}" ]]; then
    dim "   Using GITHUB_REPO=${GITHUB_REPO}"
else
    printf '\033[1m   GitHub repo to open test issue in\033[0m (e.g. org/repo): '
    read -r GITHUB_REPO
    [[ -z "${GITHUB_REPO}" ]] && die "GITHUB_REPO is required."
fi

RUN_ID=$(date +%s | tail -c6)
PROJECT_NAME="demo-github-${RUN_ID}"
AGENT_NAME="github-agent"
CRED_NAME="github-pat-demo-${RUN_ID}"

echo
dim "   Project:    ${PROJECT_NAME}"
dim "   Agent:      ${AGENT_NAME}"
dim "   Credential: ${CRED_NAME}"
dim "   Repo:       ${GITHUB_REPO}"

echo
bold "   Press Enter to begin..."
read -r

# ── cleanup trap ───────────────────────────────────────────────────────────────

CREATED_SESSION_ID=""
CREATED_PROJECT=""
CREATED_CREDENTIAL_ID=""

cleanup() {
    if [[ -n "${NO_CLEANUP:-}" ]]; then
        echo
        yellow "   NO_CLEANUP set — skipping cleanup"
        dim    "   session:    ${CREATED_SESSION_ID}"
        dim    "   credential: ${CREATED_CREDENTIAL_ID}"
        dim    "   project:    ${CREATED_PROJECT}"
        return
    fi
    echo
    announce "Cleanup"
    if [[ -n "${CREATED_SESSION_ID}" ]]; then
        dim "   stopping session ${CREATED_SESSION_ID}..."
        "$ACPCTL" stop "${CREATED_SESSION_ID}" 2>/dev/null || true
        "$ACPCTL" delete session "${CREATED_SESSION_ID}" -y 2>/dev/null || true
    fi
    if [[ -n "${CREATED_CREDENTIAL_ID}" ]]; then
        dim "   deleting credential ${CREATED_CREDENTIAL_ID}..."
        "$ACPCTL" credential delete "${CREATED_CREDENTIAL_ID}" --confirm 2>/dev/null || true
    fi
    if [[ -n "${CREATED_PROJECT}" ]]; then
        dim "   deleting project ${CREATED_PROJECT}..."
        "$ACPCTL" delete project "${CREATED_PROJECT}" -y 2>/dev/null || true
    fi
    green "   cleanup done"
}
trap cleanup EXIT

# ── helpers ────────────────────────────────────────────────────────────────────

json_field() {
    local json="$1" field="$2"
    echo "$json" | python3 -c "import sys,json; print(json.load(sys.stdin)['${field}'])" 2>/dev/null
}

wait_for_running() {
    local session_id="$1"
    local deadline=$(( $(date +%s) + SESSION_READY_TIMEOUT ))
    local last_phase=""
    printf '   waiting for Running (timeout %ds)...\n' "${SESSION_READY_TIMEOUT}"
    while true; do
        local phase
        phase=$(
            "$ACPCTL" get session "$session_id" -o json 2>/dev/null \
            | python3 -c "import sys,json; print(json.load(sys.stdin).get('phase',''))" 2>/dev/null || true
        )
        if [[ "$phase" != "$last_phase" ]]; then
            printf '   phase: %s\n' "$phase"
            last_phase="$phase"
        fi
        [[ "$phase" == "Running" ]] && { green "   ✓ session is Running"; return 0; }
        [[ $(date +%s) -ge $deadline ]] && { yellow "   ✗ timed out (phase=${phase:-unknown})"; return 1; }
        sleep 3
    done
}

max_seq() {
    local session_id="$1"
    "$ACPCTL" session messages "${session_id}" -o json 2>/dev/null \
    | python3 -c "
import sys, json
try:
    msgs = json.load(sys.stdin)
    print(max((m.get('seq', 0) for m in msgs), default=0) if isinstance(msgs, list) else 0)
except Exception:
    print(0)
" 2>/dev/null || echo 0
}

wait_for_run_finished() {
    local session_id="$1" after_seq="$2"
    local start=$(date +%s)
    local status="none"
    local matched_type=""

    while IFS= read -r line; do
        if echo "$line" | grep -qE 'RUN_FINISHED|RUN_ERROR'; then
            if echo "$line" | grep -q 'RUN_FINISHED'; then
                status="finished"; matched_type="RUN_FINISHED"; break
            elif echo "$line" | grep -q 'RUN_ERROR'; then
                status="error"; matched_type="RUN_ERROR"; break
            fi
        fi
    done < <(timeout "${MESSAGE_WAIT_TIMEOUT}" "${ACPCTL}" session messages "${session_id}" -f --after "${after_seq}" 2>/dev/null)

    local elapsed=$(( $(date +%s) - start ))
    case "$status" in
        finished) green  "   ✓ ${matched_type} (${elapsed}s)"; return 0 ;;
        error)    yellow "   ✗ ${matched_type} (${elapsed}s)"; return 1 ;;
        *)        yellow "   ✗ timeout after ${MESSAGE_WAIT_TIMEOUT}s"; return 1 ;;
    esac
}

# ── 1: login / whoami ──────────────────────────────────────────────────────────

announce "1 · Verify login"

step "Show authenticated user" \
    "$ACPCTL" whoami

# ── 2: project ────────────────────────────────────────────────────────────────

announce "2 · Create project"

step "Create project: ${PROJECT_NAME}" \
    "$ACPCTL" create project \
        --name "${PROJECT_NAME}" \
        --description "GitHub credential demo"

CREATED_PROJECT="${PROJECT_NAME}"

step "Set project context" \
    "$ACPCTL" project "${PROJECT_NAME}"

# ── 3: agent ──────────────────────────────────────────────────────────────────

announce "3 · Create agent"

sep; bold "▶  Create agent: ${AGENT_NAME}"; sleep "$PAUSE"
AGENT_JSON=$(
    "$ACPCTL" agent create \
        --project-id "${PROJECT_NAME}" \
        --name "${AGENT_NAME}" \
        --prompt "You are a GitHub automation agent. You use the GitHub CLI (gh) and GitHub API to manage issues and pull requests. When given a credential, you authenticate with it and perform the requested GitHub operations." \
        -o json 2>/dev/null
)
AGENT_ID=$(json_field "$AGENT_JSON" "id")
[[ -z "${AGENT_ID}" ]] && die "Failed to parse agent ID"
green "   ✓ agent created: ${AGENT_ID}"
echo

# ── 4: credential ─────────────────────────────────────────────────────────────

announce "4 · Create GitHub credential"

sep; bold "▶  Create credential: ${CRED_NAME}"; sleep "$PAUSE"
_CRED_MANIFEST=$(mktemp --suffix=.yaml)
cat > "${_CRED_MANIFEST}" <<'CRED_EOF'
kind: Credential
name: CRED_NAME_PLACEHOLDER
provider: github
token: $DEMO_GITHUB_PAT
description: CRED_DESC_PLACEHOLDER
CRED_EOF
sed -i \
    -e "s/CRED_NAME_PLACEHOLDER/${CRED_NAME}/" \
    -e "s/CRED_DESC_PLACEHOLDER/GitHub PAT for demo ${RUN_ID}/" \
    "${_CRED_MANIFEST}"
DEMO_GITHUB_PAT="${GITHUB_TOKEN_VALUE}" \
    "$ACPCTL" apply -f "${_CRED_MANIFEST}" 2>/dev/null
rm -f "${_CRED_MANIFEST}"
CRED_JSON=$(
    "$ACPCTL" get credentials -o json 2>/dev/null \
    | python3 -c "
import sys, json
data = json.load(sys.stdin)
items = data.get('items', []) if isinstance(data, dict) else data
for c in items:
    if c.get('name') == '${CRED_NAME}':
        print(json.dumps(c))
        break
" 2>/dev/null
)
CREDENTIAL_ID=$(json_field "$CRED_JSON" "id")
[[ -z "${CREDENTIAL_ID}" ]] && die "Failed to parse credential ID"
CREATED_CREDENTIAL_ID="${CREDENTIAL_ID}"
green "   ✓ credential created: ${CREDENTIAL_ID}"
echo

step "Verify credential visible" \
    "$ACPCTL" get credentials

# ── 5: role binding ───────────────────────────────────────────────────────────

announce "5 · Bind credential to agent"

sep; bold "▶  Look up credential:token-reader role ID"; sleep "$PAUSE"
ROLES_JSON=$("$ACPCTL" get roles -o json 2>/dev/null)
READER_ROLE_ID=$(
    echo "$ROLES_JSON" | python3 -c "
import sys, json
data = json.load(sys.stdin)
items = data.get('items', []) if isinstance(data, dict) else data
for r in items:
    if r.get('name') == 'credential:token-reader':
        print(r['id'])
        break
" 2>/dev/null
)

MY_USER_ID=$(
    "$ACPCTL" whoami 2>/dev/null \
    | awk '/^User:/{print $2}' || true
)

if [[ -z "${READER_ROLE_ID}" ]]; then
    yellow "   credential:token-reader role not in this deployment — skipping role binding"
    dim   "   (credential roles are seeded by the api-server migration; redeploy may be needed)"
else
    dim "   credential:token-reader role ID: ${READER_ROLE_ID}"
    dim "   my user ID: ${MY_USER_ID}"

    sep; bold "▶  Create role-binding: credential:token-reader scope=agent"; sleep "$PAUSE"
    "$ACPCTL" create role-binding \
        --user-id "${MY_USER_ID}" \
        --role-id "${READER_ROLE_ID}" \
        --scope agent \
        --scope-id "${AGENT_ID}" || yellow "   role-binding creation failed (may already exist)"
    echo
    green "   ✓ credential bound to agent"
fi

# ── 6: start session ──────────────────────────────────────────────────────────

announce "6 · Start interactive session"

SESSION_PROMPT="You have access to a GitHub Personal Access Token via the platform credential API.

To authenticate with GitHub:
1. Call GET /api/ambient/v1/credentials/${CREDENTIAL_ID}/token to retrieve the raw token
2. Use the token to authenticate: export GITHUB_TOKEN=<token>
3. Use the gh CLI or curl to interact with the GitHub API

Your task: Open a test GitHub issue in the repository ${GITHUB_REPO} with:
  Title: [ambient-demo] Credential integration test ${RUN_ID}
  Body:  This issue was automatically opened by the Ambient platform credential demo on $(date -u +%Y-%m-%dT%H:%M:%SZ). It can be closed immediately.

After opening the issue, report the issue URL back as confirmation."

sep; bold "▶  Start session for agent ${AGENT_NAME}"; sleep "$PAUSE"
SESSION_JSON=$(
    "$ACPCTL" agent start "${AGENT_NAME}" \
        --project-id "${PROJECT_NAME}" \
        --prompt "${SESSION_PROMPT}" \
        -o json 2>&1
)
SESSION_ID=$(json_field "$SESSION_JSON" "id")
if [[ -z "${SESSION_ID}" ]]; then
    red "   Failed to start session. Output:"
    echo "${SESSION_JSON}"
    die "Failed to parse session ID"
fi
CREATED_SESSION_ID="${SESSION_ID}"
green "   ✓ session created: ${SESSION_ID}"
echo

# ── 7: wait for Running ───────────────────────────────────────────────────────

announce "7 · Wait for session Running"

wait_for_running "${SESSION_ID}" || true

# ── 8: stream messages ────────────────────────────────────────────────────────

announce "8 · Streaming session messages (waiting for RUN_FINISHED)"

BEFORE_SEQ=0
wait_for_run_finished "${SESSION_ID}" "${BEFORE_SEQ}" || true

# ── 9: show result ────────────────────────────────────────────────────────────

announce "9 · Session result"

step "Final session messages" \
    "$ACPCTL" session messages "${SESSION_ID}"

step "Final session state" \
    "$ACPCTL" describe session "${SESSION_ID}"

# ── done (cleanup runs via trap) ──────────────────────────────────────────────

echo
sep
green "  Demo complete ✓"
dim   "  Project ${PROJECT_NAME} and credential ${CREDENTIAL_ID} will be deleted by cleanup."
sep
echo
