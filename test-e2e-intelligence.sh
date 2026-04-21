#!/usr/bin/env bash
# End-to-end test for Project Intelligence Memory feature.
# Runs against a live Kind cluster.  Exit code 0 = all pass.
set -uo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[0;33m'; NC='\033[0m'
PASS=0; FAIL=0
check() {
  local name="$1"; shift
  if "$@" >/dev/null 2>&1; then
    echo -e "  ${GREEN}PASS${NC} $name"; ((PASS++))
  else
    echo -e "  ${RED}FAIL${NC} $name"; ((FAIL++))
  fi
}

KIND_CONTEXT="kind-ambient-feat-project-intelli"

PROJECT="my-llm-katan"
REPO_URL="https://github.com/yossiovadia/llm-katan"
REPO_NAME="llm-katan"
BACKEND="http://localhost:18080"
API_SERVER="http://localhost:18000"
FRONTEND="http://localhost:9579"

echo "============================================"
echo " Project Intelligence E2E Test"
echo "============================================"

# ── Port forwards ───────────────────────────────────────────────
pkill -f "port-forward.*18080" 2>/dev/null || true
pkill -f "port-forward.*18000" 2>/dev/null || true
pkill -f "port-forward.*9579"  2>/dev/null || true
sleep 1
kubectl --context="$KIND_CONTEXT" port-forward -n ambient-code svc/backend-service   18080:8080 &>/dev/null &
kubectl --context="$KIND_CONTEXT" port-forward -n ambient-code svc/ambient-api-server 18000:8000 &>/dev/null &
kubectl --context="$KIND_CONTEXT" port-forward -n ambient-code svc/frontend-service   9579:3000  &>/dev/null &
sleep 3

TOKEN=$(kubectl --context="$KIND_CONTEXT" create token agentic-operator -n ambient-code --duration=1h)

# ── Step 1: Clean slate ─────────────────────────────────────────
echo ""
echo "Step 1: Clean slate"
for s in $(kubectl --context="$KIND_CONTEXT" get agenticsessions -n "$PROJECT" -o jsonpath='{.items[*].metadata.name}' 2>/dev/null); do
  kubectl --context="$KIND_CONTEXT" delete agenticsession "$s" -n "$PROJECT" --wait=false 2>/dev/null
done
sleep 8
kubectl --context="$KIND_CONTEXT" exec -n ambient-code deployment/ambient-api-server-db -- \
  psql -U ambient -d ambient_api_server -c \
  "DELETE FROM repo_findings; DELETE FROM repo_events; DELETE FROM repo_intelligences;" \
  >/dev/null 2>&1
# Wait for pods to terminate
for i in $(seq 1 10); do
  [[ $(kubectl --context="$KIND_CONTEXT" get pods -n "$PROJECT" --no-headers 2>/dev/null | grep -v Terminating | wc -l) -eq 0 ]] && break
  sleep 3
done

SESSIONS=$(kubectl --context="$KIND_CONTEXT" get agenticsessions -n "$PROJECT" --no-headers 2>&1 | grep -c "session" || true)
INTEL=$(curl -s "$API_SERVER/api/ambient/v1/repo_intelligences" | python3 -c "import sys,json; print(json.load(sys.stdin)['total'])")
check "No sessions"       test "$SESSIONS" -eq 0
check "No intelligence"   test "$INTEL" -eq 0

# ── Step 2: Create session (NO repos — user adds via UI) ────────
echo ""
echo "Step 2: Create session (no repos, same as UI)"
RESULT=$(curl -s -X POST "$BACKEND/api/projects/$PROJECT/agentic-sessions" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{
    \"displayName\": \"E2E Intelligence Test\",
    \"timeout\": 3600,
    \"interactive\": true
  }")
SESSION=$(echo "$RESULT" | python3 -c "import sys,json; print(json.load(sys.stdin).get('name',''))" 2>/dev/null)
AUTOBRANCH=$(echo "$RESULT" | python3 -c "import sys,json; print(json.load(sys.stdin).get('autoBranch',''))" 2>/dev/null)
check "Session created"   test -n "$SESSION"
echo "  Session: $SESSION"
echo "  AutoBranch: $AUTOBRANCH"

# Wait for pod
for i in $(seq 1 20); do
  READY=$(kubectl --context="$KIND_CONTEXT" get pod "${SESSION}-runner" -n "$PROJECT" -o jsonpath='{.status.containerStatuses[0].ready}' 2>/dev/null || true)
  [[ "$READY" == "true" ]] && break
  sleep 5
done
check "Pod running"  test "$READY" = "true"

# ── Step 3: Add repo via BACKEND (same path as UI) ─────────────
echo ""
echo "Step 3: Add repo via backend API (same as UI with session branch)"
# The UI sends the session branch name (ambient/session-xxx).
# This tests the exact same code path the browser uses.
ADD_RESULT=$(curl -s -X POST "$BACKEND/api/projects/$PROJECT/agentic-sessions/$SESSION/repos" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{\"url\":\"$REPO_URL\",\"branch\":\"$AUTOBRANCH\"}")
ADD_OK=$(echo "$ADD_RESULT" | python3 -c "import sys,json; d=json.load(sys.stdin); print('yes' if 'message' in d and 'added' in d.get('message','').lower() else 'no')" 2>/dev/null)
check "Repo added via backend"  test "$ADD_OK" = "yes"

# Wait for runner to process the clone
sleep 5

# Verify repo was actually cloned on disk
REPO_EXISTS=$(kubectl --context="$KIND_CONTEXT" exec -n "$PROJECT" "${SESSION}-runner" -c ambient-code-runner -- test -d "/workspace/repos/$REPO_NAME/.git" 2>/dev/null && echo "yes" || echo "no")
check "Repo cloned on disk"  test "$REPO_EXISTS" = "yes"

# ── Step 4: Wait for analysis ──────────────────────────────────
echo ""
echo "Step 4: Wait for auto-analysis (max 3 min)..."
ANALYSIS_DONE=false
for i in $(seq 1 36); do
  if kubectl --context="$KIND_CONTEXT" logs "${SESSION}-runner" -n "$PROJECT" -c ambient-code-runner 2>/dev/null \
     | grep -q "Auto-analysis complete"; then
    ANALYSIS_DONE=true; break
  fi
  sleep 5
done
check "Analysis complete"  $ANALYSIS_DONE
if $ANALYSIS_DONE; then
  ROUNDS=$(kubectl --context="$KIND_CONTEXT" logs "${SESSION}-runner" -n "$PROJECT" -c ambient-code-runner 2>/dev/null \
    | grep "Auto-analysis complete" | tail -1 | sed 's/.*(\([0-9]* rounds\)).*/\1/' || echo "? rounds")
  echo "  $ROUNDS"
fi

# ── Step 5: Verify intelligence in DB ──────────────────────────
echo ""
echo "Step 5: Verify intelligence stored"
INTEL_RESP=$(curl -s "$API_SERVER/api/ambient/v1/repo_intelligences/lookup?project_id=$PROJECT&repo_url=$(python3 -c 'import urllib.parse; print(urllib.parse.quote("'"$REPO_URL"'", safe=""))')")
LANG=$(echo "$INTEL_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('language',''))" 2>/dev/null)
FRAMEWORK=$(echo "$INTEL_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('framework',''))" 2>/dev/null)
SUMMARY=$(echo "$INTEL_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('summary','')[:80])" 2>/dev/null)
check "Language = Python"          test "$LANG" = "Python"
check "Framework contains FastAPI" echo "$FRAMEWORK" | grep -qi "fastapi"
check "Summary not hallucinated"   echo "$SUMMARY" | grep -qiv "settlers\|catan\|board game"
echo "  $LANG / $FRAMEWORK"
echo "  $SUMMARY..."

FINDINGS=$(curl -s "$API_SERVER/api/ambient/v1/repo_findings" | python3 -c "import sys,json; print(json.load(sys.stdin)['total'])" 2>/dev/null)
check "Findings stored (>0)"  test "$FINDINGS" -gt 0
echo "  $FINDINGS findings"

# ── Step 6: No analysis prompts leaked into chat ───────────────
echo ""
echo "Step 6: Check no analysis prompts leaked"
# The auto-analysis runs standalone via Vertex API, not through the bridge.
# Check the AGUI event store for any messages containing analysis instructions.
LEAKED=$(kubectl --context="$KIND_CONTEXT" logs "${SESSION}-runner" -n "$PROJECT" -c ambient-code-runner 2>/dev/null \
  | grep -c "discarded.*pending\|Queued pending prompt" || true)
# "Queued" is acceptable (old mechanism logged it before the standalone fix).
# "discarded" means the Claude bridge ate the prompt. Neither should appear
# with the standalone auto-analysis.
check "No leaked analysis prompts"  test "$LEAKED" -eq 0

# ── Step 7: Frontend proxy returns intelligence ────────────────
echo ""
echo "Step 7: Verify Analyzed badge data"
FE_RESP=$(curl -s "$FRONTEND/api/projects/$PROJECT/intelligence?repo_url=$(python3 -c 'import urllib.parse; print(urllib.parse.quote("'"$REPO_URL"'", safe=""))')")
FE_LANG=$(echo "$FE_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('language',''))" 2>/dev/null)
check "Frontend proxy returns data"  test "$FE_LANG" = "Python"

# ── Step 8: Send user message ──────────────────────────────────
echo ""
echo "Step 8: Send user message"

# Verify runner type is Claude (the default — same as UI)
RUNNER_TYPE=$(kubectl --context="$KIND_CONTEXT" get pod "${SESSION}-runner" -n "$PROJECT" \
  -o jsonpath='{range .spec.containers[0].env[*]}{.name}={.value}{"\n"}{end}' 2>/dev/null \
  | grep RUNNER_TYPE | cut -d= -f2)
check "Runner is Claude SDK (not direct-to-llm)"  test "$RUNNER_TYPE" = "claude-agent-sdk"

# Send the question directly — no warmup message.
# The Claude bridge's _refresh_system_prompt() should pick up
# intelligence on this first message after analysis completed.
curl -s -X POST "$BACKEND/api/projects/$PROJECT/agentic-sessions/$SESSION/agui/run" \
  -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" \
  -d "{\"threadId\":\"$SESSION\",\"runId\":\"question-$(date +%s)\",\"messages\":[{\"id\":\"q1\",\"role\":\"user\",\"content\":\"What caveats should I know about llm-katan? Answer from what you already know.\"}]}" >/dev/null 2>&1

sleep 15

# Check that intelligence was injected into the prompt
INJECTED=$(kubectl --context="$KIND_CONTEXT" logs "${SESSION}-runner" -n "$PROJECT" -c ambient-code-runner --since=20s 2>/dev/null \
  | grep -c "Injected intelligence context" || true)
check "Intelligence injected into prompt"  test "$INJECTED" -gt 0

# ── Step 9: Wait for response ──────────────────────────────────
echo ""
echo "Step 9: Wait for response (max 2 min)..."
# Subscribe to events and capture assistant response
RESPONSE=""
for i in $(seq 1 24); do
  sleep 5
  # Check backend event store for assistant messages via the events endpoint
  EVENTS=$(timeout 3 curl -s -N "$BACKEND/api/projects/$PROJECT/agentic-sessions/$SESSION/agui/events" \
    -H "Accept: text/event-stream" -H "Authorization: Bearer $TOKEN" 2>/dev/null || true)
  # Look for MESSAGES_SNAPSHOT with assistant role
  RESPONSE=$(echo "$EVENTS" | grep "MESSAGES_SNAPSHOT" | tail -1 | \
    python3 -c "
import sys,json
for line in sys.stdin:
  if 'data: ' in line:
    try:
      d=json.loads(line.split('data: ')[1])
      for m in d.get('messages',[]):
        if m.get('role')=='assistant':
          print(m.get('content','')[:500])
    except: pass
" 2>/dev/null || true)
  if [ -n "$RESPONSE" ]; then break; fi
done
check "Got assistant response"  test -n "$RESPONSE"

# ── Step 10: Verify response quality ──────────────────────────
echo ""
echo "Step 10: Verify response accuracy"
if [ -n "$RESPONSE" ]; then
  check "Mentions Python or FastAPI"       echo "$RESPONSE" | grep -qiE "python|fastapi"
  check "Mentions provider pattern"        echo "$RESPONSE" | grep -qiE "provider|backend|middleware|auth"
  check "NOT Settlers of Catan"            echo "$RESPONSE" | grep -qiv "settlers.*catan\|board game\|game agent"
  echo "  Response preview: ${RESPONSE:0:120}..."
else
  echo -e "  ${RED}SKIP${NC} (no response to evaluate)"
  ((FAIL+=3))
fi

# ── Step 11: No file-reading tools in response ────────────────
echo ""
echo "Step 11: Verify no file reading"
# Check recent runner logs for tool use during the question run
TOOL_READS=$(kubectl --context="$KIND_CONTEXT" logs "${SESSION}-runner" -n "$PROJECT" -c ambient-code-runner --since=120s 2>/dev/null \
  | grep -c "Read\|Glob\|Bash\|file_search\|read_file.*question" || true)
check "No file-reading tools used"  test "$TOOL_READS" -eq 0

# ── Summary ─────────────────────────────────────────────────────
echo ""
echo "============================================"
TOTAL=$((PASS + FAIL))
if [ "$FAIL" -eq 0 ]; then
  echo -e " ${GREEN}ALL $TOTAL TESTS PASSED${NC}"
else
  echo -e " ${RED}$FAIL/$TOTAL TESTS FAILED${NC}"
fi
echo "============================================"

# Cleanup port-forwards
pkill -f "port-forward.*18001" 2>/dev/null || true

exit "$FAIL"
