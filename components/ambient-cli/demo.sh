#!/usr/bin/env bash
# demo.sh — acpctl end-to-end walkthrough: project → session → multi-turn conversation → cleanup
#
# Works against any deployment: kind, ROSA, or localhost.
#
# Required env vars (or set them below):
#   AMBIENT_API_URL   — e.g. http://localhost:8000 or https://ambient-api-server-ambient-code.apps...
#   AMBIENT_TOKEN     — bearer token (oc whoami --show-token, or a static dev token)
#
# Optional:
#   ACPCTL            — path to acpctl binary (default: acpctl from PATH)
#   PAUSE             — seconds to sleep between steps for demos (default: 0)
#   SESSION_READY_TIMEOUT — seconds to wait for session Running (default: 120)
#   MESSAGE_WAIT_TIMEOUT  — seconds to wait for each message response (default: 60)

set -euo pipefail

ACPCTL="${ACPCTL:-acpctl}"
PAUSE="${PAUSE:-0}"
SESSION_READY_TIMEOUT="${SESSION_READY_TIMEOUT:-120}"
MESSAGE_WAIT_TIMEOUT="${MESSAGE_WAIT_TIMEOUT:-60}"

# ── resolve API URL and token ──────────────────────────────────────────────────

if [[ -z "${AMBIENT_API_URL:-}" ]]; then
    if command -v oc &>/dev/null && oc whoami &>/dev/null 2>&1; then
        AMBIENT_API_URL=$(oc whoami --show-console 2>/dev/null \
            | sed 's|console-openshift-console\.apps\.|ambient-api-server-ambient-code.apps.|')
    else
        AMBIENT_API_URL="http://localhost:8000"
    fi
fi

if [[ -z "${AMBIENT_TOKEN:-}" ]]; then
    if command -v oc &>/dev/null && oc whoami &>/dev/null 2>&1; then
        AMBIENT_TOKEN=$(oc whoami --show-token 2>/dev/null)
    else
        printf 'error: AMBIENT_TOKEN is required. Export it or log in with oc.\n' >&2
        exit 1
    fi
fi

RUN_ID=$(date +%s | tail -c5)
PROJECT_NAME="demo-${RUN_ID}"

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

api_get() {
    local path="$1"
    curl -sk \
        -H "Authorization: Bearer ${AMBIENT_TOKEN}" \
        -H "X-Ambient-Project: ${PROJECT_NAME}" \
        "${AMBIENT_API_URL}/api/ambient/v1${path}"
}

wait_for_running() {
    local session_id="$1"
    local deadline=$(( $(date +%s) + SESSION_READY_TIMEOUT ))
    local last_phase=""

    printf '   waiting for session to reach Running (timeout %ds)...\n' "$SESSION_READY_TIMEOUT"

    while true; do
        local phase
        phase=$(
            "$ACPCTL" get session "$session_id" -o json 2>/dev/null \
            | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('phase',''))" 2>/dev/null || true
        )

        if [[ "$phase" != "$last_phase" ]]; then
            printf '   phase: %s\n' "$phase"
            last_phase="$phase"
        fi

        if [[ "$phase" == "Running" ]]; then
            green "   ✓ session is Running"
            return 0
        fi

        if [[ $(date +%s) -ge $deadline ]]; then
            yellow "   ✗ timed out after ${SESSION_READY_TIMEOUT}s (phase=${phase:-unknown})"
            return 1
        fi

        sleep 3
    done
}

wait_for_new_messages() {
    local session_id="$1"
    local after_seq="$2"
    local deadline=$(( $(date +%s) + MESSAGE_WAIT_TIMEOUT ))

    printf '   waiting for response (timeout %ds, after_seq=%s)...\n' "$MESSAGE_WAIT_TIMEOUT" "$after_seq"

    while true; do
        local count
        count=$(
            api_get "/sessions/${session_id}/messages?after_seq=${after_seq}" \
            | python3 -c "
import sys, json
try:
    msgs = json.load(sys.stdin)
    print(len(msgs) if isinstance(msgs, list) else 0)
except Exception:
    print(0)
" 2>/dev/null || echo 0
        )

        if [[ "$count" -gt 0 ]]; then
            green "   ✓ ${count} new message(s) received"
            return 0
        fi

        if [[ $(date +%s) -ge $deadline ]]; then
            yellow "   ✗ no response after ${MESSAGE_WAIT_TIMEOUT}s"
            return 1
        fi

        sleep 2
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

# ── preflight ──────────────────────────────────────────────────────────────────

if ! command -v "$ACPCTL" &>/dev/null; then
    printf 'error: %s not found. Set ACPCTL=/path/to/acpctl or add it to PATH.\n' "$ACPCTL" >&2
    exit 1
fi

echo
bold "Ambient CLI Demo"
dim  "  API:   ${AMBIENT_API_URL}"
dim  "  Run:   ${PROJECT_NAME}"
dim  "  Token: ${AMBIENT_TOKEN:0:8}..."

# ── section 0: login ───────────────────────────────────────────────────────────

announce "0 · Log in"

step "Log in to the Ambient API server" \
    "$ACPCTL" login "$AMBIENT_API_URL" \
        --token "$AMBIENT_TOKEN" \
        --insecure-skip-tls-verify

step "Show authenticated user" \
    "$ACPCTL" whoami

# ── section 1: project ────────────────────────────────────────────────────────

announce "1 · Create project"

step "Create project: ${PROJECT_NAME}" \
    "$ACPCTL" create project \
        --name "${PROJECT_NAME}" \
        --display-name "Demo Project ${RUN_ID}" \
        --description "End-to-end demo"

step "Set project context" \
    "$ACPCTL" project "${PROJECT_NAME}"

step "Confirm project context" \
    "$ACPCTL" project current

# ── section 2: session ────────────────────────────────────────────────────────

announce "2 · Create session"

sep; bold "▶  Create demo-session-1"; sleep "$PAUSE"
SESSION_JSON=$(
    "$ACPCTL" create session \
        --name demo-session-1 \
        --prompt "You are a helpful assistant. Reply concisely and clearly." \
        -o json 2>/dev/null
)
SESSION_ID=$(echo "$SESSION_JSON" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])" 2>/dev/null)
if [[ -z "$SESSION_ID" ]]; then
    red "   ✗ failed to parse session ID"
    exit 1
fi
dim "   session ID: ${SESSION_ID}"; echo

step "List sessions" \
    "$ACPCTL" get sessions

step "Describe demo-session-1" \
    "$ACPCTL" describe session "${SESSION_ID}"

# ── section 3: wait for Running ───────────────────────────────────────────────

announce "3 · Wait for session Running"

wait_for_running "${SESSION_ID}" || true

# ── section 4: multi-turn conversation ────────────────────────────────────────

announce "4 · Multi-turn conversation"

send_turn() {
    local turn="$1"
    local msg="$2"

    local before_seq
    before_seq=$(max_seq "${SESSION_ID}")

    echo
    sep
    bold "▶  Turn ${turn}: sending message"
    dim  "   ${msg}"
    sleep "$PAUSE"
    "$ACPCTL" session send "${SESSION_ID}" "$msg"
    echo

    bold "▶  Turn ${turn}: waiting for response..."
    wait_for_new_messages "${SESSION_ID}" "${before_seq}" || true

    step "Turn ${turn}: conversation so far" \
        "$ACPCTL" session messages "${SESSION_ID}"
}

send_turn 1 "Hello! Confirm you are running and tell me today's date."
send_turn 2 "What are three practical uses of AI in software development?"
send_turn 3 "Write a one-sentence mission statement for an AI-powered developer platform."

# ── section 5: stream ─────────────────────────────────────────────────────────

announce "5 · Stream all messages (5s window)"

bold "▶  Streaming session messages..."
sleep "$PAUSE"
timeout 5 "$ACPCTL" session messages "${SESSION_ID}" -f || true
echo

# ── section 6: final state ────────────────────────────────────────────────────

announce "6 · Final state"

step "Session detail" \
    "$ACPCTL" describe session "${SESSION_ID}"

step "All messages (JSON)" \
    "$ACPCTL" session messages "${SESSION_ID}" -o json

# ── section 7: cleanup ────────────────────────────────────────────────────────

announce "7 · Stop and clean up"

sep; bold "▶  Stop demo-session-1"; sleep "$PAUSE"
"$ACPCTL" stop "${SESSION_ID}" || true; echo

step "Verify session stopped" \
    "$ACPCTL" get sessions

step "Delete session" \
    "$ACPCTL" delete session "${SESSION_ID}" -y

step "Delete project ${PROJECT_NAME}" \
    "$ACPCTL" delete project "${PROJECT_NAME}" -y

step "Confirm cleanup" \
    "$ACPCTL" get projects

# ── done ──────────────────────────────────────────────────────────────────────

echo
sep
green "  Demo complete ✓"
sep
echo
