#!/usr/bin/env bash
set -euo pipefail

API_URL="${API_URL:-http://localhost:13592/api/ambient/v1}"
KC_URL="${KC_URL:-http://localhost:18592}"
KC_REALM="ambient-code"
KC_ADMIN_USER="admin"
KC_ADMIN_PASS="admin"
KC_CLIENT_ID="ambient-frontend"

PASS_COUNT=0
FAIL_COUNT=0
SKIP_COUNT=0

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
BOLD='\033[1m'
NC='\033[0m'

pass() { PASS_COUNT=$((PASS_COUNT + 1)); echo -e "  ${GREEN}[PASS]${NC} $1"; }
fail() { FAIL_COUNT=$((FAIL_COUNT + 1)); echo -e "  ${RED}[FAIL]${NC} $1: $2"; }
skip() { SKIP_COUNT=$((SKIP_COUNT + 1)); echo -e "  ${YELLOW}[SKIP]${NC} $1"; }

HTTP_STATUS=""
HTTP_BODY=""

api() {
  local method="$1" path="$2" token="$3" body="${4:-}"
  local args=(-s -w '\n%{http_code}' -H "Authorization: Bearer $token" -H "Content-Type: application/json")
  if [[ -n "$body" ]]; then
    args+=(-d "$body")
  fi
  local response
  response=$(curl "${args[@]}" -X "$method" "${API_URL}${path}")
  HTTP_STATUS=$(echo "$response" | tail -1)
  HTTP_BODY=$(echo "$response" | sed '$d')
}

assert_status() {
  local expected="$1" actual="$2" desc="$3"
  if [[ "$actual" == "$expected" ]]; then
    pass "$desc"
  else
    fail "$desc" "expected $expected, got $actual"
  fi
}

assert_list_contains() {
  local json="$1" field="$2" value="$3" desc="$4"
  if echo "$json" | jq -e ".items[]? | select(.${field} == \"${value}\")" >/dev/null 2>&1; then
    pass "$desc"
  else
    fail "$desc" "items missing ${field}=${value}"
  fi
}

assert_list_not_contains() {
  local json="$1" field="$2" value="$3" desc="$4"
  if echo "$json" | jq -e ".items[]? | select(.${field} == \"${value}\")" >/dev/null 2>&1; then
    fail "$desc" "items unexpectedly contain ${field}=${value}"
  else
    pass "$desc"
  fi
}

assert_list_count() {
  local json="$1" expected="$2" desc="$3"
  local actual
  actual=$(echo "$json" | jq '.items | length')
  if [[ "$actual" == "$expected" ]]; then
    pass "$desc"
  else
    fail "$desc" "expected $expected items, got $actual"
  fi
}

# --- Keycloak helpers ---

KC_ADMIN_TOKEN=""

get_admin_token() {
  KC_ADMIN_TOKEN=$(curl -s -X POST "${KC_URL}/realms/master/protocol/openid-connect/token" \
    -d "client_id=admin-cli" \
    -d "grant_type=password" \
    -d "username=${KC_ADMIN_USER}" \
    -d "password=${KC_ADMIN_PASS}" | jq -r '.access_token')
  if [[ -z "$KC_ADMIN_TOKEN" || "$KC_ADMIN_TOKEN" == "null" ]]; then
    echo "ERROR: Failed to get Keycloak admin token"
    exit 1
  fi
}

KC_CLIENT_SECRET=""

get_client_secret() {
  local clients
  clients=$(curl -s -H "Authorization: Bearer $KC_ADMIN_TOKEN" \
    "${KC_URL}/admin/realms/${KC_REALM}/clients?clientId=${KC_CLIENT_ID}")
  local client_uuid
  client_uuid=$(echo "$clients" | jq -r '.[0].id // empty')
  if [[ -z "$client_uuid" ]]; then
    echo "WARN: Could not find client ${KC_CLIENT_ID}, trying without secret"
    return
  fi
  local secret_resp
  secret_resp=$(curl -s -H "Authorization: Bearer $KC_ADMIN_TOKEN" \
    "${KC_URL}/admin/realms/${KC_REALM}/clients/${client_uuid}/client-secret")
  KC_CLIENT_SECRET=$(echo "$secret_resp" | jq -r '.value // empty')
}

create_keycloak_user() {
  local username="$1" password="$2" email="$3"
  local firstname="${4:-Test}" lastname="${5:-User}"
  curl -s -o /dev/null -X POST \
    -H "Authorization: Bearer $KC_ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    "${KC_URL}/admin/realms/${KC_REALM}/users" \
    -d "{\"username\":\"${username}\",\"email\":\"${email}\",\"firstName\":\"${firstname}\",\"lastName\":\"${lastname}\",\"emailVerified\":true,\"enabled\":true,\"requiredActions\":[],\"credentials\":[{\"type\":\"password\",\"value\":\"${password}\",\"temporary\":false}]}" 2>/dev/null || true
}

delete_keycloak_user() {
  local username="$1"
  local kc_uid
  kc_uid=$(curl -s -H "Authorization: Bearer $KC_ADMIN_TOKEN" \
    "${KC_URL}/admin/realms/${KC_REALM}/users?username=${username}&exact=true" | jq -r '.[0].id // empty')
  if [[ -n "$kc_uid" ]]; then
    curl -s -o /dev/null -X DELETE \
      -H "Authorization: Bearer $KC_ADMIN_TOKEN" \
      "${KC_URL}/admin/realms/${KC_REALM}/users/${kc_uid}"
  fi
}

get_token() {
  local username="$1" password="$2"
  local args=(-d "client_id=${KC_CLIENT_ID}" -d "grant_type=password" -d "username=${username}" -d "password=${password}" -d "scope=openid")
  if [[ -n "$KC_CLIENT_SECRET" ]]; then
    args+=(-d "client_secret=${KC_CLIENT_SECRET}")
  fi
  local resp
  resp=$(curl -s -X POST "${KC_URL}/realms/${KC_REALM}/protocol/openid-connect/token" "${args[@]}")
  local token
  token=$(echo "$resp" | jq -r '.access_token // empty')
  if [[ -z "$token" ]]; then
    echo "ERROR: Failed to get token for ${username}: $(echo "$resp" | jq -r '.error_description // .error // "unknown"')"
    exit 1
  fi
  echo "$token"
}

# --- Cleanup trap ---

CREATED_PROJECTS=()
CREATED_CRED_IDS=()

cleanup() {
  echo ""
  echo -e "${BOLD}Phase 7: Cleanup${NC}"

  get_admin_token

  # Delete test projects (need a token with access)
  for proj in "${CREATED_PROJECTS[@]:-}"; do
    if [[ -n "$proj" ]]; then
      # Use admin token or skip — projects may need owner token
      echo "  Cleaning project: $proj"
    fi
  done

  delete_keycloak_user "rbac-user-a"
  delete_keycloak_user "rbac-user-b"
  echo "  Keycloak users cleaned up"
}
trap cleanup EXIT

# ============================================================
echo -e "${BOLD}RBAC Enforcement E2E Tests${NC}"
echo "API: $API_URL"
echo "Keycloak: $KC_URL"
echo ""

# ============================================================
echo -e "${BOLD}Phase 1: Setup${NC}"

get_admin_token
get_client_secret

create_keycloak_user "rbac-user-a" "testpass" "rbac-a@test.dev" "Alice" "TestA"
create_keycloak_user "rbac-user-b" "testpass" "rbac-b@test.dev" "Bob" "TestB"
echo "  Created Keycloak users"

TOKEN_A=$(get_token "rbac-user-a" "testpass")
TOKEN_B=$(get_token "rbac-user-b" "testpass")
echo "  Got tokens for both users"

# Test 1: Auth-exempt endpoint works for both
api GET "/roles" "$TOKEN_A"
assert_status "200" "$HTTP_STATUS" "User A can GET /roles (auth-exempt)"

api GET "/roles" "$TOKEN_B"
assert_status "200" "$HTTP_STATUS" "User B can GET /roles (auth-exempt)"

# ============================================================
echo ""
echo -e "${BOLD}Phase 2: Project Isolation${NC}"

# Test 4: User A creates project
api POST "/projects" "$TOKEN_A" '{"name":"proj-alpha","description":"User A project"}'
assert_status "201" "$HTTP_STATUS" "User A creates proj-alpha"
CREATED_PROJECTS+=("proj-alpha")

# Test 5: User B creates project
api POST "/projects" "$TOKEN_B" '{"name":"proj-beta","description":"User B project"}'
assert_status "201" "$HTTP_STATUS" "User B creates proj-beta"
CREATED_PROJECTS+=("proj-beta")

# Test 6: User A lists projects — should see only proj-alpha
api GET "/projects?page=1&size=100" "$TOKEN_A"
assert_status "200" "$HTTP_STATUS" "User A GET /projects returns 200"
assert_list_contains "$HTTP_BODY" "name" "proj-alpha" "User A sees proj-alpha in list"
assert_list_not_contains "$HTTP_BODY" "name" "proj-beta" "User A does NOT see proj-beta in list"

# Test 7: User B lists projects — should see only proj-beta
api GET "/projects?page=1&size=100" "$TOKEN_B"
assert_status "200" "$HTTP_STATUS" "User B GET /projects returns 200"
assert_list_contains "$HTTP_BODY" "name" "proj-beta" "User B sees proj-beta in list"
assert_list_not_contains "$HTTP_BODY" "name" "proj-alpha" "User B does NOT see proj-alpha in list"

# Test 8: User A GETs proj-beta — should be 404 (not 403)
api GET "/projects/proj-beta" "$TOKEN_A"
assert_status "404" "$HTTP_STATUS" "User A GET proj-beta returns 404 (not 403)"

# Test 9: User B GETs proj-alpha — should be 404
api GET "/projects/proj-alpha" "$TOKEN_B"
assert_status "404" "$HTTP_STATUS" "User B GET proj-alpha returns 404 (not 403)"

# ============================================================
echo ""
echo -e "${BOLD}Phase 3: Agent & Session Isolation${NC}"

# Test 10: User A creates agent in proj-alpha
api POST "/projects/proj-alpha/agents" "$TOKEN_A" '{"name":"agent-a","prompt":"test agent a","project_id":"proj-alpha"}'
assert_status "201" "$HTTP_STATUS" "User A creates agent-a in proj-alpha"
AGENT_A_ID=$(echo "$HTTP_BODY" | jq -r '.id // empty')

# Test 11: User B creates agent in proj-beta
api POST "/projects/proj-beta/agents" "$TOKEN_B" '{"name":"agent-b","prompt":"test agent b","project_id":"proj-beta"}'
assert_status "201" "$HTTP_STATUS" "User B creates agent-b in proj-beta"

# Test 12: User A cannot list agents in proj-beta
api GET "/projects/proj-beta/agents?page=1&size=100" "$TOKEN_A"
assert_status "404" "$HTTP_STATUS" "User A GET proj-beta/agents returns 404"

# Test 13: User B cannot list agents in proj-alpha
api GET "/projects/proj-alpha/agents?page=1&size=100" "$TOKEN_B"
assert_status "404" "$HTTP_STATUS" "User B GET proj-alpha/agents returns 404"

# ============================================================
echo ""
echo -e "${BOLD}Phase 4: Credential Isolation${NC}"

# Test 14: User A creates credential
api POST "/credentials" "$TOKEN_A" '{"name":"cred-a","provider":"github","token":"test-fake-token-a"}'
assert_status "201" "$HTTP_STATUS" "User A creates cred-a"
CRED_A_ID=$(echo "$HTTP_BODY" | jq -r '.id // empty')
CREATED_CRED_IDS+=("$CRED_A_ID")

# Test 15: User B creates credential
api POST "/credentials" "$TOKEN_B" '{"name":"cred-b","provider":"github","token":"test-fake-token-b"}'
assert_status "201" "$HTTP_STATUS" "User B creates cred-b"
CRED_B_ID=$(echo "$HTTP_BODY" | jq -r '.id // empty')
CREATED_CRED_IDS+=("$CRED_B_ID")

# Test 16: User A lists credentials — only cred-a
api GET "/credentials?page=1&size=100" "$TOKEN_A"
assert_status "200" "$HTTP_STATUS" "User A GET /credentials returns 200"
assert_list_contains "$HTTP_BODY" "name" "cred-a" "User A sees cred-a"
assert_list_not_contains "$HTTP_BODY" "name" "cred-b" "User A does NOT see cred-b"

# Test 17: User B lists credentials — only cred-b
api GET "/credentials?page=1&size=100" "$TOKEN_B"
assert_status "200" "$HTTP_STATUS" "User B GET /credentials returns 200"
assert_list_contains "$HTTP_BODY" "name" "cred-b" "User B sees cred-b"
assert_list_not_contains "$HTTP_BODY" "name" "cred-a" "User B does NOT see cred-a"

# Test 18: User A GETs cred-b — should be 404
api GET "/credentials/${CRED_B_ID}" "$TOKEN_A"
assert_status "404" "$HTTP_STATUS" "User A GET cred-b returns 404"

# ============================================================
echo ""
echo -e "${BOLD}Phase 5: Sharing via RoleBindings${NC}"

# Test 19: Look up project:editor role ID
api GET "/roles" "$TOKEN_A"
EDITOR_ROLE_ID=$(echo "$HTTP_BODY" | jq -r '.items[] | select(.name == "project:editor") | .id')
if [[ -z "$EDITOR_ROLE_ID" ]]; then
  fail "Look up project:editor role" "role not found"
else
  pass "Found project:editor role ID"
fi

# Test 20: User A grants User B project:editor on proj-alpha
api POST "/role_bindings" "$TOKEN_A" "{\"role_id\":\"${EDITOR_ROLE_ID}\",\"scope\":\"project\",\"user_id\":\"rbac-user-b\",\"project_id\":\"proj-alpha\"}"
assert_status "201" "$HTTP_STATUS" "User A grants User B project:editor on proj-alpha"

# Test 21: User B now sees both projects
api GET "/projects?page=1&size=100" "$TOKEN_B"
assert_status "200" "$HTTP_STATUS" "User B GET /projects after sharing returns 200"
assert_list_contains "$HTTP_BODY" "name" "proj-alpha" "User B now sees proj-alpha"
assert_list_contains "$HTTP_BODY" "name" "proj-beta" "User B still sees proj-beta"

# Test 22: User B can GET proj-alpha
api GET "/projects/proj-alpha" "$TOKEN_B"
assert_status "200" "$HTTP_STATUS" "User B GET proj-alpha returns 200 after sharing"

# Test 23: User B can create agent in proj-alpha
api POST "/projects/proj-alpha/agents" "$TOKEN_B" '{"name":"agent-shared","prompt":"shared agent","project_id":"proj-alpha"}'
assert_status "201" "$HTTP_STATUS" "User B creates agent in proj-alpha (shared)"

# ============================================================
echo ""
echo -e "${BOLD}Phase 6: Escalation Prevention${NC}"

skip "User B (editor) grants project:owner — not yet wired"
skip "User A (owner) grants project:owner to B — not yet wired"

# ============================================================
echo ""
echo -e "${BOLD}Summary${NC}"
echo -e "  ${GREEN}${PASS_COUNT} passed${NC}, ${RED}${FAIL_COUNT} failed${NC}, ${YELLOW}${SKIP_COUNT} skipped${NC}"

if [[ "$FAIL_COUNT" -gt 0 ]]; then
  exit 1
fi
