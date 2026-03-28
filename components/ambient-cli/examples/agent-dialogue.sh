#!/usr/bin/env bash
# agent-dialogue.sh — two running sessions converse with each other via acpctl session send
#
# Each session plays a role (Designer, Critic) and responds to the other's last message.
# The topic: designing the TUI interface for watching agentic session messages.
#
# Usage:
#   SESSION_A=<id> SESSION_B=<id> ./agent-dialogue.sh
#   ./agent-dialogue.sh   # uses the two hardcoded IDs below as defaults
#
# Optional env:
#   SESSION_A              — ID of agent A (role: Designer)
#   SESSION_B              — ID of agent B (role: Critic)
#   TURNS                  — number of back-and-forth exchanges (default: 4)
#   MESSAGE_WAIT_TIMEOUT   — seconds to wait for each response (default: 90)
#   API_PORT               — local REST API port (default: 18000)
#   NAMESPACE              — k8s namespace (default: ambient-code)
#   KIND_CONTEXT           — kubectl context (default: auto-detect)

set -euo pipefail

SESSION_A="${SESSION_A:-3B7qaIYmX7K3jhfItFAYViYydwE}"
SESSION_B="${SESSION_B:-3B7qaPCcv3Ro3lCSqJf3OCRSEfB}"
TURNS="${TURNS:-4}"
MESSAGE_WAIT_TIMEOUT="${MESSAGE_WAIT_TIMEOUT:-90}"
API_PORT="${API_PORT:-18000}"
NAMESPACE="${NAMESPACE:-ambient-code}"
KIND_CONTEXT="${KIND_CONTEXT:-$(kubectl config current-context 2>/dev/null | head -1)}"
KIND_CONTEXT="${KIND_CONTEXT:-kind-ambient-local}"
ACPCTL="${ACPCTL:-acpctl}"

KUBECTL="kubectl --context=${KIND_CONTEXT}"

# ── colors ─────────────────────────────────────────────────────────────────────

bold()   { printf '\033[1m%s\033[0m\n' "$*"; }
dim()    { printf '\033[2m%s\033[0m\n' "$*"; }
cyan()   { printf '\033[36m%s\033[0m\n' "$*"; }
green()  { printf '\033[32m%s\033[0m\n' "$*"; }
yellow() { printf '\033[33m%s\033[0m\n' "$*"; }
red()    { printf '\033[31m%s\033[0m\n' "$*"; }
magenta(){ printf '\033[35m%s\033[0m\n' "$*"; }
sep()    { printf '\033[2m%s\033[0m\n' "──────────────────────────────────────────────────"; }

# ── helpers ────────────────────────────────────────────────────────────────────

max_seq() {
    local session_id="$1"
    "$ACPCTL" session messages "${session_id}" -o json 2>/dev/null \
    | python3 -c "
import sys, json
try:
    msgs = json.load(sys.stdin)
    seqs = [m.get('seq', 0) for m in msgs] if isinstance(msgs, list) else [0]
    print(max(seqs, default=0))
except Exception:
    print(0)
" 2>/dev/null || echo 0
}

wait_for_response() {
    local session_id="$1" after_seq="$2"
    local start=$(date +%s)
    local deadline=$(( start + MESSAGE_WAIT_TIMEOUT ))
    local spinner='⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏'
    local spin_i=0
    printf '   '
    while true; do
        local result
        result=$(
            "$ACPCTL" session messages "${session_id}" --after "${after_seq}" -o json 2>/dev/null \
            | python3 -c "
import sys, json
try:
    msgs = json.load(sys.stdin)
    if not isinstance(msgs, list):
        print('none')
    else:
        types = [m.get('event_type','') for m in msgs]
        if 'MESSAGES_SNAPSHOT' in types or 'RUN_FINISHED' in types:
            print('finished')
        elif 'RUN_ERROR' in types:
            print('error')
        else:
            print('none')
except Exception:
    print('none')
" 2>/dev/null || echo none
        )
        local elapsed=$(( $(date +%s) - start ))
        case "$result" in
            finished)
                printf '\r'
                green "   ✓ response received (${elapsed}s)"; return 0 ;;
            error)
                printf '\r'
                yellow "   ✗ RUN_ERROR (${elapsed}s)"; return 1 ;;
        esac
        [[ $(date +%s) -ge $deadline ]] && { printf '\r'; yellow "   ✗ timeout after ${MESSAGE_WAIT_TIMEOUT}s"; return 1; }
        local ch="${spinner:$(( spin_i % ${#spinner} )):1}"
        printf "\r   %s %ds" "$ch" "$elapsed"
        spin_i=$(( spin_i + 1 ))
        sleep 2
    done
}

extract_last_assistant_message() {
    local session_id="$1" after_seq="$2"
    "$ACPCTL" session messages "${session_id}" --after "${after_seq}" -o json 2>/dev/null \
    | python3 -c "
import sys, json
try:
    msgs = json.load(sys.stdin)
    for m in reversed(msgs):
        if m.get('event_type') == 'MESSAGES_SNAPSHOT':
            payload = m.get('payload', '[]')
            try:
                payload = json.loads(payload)
            except Exception:
                pass
            if isinstance(payload, str):
                try:
                    payload = json.loads(payload)
                except Exception:
                    pass
            if isinstance(payload, list):
                for msg in reversed(payload):
                    if msg.get('role') == 'assistant':
                        content = msg.get('content', '')
                        if isinstance(content, list):
                            content = ' '.join(p.get('text','') for p in content if isinstance(p, dict))
                        print(content.strip())
                        sys.exit(0)
    print('(no assistant response found)')
except Exception as e:
    print(f'(error: {e})')
" 2>/dev/null || echo "(could not extract response)"
}

# ── intro ──────────────────────────────────────────────────────────────────────

echo
bold "Agent Dialogue: TUI Design Discussion"
sep
dim "  Session A (Designer): ${SESSION_A}"
dim "  Session B (Critic):   ${SESSION_B}"
dim "  Turns: ${TURNS} exchanges"
dim "  Topic: designing the TUI interface for watching agentic session messages"
sep
echo

# ── opening prompt ─────────────────────────────────────────────────────────────

bold "Seeding the conversation..."
echo

OPENING_PROMPT="You are 'Designer', one of two AI agents in a live dialogue. The other agent is 'Critic'. \
You are both looking at a terminal TUI (built with Bubbletea in Go) that displays agentic session messages in real time. \
The TUI shows a sidebar with sessions, and tiles in the main panel — each tile shows a session's recent messages: \
timestamp, event type (color-coded), and a single-line payload preview. \
Messages types include: user, MESSAGES_SNAPSHOT, RUN_FINISHED, RUN_ERROR, TOOL_CALL_START, TOOL_CALL_RESULT. \
\
Start the dialogue: propose ONE concrete improvement to how the TUI currently displays messages. \
Keep your response under 5 sentences. Sign it: —Designer"

CRITIC_CONTEXT="You are 'Critic', one of two AI agents in a live dialogue. The other agent is 'Designer'. \
You are both discussing a terminal TUI (built with Bubbletea in Go) that shows agentic session messages. \
The TUI shows session tiles with: timestamp, event type, and a truncated single-line payload preview. \
When the Designer proposes a change, you respond with a counter-point, refinement, or concern — then add your own small improvement idea. \
Keep your response under 5 sentences. Sign it: —Critic"

# seed Designer
dim "  Sending opening prompt to Designer (A)..."
SEQ_A_BEFORE=$(max_seq "${SESSION_A}")
"$ACPCTL" session send "${SESSION_A}" "${OPENING_PROMPT}" >/dev/null
wait_for_response "${SESSION_A}" "${SEQ_A_BEFORE}" || true
DESIGNER_REPLY=$(extract_last_assistant_message "${SESSION_A}" "${SEQ_A_BEFORE}")

echo
cyan "━━  Designer (turn 0) ━━"
echo "${DESIGNER_REPLY}"
sep

# ── dialogue loop ──────────────────────────────────────────────────────────────

LAST_REPLY="${DESIGNER_REPLY}"
CURRENT_SENDER="B"

for (( turn=1; turn<=TURNS; turn++ )); do
    echo

    if [[ "${CURRENT_SENDER}" == "B" ]]; then
        # Critic's turn
        bold "Turn ${turn}: Critic (B) responds..."
        SEQ_B_BEFORE=$(max_seq "${SESSION_B}")
        PROMPT="${CRITIC_CONTEXT}

Designer just said:
\"${LAST_REPLY}\"

Respond as Critic."
        "$ACPCTL" session send "${SESSION_B}" "${PROMPT}" >/dev/null
        wait_for_response "${SESSION_B}" "${SEQ_B_BEFORE}" || true
        REPLY=$(extract_last_assistant_message "${SESSION_B}" "${SEQ_B_BEFORE}")
        echo
        magenta "━━  Critic (turn ${turn}) ━━"
        echo "${REPLY}"
        sep
        CURRENT_SENDER="A"
    else
        # Designer's turn
        bold "Turn ${turn}: Designer (A) responds..."
        SEQ_A_BEFORE=$(max_seq "${SESSION_A}")
        PROMPT="You are 'Designer' in a dialogue with 'Critic' about a terminal TUI for agentic session messages.

Critic just said:
\"${LAST_REPLY}\"

Respond as Designer — acknowledge the critique, refine your idea or defend it, and propose one more improvement. Under 5 sentences. Sign it: —Designer"
        "$ACPCTL" session send "${SESSION_A}" "${PROMPT}" >/dev/null
        wait_for_response "${SESSION_A}" "${SEQ_A_BEFORE}" || true
        REPLY=$(extract_last_assistant_message "${SESSION_A}" "${SEQ_A_BEFORE}")
        echo
        cyan "━━  Designer (turn ${turn}) ━━"
        echo "${REPLY}"
        sep
        CURRENT_SENDER="B"
    fi

    LAST_REPLY="${REPLY}"
done

# ── summary ────────────────────────────────────────────────────────────────────

echo
bold "Dialogue complete. Asking Designer for a summary..."
SEQ_A_BEFORE=$(max_seq "${SESSION_A}")
"$ACPCTL" session send "${SESSION_A}" \
    "Summarize the key TUI improvement ideas from this dialogue in 3 bullet points. Sign it: —Designer" >/dev/null
wait_for_response "${SESSION_A}" "${SEQ_A_BEFORE}" || true
SUMMARY=$(extract_last_assistant_message "${SESSION_A}" "${SEQ_A_BEFORE}")

echo
cyan "━━  Summary (Designer) ━━"
echo "${SUMMARY}"
sep
echo
green "  Dialogue complete ✓"
echo
