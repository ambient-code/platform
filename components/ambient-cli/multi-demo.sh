#!/usr/bin/env bash
# multi-demo.sh — create a 4-panel tmux session for a lead/fe/api/cp agent group discussion
#
# Usage:
#   ./multi-demo.sh [--project-id <id>] [--session <tmux-session-name>]
#
# Requires: acpctl, jq, tmux

set -euo pipefail

TMUX_SESSION="${TMUX_SESSION:-ambient-demo}"
PROJECT_ID="${PROJECT_ID:-}"

while [[ $# -gt 0 ]]; do
  case $1 in
    --project-id) PROJECT_ID="$2"; shift 2 ;;
    --session)    TMUX_SESSION="$2"; shift 2 ;;
    *) echo "Unknown argument: $1"; exit 1 ;;
  esac
done

# ── 1. Resolve project ────────────────────────────────────────────────────────

if [[ -z "$PROJECT_ID" ]]; then
  PROJECT_ID=$(acpctl project current 2>/dev/null | awk '{print $NF}')
fi
if [[ -z "$PROJECT_ID" ]]; then
  echo "error: no project set — use --project-id or: acpctl project set <name>" >&2
  exit 1
fi
echo "Project: $PROJECT_ID"

# ── 2. Fetch sessions and resolve lead/fe/api/cp ──────────────────────────────

SESSIONS_JSON=$(acpctl get sessions -o json 2>/dev/null)

resolve_session() {
  local prefix=$1
  echo "$SESSIONS_JSON" | jq -r --arg p "$prefix" \
    '.items[] | select(.name | startswith($p)) | .id' | head -1
}

resolve_agent() {
  local prefix=$1
  echo "$SESSIONS_JSON" | jq -r --arg p "$prefix" \
    '.items[] | select(.name | startswith($p)) | .agent_id' | head -1
}

# Resolve or start a session for the given agent name prefix.
# Prints the session ID; starts the agent if no session exists.
resolve_or_start_session() {
  local prefix=$1
  local sid
  sid=$(resolve_session "$prefix")
  if [[ -n "$sid" ]]; then
    echo "$sid"
    return
  fi
  echo "No session found for '$prefix' — starting agent..." >&2
  local out
  out=$(acpctl agent start "$prefix" --project-id "$PROJECT_ID" 2>&1) || {
    echo "error: failed to start agent '$prefix': $out" >&2
    exit 1
  }
  # output format: "session/<id> started (phase: ...)"
  echo "$out" | sed -n 's|^session/\([^ ]*\) .*|\1|p'
}

LEAD_SID=$(resolve_or_start_session "lead")
API_SID=$(resolve_or_start_session "api")
FE_SID=$(resolve_or_start_session "fe")
CP_SID=$(resolve_or_start_session "cp")

# Refresh session list so agent_id fields are available for newly started sessions
SESSIONS_JSON=$(acpctl get sessions -o json 2>/dev/null)

LEAD_AID=$(resolve_agent "lead")
API_AID=$(resolve_agent "api")
FE_AID=$(resolve_agent "fe")
CP_AID=$(resolve_agent "cp")

echo "Sessions found:"
echo "  lead  session=$LEAD_SID  agent=$LEAD_AID"
echo "  api   session=$API_SID   agent=$API_AID"
echo "  fe    session=$FE_SID    agent=$FE_AID"
echo "  cp    session=$CP_SID    agent=$CP_AID"

# ── 3. Build initial lead message ─────────────────────────────────────────────

read -r -d '' LEAD_MSG <<EOF || true
You are the lead agent coordinating a multi-agent team in project '${PROJECT_ID}'.
Your team members and their agent IDs are:
  - fe  (frontend):       ${FE_AID}
  - api (backend API):    ${API_AID}
  - cp  (control-plane):  ${CP_AID}

Your task for this session:

1. First, examine your own annotations using available tools (e.g. acpctl or kubectl).
   Report what you find — especially any communication protocol or coordination
   conventions encoded there.

2. Send an inbox message to each team member asking them to:
   a) Examine their own annotations and report what they mean.
   b) Reply to you (lead) with a summary.
   c) If they find a communication protocol in the annotations, use it going forward.

   Use: acpctl inbox send --project-id ${PROJECT_ID} --pa-id <agent-id> --body "<message>" --from-name lead

3. After each agent replies, synthesize what you learn and send a follow-up directing
   each agent on next steps based on their annotations and any discovered protocol.

Begin now. Send the inbox messages to your team first, then examine your own annotations.
EOF

# ── 4. Kill any existing tmux session with the same name ─────────────────────

if tmux has-session -t "$TMUX_SESSION" 2>/dev/null; then
  echo "Killing existing tmux session '$TMUX_SESSION'..."
  tmux kill-session -t "$TMUX_SESSION"
fi

# ── 5. Create layout ──────────────────────────────────────────────────────────
#
#  ┌───────────────────┬──────────────────┐
#  │                   │  lead  (pane 1)  │
#  │                   ├──────────────────┤
#  │  interactive      │  api   (pane 2)  │
#  │  (pane 0)         ├──────────────────┤
#  │                   │  cp    (pane 3)  │
#  │                   ├──────────────────┤
#  │                   │  fe    (pane 4)  │
#  └───────────────────┴──────────────────┘

tmux new-session -d -s "$TMUX_SESSION" -x 220 -y 55

# Split right column off pane 0
tmux split-window -h -t "${TMUX_SESSION}:0.0"   # pane 0=left, pane 1=right

# Stack 3 more panes in the right column by splitting pane 1 repeatedly
tmux split-window -v -t "${TMUX_SESSION}:0.1"   # pane 2 below pane 1
tmux split-window -v -t "${TMUX_SESSION}:0.2"   # pane 3 below pane 2
tmux split-window -v -t "${TMUX_SESSION}:0.3"   # pane 4 below pane 3

# Set pane titles
tmux select-pane -t "${TMUX_SESSION}:0.0" -T "INTERACTIVE"
tmux select-pane -t "${TMUX_SESSION}:0.1" -T "lead msgs"
tmux select-pane -t "${TMUX_SESSION}:0.2" -T "api msgs"
tmux select-pane -t "${TMUX_SESSION}:0.3" -T "cp msgs"
tmux select-pane -t "${TMUX_SESSION}:0.4" -T "fe msgs"

# ── 6. Start message watchers in right column ─────────────────────────────────

tmux send-keys -t "${TMUX_SESSION}:0.1" \
  "echo '=== lead: ${LEAD_SID} ===' && acpctl session messages '${LEAD_SID}' -f" Enter

tmux send-keys -t "${TMUX_SESSION}:0.2" \
  "echo '=== api: ${API_SID} ===' && acpctl session messages '${API_SID}' -f" Enter

tmux send-keys -t "${TMUX_SESSION}:0.3" \
  "echo '=== cp: ${CP_SID} ===' && acpctl session messages '${CP_SID}' -f" Enter

tmux send-keys -t "${TMUX_SESSION}:0.4" \
  "echo '=== fe: ${FE_SID} ===' && acpctl session messages '${FE_SID}' -f" Enter

# ── 7. Left pane: send initial message, then stay interactive ─────────────────

tmux send-keys -t "${TMUX_SESSION}:0.0" \
  "echo '=== interactive — lead session: ${LEAD_SID} ==='" Enter

# Write message to a temp file to avoid quoting nightmares
TMPFILE=$(mktemp /tmp/lead-msg-XXXXXX.txt)
printf '%s' "$LEAD_MSG" > "$TMPFILE"

tmux send-keys -t "${TMUX_SESSION}:0.0" \
  "acpctl session send '${LEAD_SID}' \"\$(cat ${TMPFILE})\"; rm -f ${TMPFILE}" Enter

# Focus the lead pane after startup
tmux select-pane -t "${TMUX_SESSION}:0.0"

# ── 8. Attach ─────────────────────────────────────────────────────────────────

echo ""
echo "Attaching to tmux session '$TMUX_SESSION'..."
echo "  Ctrl+B D  — detach"
echo "  Ctrl+B arrow — switch pane"
echo ""
echo "Lead pane shortcuts (once interactive):"
echo "  acpctl session send '${LEAD_SID}' \"your message\""
echo "  acpctl inbox send --project-id '${PROJECT_ID}' --pa-id <agent-id> --body \"...\""
echo ""

tmux attach-session -t "$TMUX_SESSION"
