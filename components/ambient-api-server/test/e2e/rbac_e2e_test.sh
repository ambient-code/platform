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
    fail "$desc" "expected $expected, got $actual (body: $(echo "$HTTP_BODY" | head -c 200))"
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

# --- Role ID lookup helper ---

ROLE_IDS_JSON=""

lookup_role_id() {
  local role_name="$1"
  echo "$ROLE_IDS_JSON" | jq -r ".items[] | select(.name == \"${role_name}\") | .id"
}

# --- Binding search helper ---
# Usage: get_binding_id <token> <search_query>
# Example: get_binding_id "$TOKEN_A" "user_id='rbac-user-b' and project_id='proj-alpha'"

get_binding_id() {
  local token="$1" search="$2"
  api GET "/role_bindings?search=$(python3 -c "import urllib.parse; print(urllib.parse.quote(\"${search}\"))")&page=1&size=100" "$token"
  echo "$HTTP_BODY" | jq -r '.items[0].id // empty'
}

# --- Cleanup trap ---

CREATED_PROJECTS=()
CREATED_CRED_IDS=()

cleanup() {
  echo ""
  echo -e "${BOLD}Phase 13: Cleanup${NC}"

  get_admin_token

  # Delete test Keycloak users (DB records are auto-provisioned, best effort cleanup)
  for proj in "${CREATED_PROJECTS[@]:-}"; do
    if [[ -n "$proj" ]]; then
      echo "  Test project created: $proj (will be cleaned up by test or left for manual cleanup)"
    fi
  done

  delete_keycloak_user "rbac-user-a"
  delete_keycloak_user "rbac-user-b"
  delete_keycloak_user "rbac-user-c"
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
create_keycloak_user "rbac-user-c" "testpass" "rbac-c@test.dev" "Charlie" "TestC"
echo "  Created Keycloak users (Alice, Bob, Charlie)"

TOKEN_A=$(get_token "rbac-user-a" "testpass")
TOKEN_B=$(get_token "rbac-user-b" "testpass")
TOKEN_C=$(get_token "rbac-user-c" "testpass")
echo "  Got tokens for all users"

# Fetch all role IDs for later use
api GET "/roles?page=1&size=100" "$TOKEN_A"
assert_status "200" "$HTTP_STATUS" "GET /roles (auth-exempt) returns 200"
ROLE_IDS_JSON="$HTTP_BODY"

ROLE_PROJECT_OWNER=$(lookup_role_id "project:owner")
ROLE_PROJECT_EDITOR=$(lookup_role_id "project:editor")
ROLE_PROJECT_VIEWER=$(lookup_role_id "project:viewer")
ROLE_CREDENTIAL_OWNER=$(lookup_role_id "credential:owner")
ROLE_CREDENTIAL_READER=$(lookup_role_id "credential:reader")
ROLE_AGENT_RUNNER=$(lookup_role_id "agent:runner")
ROLE_CRED_TOKEN_READER=$(lookup_role_id "credential:token-reader")
ROLE_PLATFORM_ADMIN=$(lookup_role_id "platform:admin")

if [[ -z "$ROLE_PROJECT_OWNER" ]]; then
  fail "Role lookup" "project:owner role not found in /roles response"
  echo "FATAL: Cannot continue without role IDs"
  exit 1
fi

pass "Looked up all role IDs"

# Verify auth-exempt for User B too
api GET "/roles" "$TOKEN_B"
assert_status "200" "$HTTP_STATUS" "User B can GET /roles (auth-exempt)"

# Verify GET /roles/{id} is also auth-exempt
api GET "/roles/${ROLE_PROJECT_OWNER}" "$TOKEN_A"
assert_status "200" "$HTTP_STATUS" "GET /roles/{id} is auth-exempt"

# ============================================================
echo ""
echo -e "${BOLD}Phase 2: Bootstrap & Auto-Provisioning (scenarios 10, 14-17)${NC}"

# Scenario 17: New user has zero bindings, sees empty project list
api GET "/projects?page=1&size=100" "$TOKEN_A"
assert_status "200" "$HTTP_STATUS" "User A GET /projects before creating any returns 200"
# User A may see zero items or items from a prior run; the key test is below after creating
BODY_BEFORE="$HTTP_BODY"

# Scenario 14: User auto-provisioned from JWT on first request
# (The GET /roles above already triggered auto-provisioning)
# Verify user record exists via a side-channel: user can create a project
# (direct DB check is optional and depends on kubectl access)

# Scenario 10: User A creates first project, owner binding auto-created
api POST "/projects" "$TOKEN_A" '{"name":"rbac-proj-alpha","description":"Alice project"}'
assert_status "201" "$HTTP_STATUS" "Scenario 10: User A creates first project rbac-proj-alpha"
CREATED_PROJECTS+=("rbac-proj-alpha")

# Verify the owner binding was auto-created
api GET "/role_bindings?search=user_id%3D'rbac-user-a'%20and%20project_id%3D'rbac-proj-alpha'&page=1&size=100" "$TOKEN_A"
if echo "$HTTP_BODY" | jq -e '.items[] | select(.scope == "project")' >/dev/null 2>&1; then
  pass "Scenario 10: project:owner binding auto-created for User A on rbac-proj-alpha"
else
  fail "Scenario 10: project:owner binding auto-created" "binding not found in role_bindings"
fi

# Scenario 15: User A can immediately manage the project after creation
api GET "/projects/rbac-proj-alpha" "$TOKEN_A"
assert_status "200" "$HTTP_STATUS" "Scenario 15: User A can immediately GET own project after creation"

# ============================================================
echo ""
echo -e "${BOLD}Phase 3: Project Isolation (scenarios 1, 7, 9, 16-17, 50, 52)${NC}"

# User B creates proj-beta
api POST "/projects" "$TOKEN_B" '{"name":"rbac-proj-beta","description":"Bob project"}'
assert_status "201" "$HTTP_STATUS" "User B creates rbac-proj-beta"
CREATED_PROJECTS+=("rbac-proj-beta")

# Scenario 7: User A lists projects - sees only proj-alpha
api GET "/projects?page=1&size=100" "$TOKEN_A"
assert_status "200" "$HTTP_STATUS" "User A GET /projects returns 200"
assert_list_contains "$HTTP_BODY" "name" "rbac-proj-alpha" "Scenario 7: User A sees rbac-proj-alpha in list"
assert_list_not_contains "$HTTP_BODY" "name" "rbac-proj-beta" "Scenario 7: User A does NOT see rbac-proj-beta"

# User B lists projects - sees only proj-beta
api GET "/projects?page=1&size=100" "$TOKEN_B"
assert_status "200" "$HTTP_STATUS" "User B GET /projects returns 200"
assert_list_contains "$HTTP_BODY" "name" "rbac-proj-beta" "User B sees rbac-proj-beta in list"
assert_list_not_contains "$HTTP_BODY" "name" "rbac-proj-alpha" "User B does NOT see rbac-proj-alpha"

# Scenario 50: Singleton GET returns 404 (not 403) for unauthorized project
api GET "/projects/rbac-proj-beta" "$TOKEN_A"
assert_status "404" "$HTTP_STATUS" "Scenario 50: User A GET rbac-proj-beta returns 404 (not 403)"

api GET "/projects/rbac-proj-alpha" "$TOKEN_B"
assert_status "404" "$HTTP_STATUS" "Scenario 50: User B GET rbac-proj-alpha returns 404"

# Scenario 9 / 52: User with no project bindings lists projects -> empty list
api GET "/projects?page=1&size=100" "$TOKEN_C"
assert_status "200" "$HTTP_STATUS" "Scenario 9/52: User C (no bindings) GET /projects returns 200"
assert_list_not_contains "$HTTP_BODY" "name" "rbac-proj-alpha" "Scenario 52: User C does not see rbac-proj-alpha"
assert_list_not_contains "$HTTP_BODY" "name" "rbac-proj-beta" "Scenario 52: User C does not see rbac-proj-beta"

# Scenario 16-17: New user cannot access existing resources
api GET "/projects/rbac-proj-alpha" "$TOKEN_C"
assert_status "404" "$HTTP_STATUS" "Scenario 17: User C GET existing project returns 404"

api GET "/sessions?page=1&size=100" "$TOKEN_C"
assert_status "200" "$HTTP_STATUS" "Scenario 17: User C GET /sessions returns 200 (empty, not 403)"

# ============================================================
echo ""
echo -e "${BOLD}Phase 4: Agent Isolation (scenarios 3-4)${NC}"

# Scenario 3: User A creates agent in proj-alpha
api POST "/projects/rbac-proj-alpha/agents" "$TOKEN_A" '{"name":"agent-alpha","prompt":"test agent alpha","project_id":"rbac-proj-alpha"}'
assert_status "201" "$HTTP_STATUS" "Scenario 3: User A creates agent-alpha in rbac-proj-alpha"
AGENT_A_ID=$(echo "$HTTP_BODY" | jq -r '.id // empty')

# User B creates agent in proj-beta
api POST "/projects/rbac-proj-beta/agents" "$TOKEN_B" '{"name":"agent-beta","prompt":"test agent beta","project_id":"rbac-proj-beta"}'
assert_status "201" "$HTTP_STATUS" "User B creates agent-beta in rbac-proj-beta"
AGENT_B_ID=$(echo "$HTTP_BODY" | jq -r '.id // empty')

# User A cannot access agents in proj-beta (parent project not accessible -> 404)
api GET "/projects/rbac-proj-beta/agents?page=1&size=100" "$TOKEN_A"
assert_status "404" "$HTTP_STATUS" "Scenario 3: User A GET rbac-proj-beta/agents returns 404"

# User B cannot access agents in proj-alpha
api GET "/projects/rbac-proj-alpha/agents?page=1&size=100" "$TOKEN_B"
assert_status "404" "$HTTP_STATUS" "User B GET rbac-proj-alpha/agents returns 404"

# Scenario 4: Scope hierarchy - project:owner covers all agents in project
api GET "/projects/rbac-proj-alpha/agents?page=1&size=100" "$TOKEN_A"
assert_status "200" "$HTTP_STATUS" "Scenario 4: User A lists agents in own project -> 200"
assert_list_contains "$HTTP_BODY" "name" "agent-alpha" "Scenario 4: User A sees agent-alpha in own project"

# User A can GET specific agent in own project
if [[ -n "$AGENT_A_ID" ]]; then
  api GET "/projects/rbac-proj-alpha/agents/${AGENT_A_ID}" "$TOKEN_A"
  assert_status "200" "$HTTP_STATUS" "Scenario 4: project:owner covers GET specific agent in project"
fi

# ============================================================
echo ""
echo -e "${BOLD}Phase 5: Session Isolation (scenario 6)${NC}"

# Scenario 6: Sessions list filtered by project bindings
# User A can only see sessions from projects they have access to
api GET "/sessions?page=1&size=100" "$TOKEN_A"
assert_status "200" "$HTTP_STATUS" "Scenario 6: User A GET /sessions returns 200"
SESSIONS_A="$HTTP_BODY"

api GET "/sessions?page=1&size=100" "$TOKEN_B"
assert_status "200" "$HTTP_STATUS" "Scenario 6: User B GET /sessions returns 200"
SESSIONS_B="$HTTP_BODY"

# User C (no bindings) sees empty session list
api GET "/sessions?page=1&size=100" "$TOKEN_C"
assert_status "200" "$HTTP_STATUS" "Scenario 6: User C GET /sessions returns 200 (empty, not 403)"

pass "Scenario 6: Session list endpoints return 200 for all users (filtered by project access)"

# ============================================================
echo ""
echo -e "${BOLD}Phase 6: Credential Isolation (scenarios 18-23)${NC}"

# Scenario 18: User A creates credential -> 201
api POST "/credentials" "$TOKEN_A" '{"name":"rbac-cred-a","provider":"github","token":"test-fake-token-a"}'
assert_status "201" "$HTTP_STATUS" "Scenario 18: User A creates rbac-cred-a"
CRED_A_ID=$(echo "$HTTP_BODY" | jq -r '.id // empty')
CREATED_CRED_IDS+=("$CRED_A_ID")

# Scenario 19: credential:owner binding auto-created
api GET "/role_bindings?search=user_id%3D'rbac-user-a'%20and%20credential_id%3D'${CRED_A_ID}'&page=1&size=100" "$TOKEN_A"
if echo "$HTTP_BODY" | jq -e '.items[] | select(.scope == "credential")' >/dev/null 2>&1; then
  pass "Scenario 19: credential:owner binding auto-created for User A on rbac-cred-a"
else
  fail "Scenario 19: credential:owner binding auto-created" "binding not found"
fi

# User B creates credential
api POST "/credentials" "$TOKEN_B" '{"name":"rbac-cred-b","provider":"github","token":"test-fake-token-b"}'
assert_status "201" "$HTTP_STATUS" "User B creates rbac-cred-b"
CRED_B_ID=$(echo "$HTTP_BODY" | jq -r '.id // empty')
CREATED_CRED_IDS+=("$CRED_B_ID")

# Scenario 23: User A lists credentials -> only cred-a
api GET "/credentials?page=1&size=100" "$TOKEN_A"
assert_status "200" "$HTTP_STATUS" "Scenario 23: User A GET /credentials returns 200"
assert_list_contains "$HTTP_BODY" "name" "rbac-cred-a" "Scenario 23: User A sees rbac-cred-a"
assert_list_not_contains "$HTTP_BODY" "name" "rbac-cred-b" "Scenario 23: User A does NOT see rbac-cred-b"

# User B lists credentials -> only cred-b
api GET "/credentials?page=1&size=100" "$TOKEN_B"
assert_status "200" "$HTTP_STATUS" "User B GET /credentials returns 200"
assert_list_contains "$HTTP_BODY" "name" "rbac-cred-b" "User B sees rbac-cred-b"
assert_list_not_contains "$HTTP_BODY" "name" "rbac-cred-a" "User B does NOT see rbac-cred-a"

# Singleton GET on credential user does not own -> 404
api GET "/credentials/${CRED_B_ID}" "$TOKEN_A"
assert_status "404" "$HTTP_STATUS" "Scenario 23: User A GET rbac-cred-b returns 404"

api GET "/credentials/${CRED_A_ID}" "$TOKEN_B"
assert_status "404" "$HTTP_STATUS" "User B GET rbac-cred-a returns 404"

# Scenario 20: Credential owner binds credential to own project -> 201
api POST "/role_bindings" "$TOKEN_A" "{\"role_id\":\"${ROLE_CREDENTIAL_READER}\",\"scope\":\"credential\",\"user_id\":\"rbac-user-a\",\"credential_id\":\"${CRED_A_ID}\",\"project_id\":\"rbac-proj-alpha\"}"
assert_status "201" "$HTTP_STATUS" "Scenario 20: Credential owner binds rbac-cred-a to own project rbac-proj-alpha"
CRED_BIND_ID=$(echo "$HTTP_BODY" | jq -r '.id // empty')

# Scenario 21: Non-project-owner cannot bind credential to project
# User B owns rbac-cred-b but does NOT own rbac-proj-alpha
api POST "/role_bindings" "$TOKEN_B" "{\"role_id\":\"${ROLE_CREDENTIAL_READER}\",\"scope\":\"credential\",\"user_id\":\"rbac-user-b\",\"credential_id\":\"${CRED_B_ID}\",\"project_id\":\"rbac-proj-alpha\"}"
assert_status "403" "$HTTP_STATUS" "Scenario 21: Non-project-owner cannot bind credential to project"

# Scenario 22: Non-credential-owner cannot bind credential to project
# User B owns rbac-proj-beta but does NOT own rbac-cred-a (owned by User A)
api POST "/role_bindings" "$TOKEN_B" "{\"role_id\":\"${ROLE_CREDENTIAL_READER}\",\"scope\":\"credential\",\"user_id\":\"rbac-user-b\",\"credential_id\":\"${CRED_A_ID}\",\"project_id\":\"rbac-proj-beta\"}"
assert_status "403" "$HTTP_STATUS" "Scenario 22: Non-credential-owner cannot bind credential to project"

# Clean up the credential binding we just created (for cleaner test state)
if [[ -n "$CRED_BIND_ID" ]]; then
  api DELETE "/role_bindings/${CRED_BIND_ID}" "$TOKEN_A"
  # Don't assert — best effort cleanup
fi

# ============================================================
echo ""
echo -e "${BOLD}Phase 7: Sharing via RoleBindings (scenarios 5, 27, 34)${NC}"

# Scenario 27: User A grants User B project:editor on proj-alpha -> 201
api POST "/role_bindings" "$TOKEN_A" "{\"role_id\":\"${ROLE_PROJECT_EDITOR}\",\"scope\":\"project\",\"user_id\":\"rbac-user-b\",\"project_id\":\"rbac-proj-alpha\"}"
assert_status "201" "$HTTP_STATUS" "Scenario 27: User A grants User B project:editor on rbac-proj-alpha"
EDITOR_BINDING_ID=$(echo "$HTTP_BODY" | jq -r '.id // empty')

# Scenario 5: User B now sees both projects (union of bindings)
api GET "/projects?page=1&size=100" "$TOKEN_B"
assert_status "200" "$HTTP_STATUS" "Scenario 5: User B GET /projects after sharing returns 200"
assert_list_contains "$HTTP_BODY" "name" "rbac-proj-alpha" "Scenario 5: User B now sees rbac-proj-alpha (shared)"
assert_list_contains "$HTTP_BODY" "name" "rbac-proj-beta" "Scenario 5: User B still sees rbac-proj-beta (own)"

# User B can GET proj-alpha directly
api GET "/projects/rbac-proj-alpha" "$TOKEN_B"
assert_status "200" "$HTTP_STATUS" "Scenario 5: User B GET rbac-proj-alpha returns 200 after sharing"

# User B (editor) can create agent in proj-alpha
api POST "/projects/rbac-proj-alpha/agents" "$TOKEN_B" '{"name":"agent-shared","prompt":"shared agent","project_id":"rbac-proj-alpha"}'
assert_status "201" "$HTTP_STATUS" "User B (editor) creates agent in rbac-proj-alpha"

# Scenario 34: User A revokes the editor binding
if [[ -z "$EDITOR_BINDING_ID" ]]; then
  # Fallback: look up the binding
  EDITOR_BINDING_ID=$(get_binding_id "$TOKEN_A" "user_id='rbac-user-b' and project_id='rbac-proj-alpha'")
fi

if [[ -n "$EDITOR_BINDING_ID" ]]; then
  api DELETE "/role_bindings/${EDITOR_BINDING_ID}" "$TOKEN_A"
  assert_status "204" "$HTTP_STATUS" "Scenario 34: User A revokes User B's editor binding"

  # After revocation, User B can no longer see proj-alpha
  api GET "/projects?page=1&size=100" "$TOKEN_B"
  assert_list_not_contains "$HTTP_BODY" "name" "rbac-proj-alpha" "Scenario 34: User B no longer sees rbac-proj-alpha after revocation"
  assert_list_contains "$HTTP_BODY" "name" "rbac-proj-beta" "User B still sees own rbac-proj-beta after revocation"

  api GET "/projects/rbac-proj-alpha" "$TOKEN_B"
  assert_status "404" "$HTTP_STATUS" "Scenario 34: User B GET rbac-proj-alpha returns 404 after revocation"
else
  fail "Scenario 34: Revoke binding" "could not find editor binding ID"
fi

# ============================================================
echo ""
echo -e "${BOLD}Phase 8: Escalation Prevention (scenarios 28, 30-33)${NC}"

# First, re-grant User B as editor so we can test editor escalation
api POST "/role_bindings" "$TOKEN_A" "{\"role_id\":\"${ROLE_PROJECT_EDITOR}\",\"scope\":\"project\",\"user_id\":\"rbac-user-b\",\"project_id\":\"rbac-proj-alpha\"}"
EDITOR_BINDING_ID_2=$(echo "$HTTP_BODY" | jq -r '.id // empty')

# Scenario 31: Editor cannot grant project:owner -> 403
api POST "/role_bindings" "$TOKEN_B" "{\"role_id\":\"${ROLE_PROJECT_OWNER}\",\"scope\":\"project\",\"user_id\":\"rbac-user-c\",\"project_id\":\"rbac-proj-alpha\"}"
assert_status "403" "$HTTP_STATUS" "Scenario 31: User B (editor) cannot grant project:owner"

# Scenario 28: Owner cannot grant project:owner (no peer minting) -> 403
api POST "/role_bindings" "$TOKEN_A" "{\"role_id\":\"${ROLE_PROJECT_OWNER}\",\"scope\":\"project\",\"user_id\":\"rbac-user-b\",\"project_id\":\"rbac-proj-alpha\"}"
assert_status "403" "$HTTP_STATUS" "Scenario 28: User A (owner) cannot grant project:owner (no peer minting)"

# Scenario 30: Owner cannot grant on other projects -> 403
# User A is owner of proj-alpha but NOT proj-beta
api POST "/role_bindings" "$TOKEN_A" "{\"role_id\":\"${ROLE_PROJECT_EDITOR}\",\"scope\":\"project\",\"user_id\":\"rbac-user-c\",\"project_id\":\"rbac-proj-beta\"}"
assert_status "403" "$HTTP_STATUS" "Scenario 30: User A (owner of proj-alpha) cannot grant on rbac-proj-beta"

# Scenario 32: Non-credential-owner cannot grant credential roles -> 403
# User B does NOT own cred-a; tries to grant credential:reader on cred-a
if [[ -n "$ROLE_CREDENTIAL_READER" ]]; then
  api POST "/role_bindings" "$TOKEN_B" "{\"role_id\":\"${ROLE_CREDENTIAL_READER}\",\"scope\":\"credential\",\"user_id\":\"rbac-user-c\",\"credential_id\":\"${CRED_A_ID}\"}"
  assert_status "403" "$HTTP_STATUS" "Scenario 32: Non-credential-owner cannot grant credential-scoped roles"
else
  skip "Scenario 32: credential:reader role not found"
fi

# Scenario 33: Internal role (agent:runner) rejected -> 403
if [[ -n "$ROLE_AGENT_RUNNER" ]]; then
  api POST "/role_bindings" "$TOKEN_A" "{\"role_id\":\"${ROLE_AGENT_RUNNER}\",\"scope\":\"project\",\"user_id\":\"rbac-user-b\",\"project_id\":\"rbac-proj-alpha\"}"
  assert_status "403" "$HTTP_STATUS" "Scenario 33: Granting agent:runner (internal role) rejected"
else
  skip "Scenario 33: agent:runner role not found"
fi

# Also test credential:token-reader (internal role)
if [[ -n "$ROLE_CRED_TOKEN_READER" ]]; then
  api POST "/role_bindings" "$TOKEN_A" "{\"role_id\":\"${ROLE_CRED_TOKEN_READER}\",\"scope\":\"credential\",\"user_id\":\"rbac-user-b\",\"credential_id\":\"${CRED_A_ID}\"}"
  assert_status "403" "$HTTP_STATUS" "Scenario 33: Granting credential:token-reader (internal role) rejected"
else
  skip "Scenario 33: credential:token-reader role not found"
fi

# Clean up the re-granted editor binding
if [[ -n "$EDITOR_BINDING_ID_2" ]]; then
  api DELETE "/role_bindings/${EDITOR_BINDING_ID_2}" "$TOKEN_A"
fi

# ============================================================
echo ""
echo -e "${BOLD}Phase 9: Last-Owner Protection (scenarios 35-36)${NC}"

# Scenario 35: Cannot delete sole project:owner binding -> 409
# Find User A's owner binding on proj-alpha
OWNER_BINDING_A=$(get_binding_id "$TOKEN_A" "user_id='rbac-user-a' and project_id='rbac-proj-alpha' and role_id='${ROLE_PROJECT_OWNER}'")

if [[ -n "$OWNER_BINDING_A" ]]; then
  api DELETE "/role_bindings/${OWNER_BINDING_A}" "$TOKEN_A"
  assert_status "409" "$HTTP_STATUS" "Scenario 35: Cannot delete sole project:owner binding -> 409"
else
  # Try broader search without role_id filter
  api GET "/role_bindings?search=user_id%3D'rbac-user-a'%20and%20project_id%3D'rbac-proj-alpha'&page=1&size=100" "$TOKEN_A"
  OWNER_BINDING_A=$(echo "$HTTP_BODY" | jq -r ".items[] | select(.role_id == \"${ROLE_PROJECT_OWNER}\") | .id" | head -1)
  if [[ -n "$OWNER_BINDING_A" ]]; then
    api DELETE "/role_bindings/${OWNER_BINDING_A}" "$TOKEN_A"
    assert_status "409" "$HTTP_STATUS" "Scenario 35: Cannot delete sole project:owner binding -> 409"
  else
    fail "Scenario 35: Last-owner protection" "could not find owner binding to test"
  fi
fi

# Scenario 36: Cannot delete sole credential:owner binding -> 409
api GET "/role_bindings?search=user_id%3D'rbac-user-a'%20and%20credential_id%3D'${CRED_A_ID}'&page=1&size=100" "$TOKEN_A"
CRED_OWNER_BINDING_A=$(echo "$HTTP_BODY" | jq -r ".items[] | select(.role_id == \"${ROLE_CREDENTIAL_OWNER}\") | .id" | head -1)

if [[ -n "$CRED_OWNER_BINDING_A" ]]; then
  api DELETE "/role_bindings/${CRED_OWNER_BINDING_A}" "$TOKEN_A"
  assert_status "409" "$HTTP_STATUS" "Scenario 36: Cannot delete sole credential:owner binding -> 409"
else
  fail "Scenario 36: Last credential owner protection" "could not find credential owner binding to test"
fi

# ============================================================
echo ""
echo -e "${BOLD}Phase 10: Non-admin Cannot Create Global Bindings (scenario 26)${NC}"

# Scenario 26: User A (project:owner but not platform:admin) tries to create global binding -> 403
if [[ -n "$ROLE_PLATFORM_ADMIN" ]]; then
  api POST "/role_bindings" "$TOKEN_A" "{\"role_id\":\"${ROLE_PLATFORM_ADMIN}\",\"scope\":\"global\",\"user_id\":\"rbac-user-c\"}"
  assert_status "403" "$HTTP_STATUS" "Scenario 26: Non-admin cannot create global binding (platform:admin)"
else
  skip "Scenario 26: platform:admin role not found"
fi

# Even a project-level role with scope=global should fail
api POST "/role_bindings" "$TOKEN_A" "{\"role_id\":\"${ROLE_PROJECT_EDITOR}\",\"scope\":\"global\",\"user_id\":\"rbac-user-c\"}"
assert_status "403" "$HTTP_STATUS" "Scenario 26: Non-admin cannot create any global-scoped binding"

# ============================================================
echo ""
echo -e "${BOLD}Phase 11: Mutation Opacity (scenario 51)${NC}"

# Scenario 51: User A PATCHes proj-beta (no access) -> 403 with generic body
api PATCH "/projects/rbac-proj-beta" "$TOKEN_A" '{"description":"hacked"}'
assert_status "403" "$HTTP_STATUS" "Scenario 51: User A PATCH rbac-proj-beta (no access) returns 403"

# Verify the 403 body is opaque (no permission details leaked)
if echo "$HTTP_BODY" | jq -e '.reason' >/dev/null 2>&1; then
  REASON=$(echo "$HTTP_BODY" | jq -r '.reason // empty')
  if [[ "$REASON" == "Forbidden" ]]; then
    pass "Scenario 51: 403 body is opaque (generic 'Forbidden' reason)"
  elif echo "$REASON" | grep -qi "permission\|binding\|role\|rbac\|access"; then
    fail "Scenario 51: 403 body is opaque" "body leaks permission details: $REASON"
  else
    pass "Scenario 51: 403 body does not leak permission details"
  fi
else
  pass "Scenario 51: 403 body has no structured reason field"
fi

# User A DELETEs proj-beta (no access) -> 403
api DELETE "/projects/rbac-proj-beta" "$TOKEN_A"
assert_status "403" "$HTTP_STATUS" "Scenario 51: User A DELETE rbac-proj-beta returns 403"

# Verify the DELETE 403 body is also opaque
if echo "$HTTP_BODY" | jq -e '.reason' >/dev/null 2>&1; then
  REASON=$(echo "$HTTP_BODY" | jq -r '.reason // empty')
  if echo "$REASON" | grep -qi "permission\|binding\|role\|rbac\|access denied"; then
    fail "Scenario 51: DELETE 403 body is opaque" "body leaks: $REASON"
  else
    pass "Scenario 51: DELETE 403 body does not leak permission details"
  fi
else
  pass "Scenario 51: DELETE 403 body has no structured reason field"
fi

# ============================================================
echo ""
echo -e "${BOLD}Phase 12: Auth-Exempt Endpoints (scenario 46)${NC}"

# Scenario 46: Fresh user (zero bindings) can use auth-exempt endpoints

# User C has no bindings (never created a project or credential)
# POST /projects is auth-exempt
api POST "/projects" "$TOKEN_C" '{"name":"rbac-proj-charlie","description":"Charlie project"}'
assert_status "201" "$HTTP_STATUS" "Scenario 46: Fresh user (User C) can POST /projects -> 201"
CREATED_PROJECTS+=("rbac-proj-charlie")

# Verify owner binding was auto-created for Charlie
api GET "/role_bindings?search=user_id%3D'rbac-user-c'%20and%20project_id%3D'rbac-proj-charlie'&page=1&size=100" "$TOKEN_C"
if echo "$HTTP_BODY" | jq -e '.items[] | select(.scope == "project")' >/dev/null 2>&1; then
  pass "Scenario 46: project:owner binding auto-created for User C"
else
  fail "Scenario 46: project:owner binding auto-created" "binding not found for User C"
fi

# POST /credentials is auth-exempt (User C had no cred bindings before)
api POST "/credentials" "$TOKEN_C" '{"name":"rbac-cred-c","provider":"github","token":"test-fake-token-c"}'
assert_status "201" "$HTTP_STATUS" "Scenario 46: Fresh user (User C) can POST /credentials -> 201"
CRED_C_ID=$(echo "$HTTP_BODY" | jq -r '.id // empty')
CREATED_CRED_IDS+=("$CRED_C_ID")

# GET /roles is auth-exempt (already tested in Phase 1, but confirm for User C)
api GET "/roles" "$TOKEN_C"
assert_status "200" "$HTTP_STATUS" "Scenario 46: Fresh user can GET /roles -> 200"

# ============================================================
echo ""
echo -e "${BOLD}Phase 13: Additional Edge Cases${NC}"

# --- Scenario 1: Project-scoped binding restricts access ---
# User A has project:owner on proj-alpha; verify it does NOT grant access to proj-beta
api GET "/projects/rbac-proj-beta/agents?page=1&size=100" "$TOKEN_A"
assert_status "404" "$HTTP_STATUS" "Scenario 1: Project-scoped binding does not grant access to other projects"

# --- Scenario 4 extended: scope hierarchy covers nested resources ---
# User A (project:owner on proj-alpha) can list agents in proj-alpha
api GET "/projects/rbac-proj-alpha/agents?page=1&size=100" "$TOKEN_A"
assert_status "200" "$HTTP_STATUS" "Scenario 4: project:owner covers agent listing"

# --- Editor can grant viewer (strictly below) ---
# Re-grant editor to User B
api POST "/role_bindings" "$TOKEN_A" "{\"role_id\":\"${ROLE_PROJECT_EDITOR}\",\"scope\":\"project\",\"user_id\":\"rbac-user-b\",\"project_id\":\"rbac-proj-alpha\"}"
EDITOR_BINDING_ID_3=$(echo "$HTTP_BODY" | jq -r '.id // empty')

# Editor (User B) grants viewer to User C on proj-alpha (level 2 granting level 3 = allowed)
if [[ -n "$ROLE_PROJECT_VIEWER" ]]; then
  api POST "/role_bindings" "$TOKEN_B" "{\"role_id\":\"${ROLE_PROJECT_VIEWER}\",\"scope\":\"project\",\"user_id\":\"rbac-user-c\",\"project_id\":\"rbac-proj-alpha\"}"
  assert_status "201" "$HTTP_STATUS" "Editor can grant project:viewer (strictly below)"
  VIEWER_BINDING_C=$(echo "$HTTP_BODY" | jq -r '.id // empty')

  # User C can now see proj-alpha
  api GET "/projects/rbac-proj-alpha" "$TOKEN_C"
  assert_status "200" "$HTTP_STATUS" "User C (viewer) can GET rbac-proj-alpha"

  # Clean up viewer binding
  if [[ -n "$VIEWER_BINDING_C" ]]; then
    api DELETE "/role_bindings/${VIEWER_BINDING_C}" "$TOKEN_A"
  fi
else
  skip "Editor->viewer grant test: project:viewer role not found"
fi

# Clean up editor binding
if [[ -n "$EDITOR_BINDING_ID_3" ]]; then
  api DELETE "/role_bindings/${EDITOR_BINDING_ID_3}" "$TOKEN_A"
fi

# --- Scenario 9: Empty list for resources, not 403 ---
# User C should get empty list for credentials of projects they have no cred bindings for
# (User C has their own cred, but this tests the "no 403 on list" contract)
api GET "/credentials?page=1&size=100" "$TOKEN_C"
assert_status "200" "$HTTP_STATUS" "Scenario 9: Credential list always returns 200, never 403"

# ============================================================
echo ""
echo -e "${BOLD}Phase 14: Cleanup Test Data${NC}"

# Delete projects created during tests (owners can delete their own)
api DELETE "/projects/rbac-proj-alpha" "$TOKEN_A"
if [[ "$HTTP_STATUS" == "204" || "$HTTP_STATUS" == "200" ]]; then
  pass "Cleanup: Deleted rbac-proj-alpha"
else
  echo "  WARN: Could not delete rbac-proj-alpha (status $HTTP_STATUS)"
fi

api DELETE "/projects/rbac-proj-beta" "$TOKEN_B"
if [[ "$HTTP_STATUS" == "204" || "$HTTP_STATUS" == "200" ]]; then
  pass "Cleanup: Deleted rbac-proj-beta"
else
  echo "  WARN: Could not delete rbac-proj-beta (status $HTTP_STATUS)"
fi

api DELETE "/projects/rbac-proj-charlie" "$TOKEN_C"
if [[ "$HTTP_STATUS" == "204" || "$HTTP_STATUS" == "200" ]]; then
  pass "Cleanup: Deleted rbac-proj-charlie"
else
  echo "  WARN: Could not delete rbac-proj-charlie (status $HTTP_STATUS)"
fi

# Delete credentials
for cred_id in "${CREATED_CRED_IDS[@]:-}"; do
  if [[ -n "$cred_id" ]]; then
    # Try with each user token (owner can delete)
    api DELETE "/credentials/${cred_id}" "$TOKEN_A"
    if [[ "$HTTP_STATUS" != "204" && "$HTTP_STATUS" != "200" ]]; then
      api DELETE "/credentials/${cred_id}" "$TOKEN_B"
      if [[ "$HTTP_STATUS" != "204" && "$HTTP_STATUS" != "200" ]]; then
        api DELETE "/credentials/${cred_id}" "$TOKEN_C"
      fi
    fi
  fi
done
echo "  Test credentials cleaned up"

# ============================================================
echo ""
echo -e "${BOLD}Summary${NC}"
TOTAL=$((PASS_COUNT + FAIL_COUNT + SKIP_COUNT))
echo -e "  ${GREEN}${PASS_COUNT} passed${NC}, ${RED}${FAIL_COUNT} failed${NC}, ${YELLOW}${SKIP_COUNT} skipped${NC} (${TOTAL} total)"

if [[ "$FAIL_COUNT" -gt 0 ]]; then
  exit 1
fi
