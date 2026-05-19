#!/usr/bin/env bash
# demo-credentials.sh — acpctl credential lifecycle demo
#
# Demonstrates the full credential workflow:
#   1. Verify login
#   2. Create three credentials (GitHub, GitLab, Jira)
#   3. Create a project
#   4. Bind all credentials to the project
#   5. Create an agent and start a session
#   6. Ask the agent to verify it can access the credentials
#   7. Clean up
#
# Prerequisites:
#   You must create three secret files before running this demo:
#
#     .secrets/GITHUB_TOKEN   — a GitHub Personal Access Token (classic or fine-grained)
#                               Create at: https://github.com/settings/tokens
#                               Required scopes: repo (or fine-grained with Contents read)
#
#     .secrets/GITLAB_TOKEN   — a GitLab Personal Access Token
#                               Create at: https://gitlab.com/-/user_settings/personal_access_tokens
#                               Required scopes: read_api
#
#     .secrets/JIRA_TOKEN     — a Jira API Token
#                               Create at: https://id.atlassian.com/manage-profile/security/api-tokens
#                               Used with your Atlassian email for Basic auth
#
#   Each file should contain the raw token string with no trailing newline.
#   Example:
#     echo -n "ghp_abc123..." > .secrets/GITHUB_TOKEN
#     echo -n "glpat-xyz..."  > .secrets/GITLAB_TOKEN
#     echo -n "ATATT3x..."    > .secrets/JIRA_TOKEN
#     chmod 600 .secrets/*
#
# Usage:
#   ./demo-credentials.sh
#   PAUSE=2 ./demo-credentials.sh          # pause between steps
#   SECRETS_DIR=~/my-secrets ./demo-credentials.sh
#   NO_CLEANUP=1 ./demo-credentials.sh     # skip cleanup
#
# Optional env:
#   SECRETS_DIR             — directory containing token files (default: .secrets)
#   JIRA_URL                — Jira instance URL (default: prompted)
#   JIRA_EMAIL              — Jira account email (default: prompted)
#   GITLAB_URL              — GitLab instance URL (default: https://gitlab.com)
#   ACPCTL                  — path to acpctl binary (default: acpctl from PATH)
#   PAUSE                   — seconds between demo steps (default: 0)
#   SESSION_READY_TIMEOUT   — seconds to wait for Running (default: 180)
#   MESSAGE_WAIT_TIMEOUT    — seconds to wait for RUN_FINISHED (default: 300)
#   NO_CLEANUP              — set to 1 to skip cleanup

set -euo pipefail

ACPCTL="${ACPCTL:-acpctl}"
PAUSE="${PAUSE:-0}"
SESSION_READY_TIMEOUT="${SESSION_READY_TIMEOUT:-180}"
MESSAGE_WAIT_TIMEOUT="${MESSAGE_WAIT_TIMEOUT:-300}"
SECRETS_DIR="${SECRETS_DIR:-.secrets}"
GITLAB_URL="${GITLAB_URL:-https://gitlab.com}"

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
        [[ "$phase" == "Running" ]] && { green "   session is Running"; return 0; }
        [[ $(date +%s) -ge $deadline ]] && { yellow "   timed out (phase=${phase:-unknown})"; return 1; }
        sleep 3
    done
}

# ── preflight ──────────────────────────────────────────────────────────────────

command -v "$ACPCTL" &>/dev/null || die "${ACPCTL} not found. Set ACPCTL=/path/to/acpctl or add to PATH."
command -v python3   &>/dev/null || die "python3 not found."

# ── intro ────────────────────────────────────────────────────────────────────

echo
bold "Ambient CLI Demo — Credential Lifecycle"
sep
echo
printf '  %s\n' "This demo creates three credentials (GitHub, GitLab, Jira),"
printf '  %s\n' "binds them to a project, starts an agent session, and verifies"
printf '  %s\n' "the agent can access all three credentials at runtime."
echo
printf '  %s\n' "Steps:"
printf '  %s\n' "  1. Verify login"
printf '  %s\n' "  2. Create credentials: github-pat, gitlab-pat, jira-token"
printf '  %s\n' "  3. Create project"
printf '  %s\n' "  4. Bind all credentials to the project"
printf '  %s\n' "  5. Create agent + start session"
printf '  %s\n' "  6. Verify credentials in session"
printf '  %s\n' "  7. Clean up"
echo
printf '  \033[38;5;214m%-38s\033[0m %s\n' "Orange text like this" "= a terminal command being run"
echo
sep
echo
bold "  Prerequisites — secret files"
echo
printf '  %s\n' "The demo reads tokens from files in ${SECRETS_DIR}/:"
echo
printf '  \033[36m%-28s\033[0m %s\n' "${SECRETS_DIR}/GITHUB_TOKEN" "GitHub PAT  (https://github.com/settings/tokens)"
printf '  \033[36m%-28s\033[0m %s\n' "${SECRETS_DIR}/GITLAB_TOKEN" "GitLab PAT  (https://gitlab.com/-/user_settings/personal_access_tokens)"
printf '  \033[36m%-28s\033[0m %s\n' "${SECRETS_DIR}/JIRA_TOKEN"   "Jira token  (https://id.atlassian.com/manage-profile/security/api-tokens)"
echo
printf '  %s\n' "Create them like this:"
dim   "    echo -n \"ghp_abc123...\" > ${SECRETS_DIR}/GITHUB_TOKEN"
dim   "    echo -n \"glpat-xyz...\"  > ${SECRETS_DIR}/GITLAB_TOKEN"
dim   "    echo -n \"ATATT3x...\"    > ${SECRETS_DIR}/JIRA_TOKEN"
dim   "    chmod 600 ${SECRETS_DIR}/*"
echo
sep

# ── validate secrets ─────────────────────────────────────────────────────────

GITHUB_TOKEN_FILE="${SECRETS_DIR}/GITHUB_TOKEN"
GITLAB_TOKEN_FILE="${SECRETS_DIR}/GITLAB_TOKEN"
JIRA_TOKEN_FILE="${SECRETS_DIR}/JIRA_TOKEN"

[[ -f "${GITHUB_TOKEN_FILE}" ]] || die "Missing ${GITHUB_TOKEN_FILE} — see prerequisites above."
[[ -f "${GITLAB_TOKEN_FILE}" ]] || die "Missing ${GITLAB_TOKEN_FILE} — see prerequisites above."
[[ -f "${JIRA_TOKEN_FILE}" ]]   || die "Missing ${JIRA_TOKEN_FILE} — see prerequisites above."

GITHUB_TOKEN_VALUE="$(cat "${GITHUB_TOKEN_FILE}")"
GITLAB_TOKEN_VALUE="$(cat "${GITLAB_TOKEN_FILE}")"
JIRA_TOKEN_VALUE="$(cat "${JIRA_TOKEN_FILE}")"

[[ -n "${GITHUB_TOKEN_VALUE}" ]] || die "${GITHUB_TOKEN_FILE} is empty."
[[ -n "${GITLAB_TOKEN_VALUE}" ]] || die "${GITLAB_TOKEN_FILE} is empty."
[[ -n "${JIRA_TOKEN_VALUE}" ]]   || die "${JIRA_TOKEN_FILE} is empty."

green "   All three secret files found."

# ── gather Jira config ───────────────────────────────────────────────────────

if [[ -z "${JIRA_URL:-}" ]]; then
    printf '\n\033[1m   Jira instance URL\033[0m (e.g. https://myco.atlassian.net): '
    read -r JIRA_URL
    [[ -n "${JIRA_URL}" ]] || die "JIRA_URL is required for the jira credential."
fi

if [[ -z "${JIRA_EMAIL:-}" ]]; then
    printf '\033[1m   Jira account email\033[0m: '
    read -r JIRA_EMAIL
    [[ -n "${JIRA_EMAIL}" ]] || die "JIRA_EMAIL is required for the jira credential."
fi

# ── generate names ───────────────────────────────────────────────────────────

RUN_ID=$(date +%s | tail -c6)
PROJECT_NAME="demo-creds-${RUN_ID}"
AGENT_NAME="credential-verifier"

CRED_GITHUB="github-pat-${RUN_ID}"
CRED_GITLAB="gitlab-pat-${RUN_ID}"
CRED_JIRA="jira-token-${RUN_ID}"

echo
dim "   Run ID:     ${RUN_ID}"
dim "   Project:    ${PROJECT_NAME}"
dim "   Agent:      ${AGENT_NAME}"
dim "   GitHub:     ${CRED_GITHUB}"
dim "   GitLab:     ${CRED_GITLAB}"
dim "   Jira:       ${CRED_JIRA}"

echo
bold "   Press Enter to begin..."
read -r

# ── cleanup trap ─────────────────────────────────────────────────────────────

CREATED_PROJECT=""
CREATED_SESSION_ID=""

cleanup() {
    if [[ -n "${NO_CLEANUP:-}" ]]; then
        echo
        yellow "   NO_CLEANUP set — skipping cleanup"
        dim    "   project:    ${CREATED_PROJECT}"
        dim    "   session:    ${CREATED_SESSION_ID}"
        dim    "   credentials: ${CRED_GITHUB}, ${CRED_GITLAB}, ${CRED_JIRA}"
        return
    fi
    echo
    announce "Cleanup"
    if [[ -n "${CREATED_SESSION_ID}" ]]; then
        dim "   stopping session ${CREATED_SESSION_ID}..."
        "$ACPCTL" stop "${CREATED_SESSION_ID}" 2>/dev/null || true
        "$ACPCTL" delete session "${CREATED_SESSION_ID}" -y 2>/dev/null || true
    fi
    for cred in "${CRED_GITHUB}" "${CRED_GITLAB}" "${CRED_JIRA}"; do
        dim "   deleting credential ${cred}..."
        "$ACPCTL" credential delete "${cred}" --confirm 2>/dev/null || true
    done
    if [[ -n "${CREATED_PROJECT}" ]]; then
        dim "   deleting project ${CREATED_PROJECT}..."
        "$ACPCTL" delete project "${CREATED_PROJECT}" -y 2>/dev/null || true
    fi
    green "   cleanup done"
}
trap cleanup EXIT

# ── 1: verify login ─────────────────────────────────────────────────────────

announce "1 · Verify login"

step "Show authenticated user" \
    "$ACPCTL" whoami

# ── 2: create credentials ───────────────────────────────────────────────────

announce "2 · Create credentials"

step "Create GitHub credential: ${CRED_GITHUB}" \
    "$ACPCTL" credential create \
        --name "${CRED_GITHUB}" \
        --provider github \
        --token "${GITHUB_TOKEN_VALUE}" \
        --description "GitHub PAT for credential demo"

step "Create GitLab credential: ${CRED_GITLAB}" \
    "$ACPCTL" credential create \
        --name "${CRED_GITLAB}" \
        --provider gitlab \
        --token "${GITLAB_TOKEN_VALUE}" \
        --url "${GITLAB_URL}" \
        --description "GitLab PAT for credential demo"

step "Create Jira credential: ${CRED_JIRA}" \
    "$ACPCTL" credential create \
        --name "${CRED_JIRA}" \
        --provider jira \
        --token "${JIRA_TOKEN_VALUE}" \
        --url "${JIRA_URL}" \
        --email "${JIRA_EMAIL}" \
        --description "Jira API token for credential demo"

step "List all credentials" \
    "$ACPCTL" credential list

# ── 3: create project ───────────────────────────────────────────────────────

announce "3 · Create project"

step "Create project: ${PROJECT_NAME}" \
    "$ACPCTL" create project \
        --name "${PROJECT_NAME}" \
        --description "Credential lifecycle demo"

CREATED_PROJECT="${PROJECT_NAME}"

step "Set project context" \
    "$ACPCTL" project "${PROJECT_NAME}"

# ── 4: bind credentials to project ──────────────────────────────────────────

announce "4 · Bind credentials to project"

step "Bind GitHub credential" \
    "$ACPCTL" credential bind "${CRED_GITHUB}" --project "${PROJECT_NAME}"

step "Bind GitLab credential" \
    "$ACPCTL" credential bind "${CRED_GITLAB}" --project "${PROJECT_NAME}"

step "Bind Jira credential" \
    "$ACPCTL" credential bind "${CRED_JIRA}" --project "${PROJECT_NAME}"

# ── 5: create agent + start session ─────────────────────────────────────────

announce "5 · Create agent and start session"

sep; bold "▶  Create agent: ${AGENT_NAME}"; sleep "$PAUSE"
AGENT_JSON=$(
    "$ACPCTL" agent create \
        --project-id "${PROJECT_NAME}" \
        --name "${AGENT_NAME}" \
        --prompt "You are a credential verification agent. When asked, you confirm which credentials are available to you by listing their provider, name, and whether the token is present." \
        -o json 2>/dev/null
)
AGENT_ID=$(json_field "$AGENT_JSON" "id")
[[ -n "${AGENT_ID}" ]] || die "Failed to parse agent ID"
green "   agent created: ${AGENT_ID}"
echo

sep; bold "▶  Start session via agent"; sleep "$PAUSE"
printf '\033[38;5;214m   $ %s\033[0m\n' "acpctl start ${AGENT_ID} --project-id ${PROJECT_NAME}"
START_OUTPUT=$(
    "$ACPCTL" start "${AGENT_ID}" \
        --project-id "${PROJECT_NAME}" \
        --prompt "List all credentials available to you. For each one, report: provider, name, and whether the token is non-empty. Do NOT print the actual token value." \
        2>&1
)
echo "   ${START_OUTPUT}"

SESSION_ID=$(
    echo "${START_OUTPUT}" | sed -n 's|^session/\([^ ]*\) started.*|\1|p'
)
if [[ -z "${SESSION_ID}" ]]; then
    red "   Failed to parse session ID from start output"
    die "Expected output like: session/<id> started (phase: ...)"
fi
CREATED_SESSION_ID="${SESSION_ID}"
green "   session: ${SESSION_ID}"
echo

# ── wait for Running ─────────────────────────────────────────────────────────

announce "5b · Wait for session Running"

wait_for_running "${SESSION_ID}" || die "Session did not reach Running phase"

# ── 6: verify credentials in session ────────────────────────────────────────

announce "6 · Verify credentials in session"

sep
bold "▶  Send verification message and stream response"
printf '\033[38;5;214m   $ %s\033[0m\n' "acpctl session send ${SESSION_ID} \"...\" -f"
sleep "$PAUSE"

"$ACPCTL" session send "${SESSION_ID}" \
    "List every credential available to this session. For each credential, report: 1) provider name, 2) credential name, 3) whether a token value is present (yes/no). Do NOT reveal the actual token. Format as a simple table." \
    -f || yellow "   stream ended (may have timed out)"

echo
step "Session messages" \
    "$ACPCTL" session messages "${SESSION_ID}"

# ── done ─────────────────────────────────────────────────────────────────────

echo
sep
green "  Demo complete"
dim   "  Project ${PROJECT_NAME} and credentials will be deleted by cleanup."
sep
echo
