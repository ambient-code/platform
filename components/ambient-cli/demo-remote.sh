#!/usr/bin/env bash
# demo-remote.sh — acpctl two-agent conversation demo against a remote deployment
#
# Layout: 2-pane tmux — left = main demo, right = live message watch.
# Creates two sessions ("historian" and "poet"). They have an extended
# conversation about Charleston, SC by exchanging messages via push_message.
# The script relays each agent's response to the other agent.
#
# Usage:
#   ./demo-remote.sh https://ambient-api-server-ambient-code--ambient-s5.apps.int.spoke.prod.us-west-2.aws.paas.redhat.com/
#   AMBIENT_API_URL=https://... ./demo-remote.sh
#
# Optional env:
#   AMBIENT_API_URL            — API server URL (or pass as positional arg)
#   AMBIENT_TOKEN              — pre-existing token (skips browser login)
#   ACPCTL                     — path to acpctl binary    (default: acpctl from PATH)
#   PAUSE                      — seconds between demo steps (default: 2)
#   SESSION_READY_TIMEOUT      — seconds to wait for Running (default: 180)
#   MESSAGE_WAIT_TIMEOUT       — seconds to wait for messages (default: 120)
#   INSECURE_TLS               — set to 1 to skip TLS verification (default: unset)

set -euo pipefail

# ── tmux layout bootstrap ──────────────────────────────────────────────────────

TMUX_SESSION="ambient-demo"
DEMO_SCRIPT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/$(basename "${BASH_SOURCE[0]}")"
DEMO_ARGS=("${@}")

if [[ -z "${TMUX:-}" ]]; then
    tmux kill-session -t "$TMUX_SESSION" 2>/dev/null || true
    tmux new-session -d -s "$TMUX_SESSION" -x 220 -y 50

    tmux split-window -h -t "${TMUX_SESSION}:0" -p 40

    tmux send-keys -t "${TMUX_SESSION}:0.1" "printf '\\033[2m[watch panel — waiting for session]\\033[0m\\n'" Enter

    ESCAPED_ARGS=""
    for arg in "${DEMO_ARGS[@]}"; do
        ESCAPED_ARGS+="$(printf ' %q' "$arg")"
    done
    tmux send-keys -t "${TMUX_SESSION}:0.0" \
        "TMUX_SESSION=$(printf '%q' "$TMUX_SESSION") INSIDE_DEMO_TMUX=1 bash $(printf '%q' "$DEMO_SCRIPT")${ESCAPED_ARGS}" Enter

    tmux select-pane -t "${TMUX_SESSION}:0.0"
    tmux attach-session -t "$TMUX_SESSION"
    exit 0
fi

WATCH_PANE="1"

attach_watch() {
    local session_id="$1"
    tmux send-keys -t "${TMUX_SESSION}:0.${WATCH_PANE}" \
        "$(printf '%q' "${ACPCTL:-acpctl}") session messages $(printf '%q' "$session_id") -F" Enter
}

# ── config ────────────────────────────────────────────────────────────────────

ACPCTL="${ACPCTL:-acpctl}"
PAUSE="${PAUSE:-2}"
SESSION_READY_TIMEOUT="${SESSION_READY_TIMEOUT:-180}"
MESSAGE_WAIT_TIMEOUT="${MESSAGE_WAIT_TIMEOUT:-120}"
INSECURE_TLS="${INSECURE_TLS:-}"

if [[ $# -ge 1 ]]; then
    AMBIENT_API_URL="${1}"
fi
AMBIENT_API_URL="${AMBIENT_API_URL:-}"

if [[ -z "${AMBIENT_API_URL}" ]]; then
    AMBIENT_API_URL=$("$ACPCTL" config get api_url 2>/dev/null || true)
fi

if [[ -z "${AMBIENT_API_URL}" ]]; then
    printf '\033[31merror: no API URL. Pass as argument or set AMBIENT_API_URL.\033[0m\n' >&2
    exit 1
fi

AMBIENT_API_URL="${AMBIENT_API_URL%/}"

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

# ── preflight ──────────────────────────────────────────────────────────────────

if ! command -v "$ACPCTL" &>/dev/null; then
    red "error: ${ACPCTL} not found. Set ACPCTL=/path/to/acpctl or add to PATH." >&2; exit 1
fi

cleanup() {
    :
}
trap cleanup EXIT

# ── intro ─────────────────────────────────────────────────────────────────────

echo
bold "Ambient CLI Demo — Two-Agent Conversation (remote)"
dim  "  API:  ${AMBIENT_API_URL}"

echo
sep
bold "What this demo will do:"
echo
printf '  %s\n' "1. Log in and create a project"
printf '  %s\n' "2. Create two sessions: historian and poet"
printf '  %s\n' "3. Wait for both to reach Running"
printf '  %s\n' "4. Attach live message watch for the poet (right panel)"
printf '  %s\n' "5. Seed the historian with Charleston context"
printf '  %s\n' "6. Relay the historian's response to the poet via push_message"
printf '  %s\n' "7. Relay the poet's response back to the historian"
printf '  %s\n' "8. Continue the relay for an extended conversation"
printf '  %s\n' "9. Show final state and clean up"
echo
printf '  \033[38;5;214m%-38s\033[0m %s\n' "Orange text like this" "= a terminal command being run"
echo
sep
if [[ "${PAUSE}" -gt 0 ]] 2>/dev/null; then
    bold "   Press Enter to begin..."
    read -r
fi

# ── session helpers ───────────────────────────────────────────────────────────

wait_for_phase() {
    local session_id="$1" target_phase="$2"
    local deadline=$(( $(date +%s) + SESSION_READY_TIMEOUT ))
    local last_phase=""
    printf '   waiting for %s (timeout %ds)...\n' "${target_phase}" "${SESSION_READY_TIMEOUT}"
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
        [[ "$phase" == "$target_phase" ]] && { green "   ✓ session is ${target_phase}"; return 0; }
        [[ "$phase" == "Failed" || "$phase" == "Stopped" ]] && { red "   ✗ session is ${phase}"; return 1; }
        [[ $(date +%s) -ge $deadline ]] && { yellow "   ✗ timed out (phase=${phase:-unknown})"; return 1; }
        sleep 3
    done
}

create_session() {
    local name="$1"
    local json
    json=$(
        "$ACPCTL" create session \
            --name "$name" \
            -o json 2>/dev/null
    )
    local sid
    sid=$(echo "$json" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])" 2>/dev/null)
    if [[ -z "$sid" ]]; then
        red "   ✗ failed to parse session ID for ${name}"
        return 1
    fi
    echo "$sid"
}

extract_last_assistant_text() {
    local session_id="$1"
    "$ACPCTL" session messages "${session_id}" -o json 2>/dev/null \
    | python3 -c "
import sys, json
try:
    msgs = json.load(sys.stdin)
    for m in reversed(msgs):
        if m.get('event_type') == 'MESSAGES_SNAPSHOT':
            snapshot = json.loads(m.get('payload', '[]'))
            for msg in reversed(snapshot):
                if msg.get('role') == 'assistant':
                    content = msg.get('content', '')
                    if isinstance(content, list):
                        parts = []
                        for p in content:
                            if isinstance(p, dict) and p.get('type') == 'text':
                                parts.append(p.get('text', ''))
                        content = ' '.join(parts)
                    if content.strip():
                        print(content.strip())
                        sys.exit(0)
except Exception:
    pass
" 2>/dev/null || true
}

send_and_capture() {
    local session_id="$1" label="$2" msg="$3"

    echo; sep
    bold "▶  → ${label}: sending message"
    if [[ ${#msg} -gt 120 ]]; then
        dim "   ${msg:0:117}..."
    else
        dim "   ${msg}"
    fi
    sleep "$PAUSE"

    bold "▶  → ${label}: waiting for response..."
    if timeout "${MESSAGE_WAIT_TIMEOUT}" "$ACPCTL" session send "${session_id}" "$msg" -f 2>&1; then
        green "   ✓ ${label} responded"
    else
        red "   ✗ ${label}: send -f failed or timed out — aborting demo"
        exit 1
    fi
    echo

    LAST_RESPONSE=$(extract_last_assistant_text "${session_id}")

    bold "▶  ${label} says:"
    if [[ -n "${LAST_RESPONSE}" ]]; then
        echo "   ${LAST_RESPONSE}" | fold -s -w 76 | sed 's/^/   /'
    else
        dim "   (no response captured)"
    fi
    echo
    sleep "$PAUSE"
}

# ── section 1: login ──────────────────────────────────────────────────────────

announce "1 · Log in"

if [[ -n "${AMBIENT_TOKEN:-}" ]]; then
    TLS_FLAG=""
    if [[ -n "${INSECURE_TLS}" ]]; then
        TLS_FLAG="--insecure-skip-tls-verify"
    fi
    step "Log in with existing token" \
        "$ACPCTL" login "${AMBIENT_API_URL}" \
            --token "${AMBIENT_TOKEN}" \
            ${TLS_FLAG}
else
    TLS_FLAG=""
    if [[ -n "${INSECURE_TLS}" ]]; then
        TLS_FLAG="--insecure-skip-tls-verify"
    fi
    step "Log in via browser (Red Hat SSO)" \
        "$ACPCTL" login --use-auth-code \
            --url "${AMBIENT_API_URL}" \
            ${TLS_FLAG}
fi

step "Show authenticated user" \
    "$ACPCTL" whoami

# ── section 2: project ───────────────────────────────────────────────────────

announce "2 · Create project"

RUN_ID=$(date +%s | tail -c5)
PROJECT_NAME="demo-${RUN_ID}"

step "Create project: ${PROJECT_NAME}" \
    "$ACPCTL" create project \
        --name "${PROJECT_NAME}" \
        --display-name "Demo Project ${RUN_ID}" \
        --description "two-agent conversation demo"

step "Set project context" \
    "$ACPCTL" project "${PROJECT_NAME}"

step "Confirm project context" \
    "$ACPCTL" project current

# ── section 3: create both sessions ──────────────────────────────────────────

announce "3 · Create two sessions"

sep; bold "▶  Create session: historian"; sleep "$PAUSE"
HISTORIAN_ID=$(create_session "historian")
dim "   historian ID: ${HISTORIAN_ID}"; echo

sep; bold "▶  Create session: poet"; sleep "$PAUSE"
POET_ID=$(create_session "poet")
dim "   poet ID: ${POET_ID}"; echo

step "List sessions" \
    "$ACPCTL" get sessions

# ── section 4: wait for both to reach Running ────────────────────────────────

announce "4 · Wait for both sessions to reach Running"

bold "▶  Waiting for historian..."
wait_for_phase "${HISTORIAN_ID}" "Running" || { red "   historian did not reach Running"; exit 1; }
echo

bold "▶  Waiting for poet..."
wait_for_phase "${POET_ID}" "Running" || { red "   poet did not reach Running"; exit 1; }
echo

# ── attach watch for poet in the right panel ─────────────────────────────────

announce "5 · Attach live watch for poet session"

dim "   attaching poet message watch to right panel..."
sleep 2
attach_watch "${POET_ID}"
sleep 3
green "   ✓ right panel watching poet messages (Ctrl+C to stop)"

# ── section 6: seed the historian ─────────────────────────────────────────────

announce "6 · Seed the historian with Charleston context"

LAST_RESPONSE=""
send_and_capture "${HISTORIAN_ID}" "historian" \
    "You are a Charleston, SC historian. Tell me about what makes Charleston's geography and ecology unique — the salt marshes, barrier islands, tidal creeks, and how they shape life in the Lowcountry. Keep your response to 2-3 sentences."

HISTORIAN_OPENING="${LAST_RESPONSE}"

# ── section 7: relay to poet ─────────────────────────────────────────────────

announce "7 · Relay historian's response → poet (watch the right panel!)"

dim "   Forwarding the historian's words to the poet via push_message..."
sleep "$PAUSE"

send_and_capture "${POET_ID}" "poet" \
    "You are a Charleston poet. A historian just told you this about Charleston: \"${HISTORIAN_OPENING}\" — Respond with your own poetic perspective on what the historian described. How does this landscape feel, smell, sound? 2-3 sentences, evocative language."

POET_RESPONSE="${LAST_RESPONSE}"

# ── section 8: relay back to historian ────────────────────────────────────────

announce "8 · Relay poet's response → historian"

dim "   Forwarding the poet's words back to the historian..."
sleep "$PAUSE"

send_and_capture "${HISTORIAN_ID}" "historian" \
    "A poet responded to your description with this: \"${POET_RESPONSE}\" — Now tell me about one specific historical event or tradition in Charleston that connects to the marshes or the sea. 2-3 sentences."

HISTORIAN_HISTORY="${LAST_RESPONSE}"

# ── section 9: relay history back to poet ─────────────────────────────────────

announce "9 · Relay historian's history → poet for a final poem"

dim "   Forwarding the historian's story back to the poet..."
sleep "$PAUSE"

send_and_capture "${POET_ID}" "poet" \
    "The historian shared this story: \"${HISTORIAN_HISTORY}\" — Write a 4-line poem inspired by what the historian told you. Mention the marshes and the sea."

# ── section 10: final summary ─────────────────────────────────────────────────

announce "10 · Conversation summary"

echo
bold "The two agents had the following exchange about Charleston:"
echo
cyan "  HISTORIAN (opening):"
echo "   ${HISTORIAN_OPENING}" | fold -s -w 76 | sed 's/^/   /'
echo
cyan "  POET (response):"
echo "   ${POET_RESPONSE}" | fold -s -w 76 | sed 's/^/   /'
echo
cyan "  HISTORIAN (history):"
echo "   ${HISTORIAN_HISTORY}" | fold -s -w 76 | sed 's/^/   /'
echo
cyan "  POET (final poem):"
echo "   ${LAST_RESPONSE}" | fold -s -w 76 | sed 's/^/   /'
echo

sep
sleep "$PAUSE"

step "Historian — all messages" \
    "$ACPCTL" session messages "${HISTORIAN_ID}"

step "Poet — all messages" \
    "$ACPCTL" session messages "${POET_ID}"

# ── section 11: cleanup ──────────────────────────────────────────────────────

announce "11 · Stop and clean up"

sep; bold "▶  Stop historian"; sleep "$PAUSE"
"$ACPCTL" stop "${HISTORIAN_ID}" || true; echo

sep; bold "▶  Stop poet"; sleep "$PAUSE"
"$ACPCTL" stop "${POET_ID}" || true; echo

sleep 3

step "Delete historian session" \
    "$ACPCTL" delete session "${HISTORIAN_ID}" -y

step "Delete poet session" \
    "$ACPCTL" delete session "${POET_ID}" -y

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
