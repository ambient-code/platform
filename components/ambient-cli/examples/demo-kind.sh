#!/usr/bin/env bash
# demo-kind.sh — acpctl end-to-end demo against a local kind cluster
#
# Layout: left half = demo output, right half = 3 stacked session watch panels.
# As sessions are created, each gets a live `acpctl session messages -f` watch
# in the next available right panel. Up to 3 concurrent session watches.
#
# Usage:
#   ./demo-kind.sh                          # auto-detects kind cluster + token
#   NAMESPACE=my-ns ./demo-kind.sh          # override namespace
#   SKIP_GRPC_WATCH=1 ./demo-kind.sh        # skip grpcurl watch (if not installed)
#
# Optional env:
#   NAMESPACE              — k8s namespace (default: ambient-code)
#   KIND_CONTEXT           — kubectl context (default: kind-ambient-local)
#   API_PORT               — local port for REST API  (default: 18000)
#   GRPC_PORT              — local port for gRPC      (default: 19000)
#   FRONTEND_PORT          — local port for frontend  (default: 18080)
#   ACPCTL                 — path to acpctl binary    (default: acpctl from PATH)
#   PAUSE                  — seconds between demo steps (default: 0)
#   SESSION_READY_TIMEOUT  — seconds to wait for Running (default: 120)
#   MESSAGE_WAIT_TIMEOUT   — seconds to wait for messages (default: 60)
#   SKIP_GRPC_WATCH        — set to 1 to skip grpcurl stream (default: unset)

set -euo pipefail

# ── tmux layout bootstrap ──────────────────────────────────────────────────────
#
# If we are not already inside a tmux session, create one and re-exec this
# script inside the left pane. The right side is pre-split into 3 watch panels
# (right-0, right-1, right-2) which start idle and are claimed as sessions appear.

TMUX_SESSION="ambient-demo"
DEMO_SCRIPT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/$(basename "${BASH_SOURCE[0]}")"

if [[ -z "${TMUX:-}" ]]; then
    tmux new-session -d -s "$TMUX_SESSION" -x 220 -y 50

    # right column: split the window vertically (left=main, right=watches)
    tmux split-window -h -t "${TMUX_SESSION}:0" -p 40
    # right pane is now pane 1 — split into 3 equal rows
    tmux split-window -v -t "${TMUX_SESSION}:0.1" -p 67
    tmux split-window -v -t "${TMUX_SESSION}:0.2" -p 50

    # label each watch pane so the user can see what it will show
    tmux send-keys -t "${TMUX_SESSION}:0.1" "printf '\\033[2m[watch slot 1 — waiting for session]\\033[0m\\n'" Enter
    tmux send-keys -t "${TMUX_SESSION}:0.2" "printf '\\033[2m[watch slot 2 — waiting for session]\\033[0m\\n'" Enter
    tmux send-keys -t "${TMUX_SESSION}:0.3" "printf '\\033[2m[watch slot 3 — waiting for session]\\033[0m\\n'" Enter

    # run this script in the left pane with TMUX_SESSION exported
    tmux send-keys -t "${TMUX_SESSION}:0.0" \
        "TMUX_SESSION=${TMUX_SESSION} INSIDE_DEMO_TMUX=1 bash ${DEMO_SCRIPT}" Enter

    tmux select-pane -t "${TMUX_SESSION}:0.0"
    tmux attach-session -t "$TMUX_SESSION"
    exit 0
fi

# track which right panes are still free (pane indices 1, 2, 3)
WATCH_PANES=(1 2 3)
WATCH_PANE_IDX=0

attach_session_watch() {
    local session_id="$1"
    if [[ $WATCH_PANE_IDX -ge ${#WATCH_PANES[@]} ]]; then
        return
    fi
    local pane="${WATCH_PANES[$WATCH_PANE_IDX]}"
    WATCH_PANE_IDX=$(( WATCH_PANE_IDX + 1 ))
    tmux send-keys -t "${TMUX_SESSION}:0.${pane}" \
        "AMBIENT_API_URL=${AMBIENT_API_URL} AMBIENT_TOKEN=${AMBIENT_TOKEN} AMBIENT_GRPC_URL=${AMBIENT_GRPC_URL} ${ACPCTL:-acpctl} session messages ${session_id} -f" Enter
}

NAMESPACE="${NAMESPACE:-ambient-code}"
KIND_CONTEXT="${KIND_CONTEXT:-$(kubectl config current-context 2>/dev/null | grep -E '^kind-' | head -1)}"
KIND_CONTEXT="${KIND_CONTEXT:-kind-ambient-local}"
API_PORT="${API_PORT:-18000}"
GRPC_PORT="${GRPC_PORT:-19000}"
FRONTEND_PORT="${FRONTEND_PORT:-18080}"
ACPCTL="${ACPCTL:-acpctl}"
PAUSE="${PAUSE:-0}"
SESSION_READY_TIMEOUT="${SESSION_READY_TIMEOUT:-120}"
MESSAGE_WAIT_TIMEOUT="${MESSAGE_WAIT_TIMEOUT:-60}"
SKIP_GRPC_WATCH="${SKIP_GRPC_WATCH:-}"

KUBECTL="kubectl --context=${KIND_CONTEXT}"

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

if ! command -v kubectl &>/dev/null; then
    red "error: kubectl not found" >&2; exit 1
fi
if ! command -v "$ACPCTL" &>/dev/null; then
    red "error: ${ACPCTL} not found. Set ACPCTL=/path/to/acpctl or add to PATH." >&2; exit 1
fi
if ! kubectl --context="${KIND_CONTEXT}" cluster-info &>/dev/null 2>&1; then
    red "error: kind context '${KIND_CONTEXT}' not available. Is the cluster running?" >&2
    echo
    dim "  Available kind contexts:"
    kubectl config get-contexts -o name 2>/dev/null | grep '^kind-' | sed 's/^/    /' || true
    dim "  Set KIND_CONTEXT=<context> to override."
    exit 1
fi

# ── port-forwards ──────────────────────────────────────────────────────────────

PF_PIDS=()

start_port_forward() {
    local label="$1" svc="$2" local_port="$3" svc_port="$4"
    $KUBECTL port-forward "svc/${svc}" "${local_port}:${svc_port}" \
        -n "${NAMESPACE}" >/dev/null 2>&1 &
    PF_PIDS+=($!)
    dim "   port-forward ${label}: localhost:${local_port} → ${svc}:${svc_port}"
}

wait_for_port() {
    local port="$1" label="$2" deadline=$(( $(date +%s) + 15 ))
    while ! bash -c "echo >/dev/tcp/localhost/${port}" 2>/dev/null; do
        if [[ $(date +%s) -ge $deadline ]]; then
            yellow "   ✗ timeout waiting for ${label} on :${port}"
            return 1
        fi
        sleep 0.3
    done
    green "   ✓ ${label} ready on localhost:${port}"
}

cleanup() {
    echo
    dim "stopping port-forwards..."
    for pid in "${PF_PIDS[@]:-}"; do
        kill "$pid" 2>/dev/null || true
    done
    if [[ -n "${GRPC_WATCH_PID:-}" ]]; then
        kill "$GRPC_WATCH_PID" 2>/dev/null || true
    fi
}
trap cleanup EXIT

# ── intro ─────────────────────────────────────────────────────────────────────

echo
bold "Ambient CLI Demo (kind)"
dim  "  Context:  ${KIND_CONTEXT}"
dim  "  API:      http://127.0.0.1:${API_PORT}"
dim  "  gRPC:     localhost:${GRPC_PORT}"
dim  "  Frontend: http://localhost:${FRONTEND_PORT}"

echo
sep
bold "What this demo will do:"
echo
printf '  %s\n' "1. Log in and create a project"
printf '  %s\n' "2. Create a session and wait for the runner pod to start"
printf '  %s\n' "3. Have a 3-turn conversation about Charleston, SC:"
printf '  %s\n' "     Turn 1 — set context: living in Charleston with marshes and beaches"
printf '  %s\n' "     Turn 2 — ask the bot what to do outdoors near Charleston"
printf '  %s\n' "     Turn 3 — ask the bot to write a 4-line Charleston poem to a file"
printf '  %s\n' "4. Verify the poem file exists inside the runner pod"
printf '  %s\n' "5. Show final session state and clean up"
echo
printf '  \033[38;5;214m%-38s\033[0m %s\n' "Orange text like this" "= a terminal command being run"
echo
sep
if [[ "${PAUSE}" -gt 0 ]] 2>/dev/null; then
    bold "   Press Enter to begin..."
    read -r
fi

announce "0 · Starting port-forwards"

start_port_forward "REST API"  ambient-api-server "${API_PORT}"      8000
start_port_forward "gRPC"      ambient-api-server "${GRPC_PORT}"     9000
start_port_forward "Frontend"  frontend-service   "${FRONTEND_PORT}" 3000

sleep 1

wait_for_port "${API_PORT}"      "REST API"
wait_for_port "${GRPC_PORT}"     "gRPC"
wait_for_port "${FRONTEND_PORT}" "Frontend"

export AMBIENT_GRPC_URL="localhost:${GRPC_PORT}"
dim "   AMBIENT_GRPC_URL=${AMBIENT_GRPC_URL}"

# ── token from cluster ─────────────────────────────────────────────────────────

AMBIENT_TOKEN=$(
    $KUBECTL get secret test-user-token -n "${NAMESPACE}" \
        -o jsonpath='{.data.token}' 2>/dev/null | base64 -d
)
if [[ -z "${AMBIENT_TOKEN}" ]]; then
    red "error: could not read test-user-token from namespace ${NAMESPACE}" >&2
    exit 1
fi

AMBIENT_API_URL="http://127.0.0.1:${API_PORT}"
RUN_ID=$(date +%s | tail -c5)
PROJECT_NAME="demo-${RUN_ID}"

# ── gRPC session watch ─────────────────────────────────────────────────────────

GRPC_WATCH_LOG=$(mktemp /tmp/grpc-watch-XXXXXX.log)

if [[ -z "${SKIP_GRPC_WATCH}" ]] && command -v grpcurl &>/dev/null; then
    announce "0b · Starting gRPC WatchSessions stream"
    grpcurl -plaintext \
        -H "authorization: Bearer ${AMBIENT_TOKEN}" \
        -d '{}' \
        "localhost:${GRPC_PORT}" \
        ambient.v1.SessionService/WatchSessions \
        >> "${GRPC_WATCH_LOG}" 2>&1 &
    GRPC_WATCH_PID=$!
    green "   ✓ gRPC watch running (PID ${GRPC_WATCH_PID}) → ${GRPC_WATCH_LOG}"
    dim   "   tail -f ${GRPC_WATCH_LOG}   (in another terminal)"
else
    GRPC_WATCH_PID=""
    yellow "   skipping gRPC watch (grpcurl not found or SKIP_GRPC_WATCH=1)"
fi

# ── api helpers ────────────────────────────────────────────────────────────────

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
    printf '   waiting for Running (timeout %ds)...\n' "${SESSION_READY_TIMEOUT}"
    while true; do
        local phase
        phase=$(
            "$ACPCTL" get session "$session_id" -o json 2>/dev/null \
            | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('phase',''))" 2>/dev/null || true
        )
        if [[ "$phase" != "$last_phase" ]]; then
            printf '   phase: %s\n' "$phase"
            last_phase="$phase"
            if [[ -s "${GRPC_WATCH_LOG}" ]]; then
                dim "   [gRPC] $(tail -1 "${GRPC_WATCH_LOG}")"
            fi
        fi
        [[ "$phase" == "Running" ]] && { green "   ✓ session is Running"; return 0; }
        [[ $(date +%s) -ge $deadline ]] && { yellow "   ✗ timed out (phase=${phase:-unknown})"; return 1; }
        sleep 3
    done
}

wait_for_run_finished() {
    local session_id="$1" after_seq="$2"
    local start=$(date +%s)
    local status="none"
    local matched_type=""

    while IFS= read -r line; do
        if echo "$line" | grep -qE '^\[|RUN_FINISHED|RUN_ERROR'; then
            if echo "$line" | grep -q 'RUN_FINISHED'; then
                status="finished"; matched_type="RUN_FINISHED"; break
            elif echo "$line" | grep -q 'RUN_ERROR'; then
                status="error"; matched_type="RUN_ERROR"; break
            fi
        fi
    done < <(timeout "${MESSAGE_WAIT_TIMEOUT}" "${ACPCTL:-acpctl}" session messages "${session_id}" -f --after "${after_seq}" 2>/dev/null)

    local elapsed=$(( $(date +%s) - start ))
    case "$status" in
        finished) green   "   ✓ ${matched_type} (${elapsed}s)"; return 0 ;;
        error)    yellow  "   ✗ ${matched_type} (${elapsed}s)"; return 1 ;;
        *)        yellow  "   ✗ timeout after ${MESSAGE_WAIT_TIMEOUT}s";  return 1 ;;
    esac
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

# ── section 0: login ───────────────────────────────────────────────────────────

announce "1 · Log in"

step "Log in to the Ambient API server" \
    "$ACPCTL" login "${AMBIENT_API_URL}" \
        --token "${AMBIENT_TOKEN}" \
        --insecure-skip-tls-verify

step "Show authenticated user" \
    "$ACPCTL" whoami

# ── section 1: project ────────────────────────────────────────────────────────

announce "2 · Create project"

step "Create project: ${PROJECT_NAME}" \
    "$ACPCTL" create project \
        --name "${PROJECT_NAME}" \
        --display-name "Demo Project ${RUN_ID}" \
        --description "kind demo"

step "Set project context" \
    "$ACPCTL" project "${PROJECT_NAME}"

step "Confirm project context" \
    "$ACPCTL" project current

# ── section 2: session ────────────────────────────────────────────────────────

announce "3 · Create session"

sep; bold "▶  Create smoke-session"; sleep "$PAUSE"
SESSION_JSON=$(
    "$ACPCTL" create session \
        --name smoke-session \
        -o json 2>/dev/null
)
SESSION_ID=$(echo "$SESSION_JSON" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])" 2>/dev/null)
if [[ -z "$SESSION_ID" ]]; then
    red "   ✗ failed to parse session ID"
    exit 1
fi
dim "   session ID: ${SESSION_ID}"; echo

attach_session_watch "${SESSION_ID}"

if [[ -n "${GRPC_WATCH_PID:-}" ]]; then
    sleep 1
    echo
    cyan "── gRPC events so far ──"
    cat "${GRPC_WATCH_LOG}" 2>/dev/null || true
    sep
fi

step "List sessions" \
    "$ACPCTL" get sessions

step "Describe smoke-session" \
    "$ACPCTL" describe session "${SESSION_ID}"

# ── section 3: wait for Running ───────────────────────────────────────────────

announce "4 · Wait for session Running"

wait_for_running "${SESSION_ID}" || true

if [[ -n "${GRPC_WATCH_PID:-}" ]]; then
    echo
    cyan "── gRPC events ──"
    cat "${GRPC_WATCH_LOG}" 2>/dev/null || true
    sep
fi

# ── section 4: multi-turn conversation ────────────────────────────────────────

announce "5 · Multi-turn conversation"

send_turn() {
    local turn="$1" msg="$2"
    local before_seq
    before_seq=$(max_seq "${SESSION_ID}")

    echo; sep
    bold "▶  Turn ${turn}: sending message"
    dim  "   ${msg}"
    sleep "$PAUSE"
    "$ACPCTL" session send "${SESSION_ID}" "$msg"
    echo

    bold "▶  Turn ${turn}: waiting for response..."
    wait_for_run_finished "${SESSION_ID}" "${before_seq}" || true

    echo
    bold "▶  Turn ${turn}: bot response"
    api_get "/sessions/${SESSION_ID}/messages?after_seq=${before_seq}" \
    | python3 -c "
import sys, json
try:
    msgs = json.load(sys.stdin)
    # find the last MESSAGES_SNAPSHOT and extract the last assistant message
    snapshot = None
    for m in reversed(msgs):
        if m.get('event_type') == 'MESSAGES_SNAPSHOT':
            snapshot = json.loads(m.get('payload', '[]'))
            break
    if snapshot:
        for msg in reversed(snapshot):
            if msg.get('role') == 'assistant':
                content = msg.get('content', '')
                if isinstance(content, list):
                    content = ' '.join(p.get('text','') for p in content if isinstance(p, dict))
                print(content)
                break
except Exception:
    pass
" 2>/dev/null || true
    echo

    step "Turn ${turn}: conversation so far" \
        "$ACPCTL" session messages "${SESSION_ID}"
}

send_turn 1 "I live in Charleston, SC — green salt marshes, warm beaches, and Spanish moss everywhere. Keep that in mind."
send_turn 2 "What are some things I might enjoy doing outdoors near Charleston given where I live?"
send_turn 3 "create a file called charleston.txt in the workspace with a short 4-line poem about Charleston, SC — mention the marshes and the sea"

# ── verify tool use: file created in runner pod ────────────────────────────────

announce "5b · Verify file created in runner pod"

sleep 5
RUNNER_POD_INFO=$($KUBECTL get pods -A -l "ambient-code.io/session-id=${SESSION_ID}" --no-headers 2>/dev/null | head -1)
RUNNER_NS=$(echo "$RUNNER_POD_INFO" | awk '{print $1}')
RUNNER_POD=$(echo "$RUNNER_POD_INFO" | awk '{print $2}')

if [[ -n "$RUNNER_POD" ]]; then
    dim "   runner pod: ${RUNNER_NS}/${RUNNER_POD}"
    step "cat /workspace/artifacts/charleston.txt" \
        $KUBECTL exec -n "${RUNNER_NS}" "${RUNNER_POD}" -- cat /workspace/artifacts/charleston.txt
else
    yellow "   runner pod not found — skipping file check"
fi

# ── section 6: final state ────────────────────────────────────────────────────

announce "6 · Final state"

step "Session detail" \
    "$ACPCTL" describe session "${SESSION_ID}"

step "All messages" \
    "$ACPCTL" session messages "${SESSION_ID}"

if [[ -n "${GRPC_WATCH_PID:-}" ]]; then
    echo
    cyan "── gRPC events (full log) ──"
    cat "${GRPC_WATCH_LOG}" 2>/dev/null || true
    sep
fi

# ── section 7: cleanup ────────────────────────────────────────────────────────

announce "7 · Stop and clean up"

sep; bold "▶  Stop smoke-session"; sleep "$PAUSE"
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
dim   "  gRPC log: ${GRPC_WATCH_LOG}"
sep
echo
