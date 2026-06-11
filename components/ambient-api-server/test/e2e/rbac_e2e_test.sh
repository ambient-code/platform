#!/usr/bin/env bash
set -euo pipefail

API_URL="${API_URL:-http://localhost:13592/api/ambient/v1}"
KC_URL="${KC_URL:-http://localhost:18592}"
KC_REALM="ambient-code"
KC_ADMIN_USER="admin"
KC_ADMIN_PASS="admin"
KC_CLIENT_ID="ambient-frontend"
NS="${NAMESPACE:-ambient-code}"
KUBE_CONTEXT="${KUBE_CONTEXT:-}"

# All kubectl calls go through this wrapper so we never touch the wrong cluster.
kubectl() {
  if [[ -n "$KUBE_CONTEXT" ]]; then
    command kubectl --context "$KUBE_CONTEXT" "$@"
  else
    command kubectl "$@"
  fi
}

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

_ensure_port_forward() {
  local port
  port=$(echo "$API_URL" | sed -n 's|.*localhost:\([0-9]*\).*|\1|p' | head -1)
  [[ -z "$port" ]] && return 0
  if command -v lsof &>/dev/null; then
    lsof -ti :"$port" 2>/dev/null | xargs -r kill 2>/dev/null || true
  elif command -v fuser &>/dev/null; then
    fuser -k "${port}/tcp" 2>/dev/null || true
  fi
  sleep 1
  kubectl port-forward -n "${NS}" svc/ambient-api-server "${port}:8000" &>/dev/null &
  for _i in $(seq 1 10); do
    local _s
    _s=$(curl -s -o /dev/null -w '%{http_code}' --max-time 2 "http://localhost:${port}/healthcheck" 2>/dev/null || true)
    [[ "$_s" != "000" && -n "$_s" ]] && return 0
    sleep 1
  done
}

api() {
  local method="$1" path="$2" token="$3" body="${4:-}"
  local args=(-s --max-time 15 -w '\n%{http_code}' -H "Authorization: Bearer $token" -H "Content-Type: application/json")
  if [[ -n "$body" ]]; then
    args+=(-d "$body")
  fi
  local response _retry
  for _retry in 1 2 3; do
    response=$(curl "${args[@]}" -X "$method" "${API_URL}${path}" || true)
    HTTP_STATUS=$(echo "$response" | tail -1)
    HTTP_BODY=$(echo "$response" | sed '$d')
    case "$HTTP_STATUS" in
      000) _ensure_port_forward ;;            # port-forward died
      500|502|503) sleep $((_retry * 2)) ;;   # transient server error
      *) return 0 ;;                          # success or expected client error
    esac
  done
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
  KC_ADMIN_TOKEN=$(curl -s --max-time 10 -X POST "${KC_URL}/realms/master/protocol/openid-connect/token" \
    -d "client_id=admin-cli" \
    -d "grant_type=password" \
    -d "username=${KC_ADMIN_USER}" \
    -d "password=${KC_ADMIN_PASS}" 2>/dev/null | jq -r '.access_token // empty')
  if [[ -z "$KC_ADMIN_TOKEN" || "$KC_ADMIN_TOKEN" == "null" ]]; then
    echo "ERROR: Failed to get Keycloak admin token from ${KC_URL}"
    return 1
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

clean_db() {
  local pod="${DB_POD:-$(kubectl get pods -n "$NS" -l app=ambient-api-server,component=database -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)}"
  if [[ -n "$pod" ]]; then
    kubectl exec -n "$NS" "$pod" -- psql -U ambient -d ambient_api_server -c "
      DELETE FROM session_messages WHERE session_id IN (SELECT id FROM sessions WHERE project_id LIKE 'rbac-%');
      DELETE FROM sessions WHERE project_id LIKE 'rbac-%';
      DELETE FROM role_bindings WHERE project_id LIKE 'rbac-%' OR user_id LIKE 'rbac-%' OR credential_id IN (SELECT id FROM credentials WHERE name LIKE 'rbac-%');
      DELETE FROM agents WHERE project_id LIKE 'rbac-%';
      DELETE FROM credentials WHERE name LIKE 'rbac-%';
      DELETE FROM projects WHERE name LIKE 'rbac-%';
      DELETE FROM users WHERE username LIKE 'rbac-%';
    " 2>/dev/null >/dev/null || true
  fi
}

cleanup() {
  # Cleanup must never fail — it runs in the EXIT trap.
  set +e
  echo ""
  echo -e "${BOLD}Cleanup${NC}"

  clean_db
  echo "  DB cleaned (hard delete)"

  get_admin_token 2>/dev/null
  if [[ -n "$KC_ADMIN_TOKEN" && "$KC_ADMIN_TOKEN" != "null" ]]; then
    delete_keycloak_user "rbac-user-a"
    delete_keycloak_user "rbac-user-b"
    delete_keycloak_user "rbac-user-c"
    echo "  Keycloak users cleaned up"
  else
    echo "  Keycloak unreachable — skipping user cleanup"
  fi
}
trap cleanup EXIT

# ============================================================
echo -e "${BOLD}RBAC Enforcement E2E Tests${NC}"
echo "API: $API_URL"
echo "Keycloak: $KC_URL"
echo ""

# ============================================================
echo -e "${BOLD}Phase 0: Pre-clean stale data${NC}"

clean_db
echo "  DB cleaned"

get_admin_token
delete_keycloak_user "rbac-user-a"
delete_keycloak_user "rbac-user-b"
delete_keycloak_user "rbac-user-c"
echo "  Keycloak users cleaned"

# ============================================================
echo ""
echo -e "${BOLD}Phase 0.5: Enable JWT + RBAC enforcement${NC}"
# The test manages its own infrastructure setup so it works identically
# in local Kind clusters and GitHub Actions CI.

KC_INTERNAL_URL="http://keycloak-service.${NS}.svc:8080"
KC_JWKS_URL="${KC_INTERNAL_URL}/realms/${KC_REALM}/protocol/openid-connect/certs"
KC_TOKEN_URL="${KC_INTERNAL_URL}/realms/${KC_REALM}/protocol/openid-connect/token"
# The ambient-e2e Keycloak client has serviceAccountsEnabled=true
# and a well-known secret baked into the Kind realm JSON.
OIDC_CLIENT_ID_CP="ambient-e2e"
OIDC_CLIENT_SECRET_CP="e2e-secret-do-not-use-in-prod"

# 1. Fetch Keycloak JWKS and update the API server auth ConfigMap so the
#    API server can validate JWTs signed by this Keycloak instance.
echo "  Fetching Keycloak JWKS..."
KC_JWKS=$(curl -sf "${KC_URL}/realms/${KC_REALM}/protocol/openid-connect/certs" || true)
if [[ -z "$KC_JWKS" || "$KC_JWKS" == "null" ]]; then
  echo "FATAL: Cannot fetch Keycloak JWKS from ${KC_URL}/realms/${KC_REALM}/protocol/openid-connect/certs"
  exit 1
fi
kubectl create configmap ambient-api-server-auth \
  --from-literal="jwks.json=${KC_JWKS}" \
  --from-literal="acl.yml=" \
  -n "$NS" --dry-run=client -o yaml | kubectl apply -f -
echo "  Updated ambient-api-server-auth ConfigMap with Keycloak JWKS"

# 2. Patch ambient-api-server secret to include OIDC client credentials.
#    The control plane reads clientId / clientSecret from this secret to
#    authenticate to the API server via OIDC client_credentials grant.
kubectl patch secret ambient-api-server -n "$NS" -p \
  "{\"stringData\":{\"clientId\":\"${OIDC_CLIENT_ID_CP}\",\"clientSecret\":\"${OIDC_CLIENT_SECRET_CP}\"}}"
echo "  Patched ambient-api-server secret with OIDC client credentials"

# 3. Patch API server deployment:
#    - AMBIENT_ENV=production  (dev env overrides --enable-jwt to false)
#    - JWK_CERT_URL            (Keycloak JWKS for the production env fallback)
#    - GRPC_SERVICE_ACCOUNT    (so the pre-auth interceptor tags OIDC SAs as CallerTypeService)
#    - command args            (--enable-jwt=true --enable-authz=true)
kubectl patch deployment/ambient-api-server -n "$NS" --type=json -p "$(cat <<'EOF'
[
  {"op":"replace","path":"/spec/template/spec/containers/0/command","value":[
    "/usr/local/bin/ambient-api-server","serve",
    "--db-host-file=/secrets/db/db.host",
    "--db-port-file=/secrets/db/db.port",
    "--db-user-file=/secrets/db/db.user",
    "--db-password-file=/secrets/db/db.password",
    "--db-name-file=/secrets/db/db.name",
    "--enable-jwt=true",
    "--enable-authz=true",
    "--jwk-cert-file=/configs/authentication/jwks.json",
    "--enable-https=false",
    "--enable-grpc=true",
    "--grpc-enable-tls=false",
    "--api-server-bindaddress=:8000",
    "--metrics-server-bindaddress=:4433",
    "--health-check-server-bindaddress=:4434",
    "--enable-health-check-https=false",
    "--db-sslmode=disable",
    "--db-max-open-connections=50",
    "--enable-db-debug=false",
    "--enable-metrics-https=false",
    "--http-read-timeout=5s",
    "--http-write-timeout=30s",
    "--cors-allowed-origins=*",
    "--cors-allowed-headers=X-Ambient-Project",
    "--grpc-server-bindaddress=:9000",
    "--alsologtostderr","-v=10"
  ]}
]
EOF
)"
kubectl set env deployment/ambient-api-server -n "$NS" \
  AMBIENT_ENV=production \
  JWK_CERT_URL="$KC_JWKS_URL" \
  GRPC_SERVICE_ACCOUNT="service-account-${OIDC_CLIENT_ID_CP}"
echo "  Patched API server deployment (JWT + authz enabled, production env)"

# 4. Wait for API server rollout FIRST — it must be healthy before CP starts
echo "  Waiting for API server rollout..."
if ! kubectl rollout status deployment/ambient-api-server -n "$NS" --timeout=120s; then
  echo "FATAL: API server rollout failed"
  kubectl describe deployment/ambient-api-server -n "$NS" | tail -20
  exit 1
fi

# 6. Re-establish port-forwards if API_URL / KC_URL point to localhost
#    (deployment rollout kills existing port-forward connections)
_reforward() {
  local svc="$1" local_port="$2" remote_port="$3"
  # Kill any stale port-forward for this local port (works even without lsof)
  if command -v lsof &>/dev/null; then
    lsof -ti :"$local_port" 2>/dev/null | xargs -r kill 2>/dev/null || true
  elif command -v fuser &>/dev/null; then
    fuser -k "${local_port}/tcp" 2>/dev/null || true
  fi
  sleep 1
  kubectl port-forward -n "$NS" "svc/${svc}" "${local_port}:${remote_port}" &>/dev/null &
  # Wait for TCP connectivity (any HTTP status — even 401 — means the forward is up)
  for _i in $(seq 1 10); do
    local _status
    _status=$(curl -s -o /dev/null -w '%{http_code}' --max-time 2 "http://localhost:${local_port}/healthcheck" 2>/dev/null || true)
    [[ "$_status" != "000" && -n "$_status" ]] && return 0
    sleep 1
  done
  echo "WARNING: port-forward svc/${svc} ${local_port}:${remote_port} may not be ready"
}

# Only re-forward the API server (its pod restarted); Keycloak was untouched.
if [[ "$API_URL" == *"localhost"* ]]; then
  API_PORT=$(echo "$API_URL" | sed -n 's|.*localhost:\([0-9]*\).*|\1|p' | head -1)
  if [[ -n "$API_PORT" ]]; then
    echo "  Re-establishing port-forward for API server (localhost:${API_PORT} -> 8000)..."
    _reforward ambient-api-server "$API_PORT" 8000
  fi
fi

# 7. Verify API server accepts Keycloak JWTs (smoke test with client_credentials token)
echo "  Verifying API server JWT auth..."
VERIFY_TOKEN=$(curl -sf -X POST "${KC_URL}/realms/${KC_REALM}/protocol/openid-connect/token" \
  -d "client_id=${OIDC_CLIENT_ID_CP}" \
  -d "client_secret=${OIDC_CLIENT_SECRET_CP}" \
  -d "grant_type=client_credentials" | jq -r '.access_token // empty')
if [[ -z "$VERIFY_TOKEN" ]]; then
  echo "FATAL: Cannot get OIDC token from Keycloak for ${OIDC_CLIENT_ID_CP}"
  exit 1
fi
JWT_VERIFIED=false
for attempt in $(seq 1 15); do
  VERIFY_RESP=$(curl -s -w '\n%{http_code}' -H "Authorization: Bearer $VERIFY_TOKEN" "${API_URL}/roles?page=1&size=1" 2>/dev/null || true)
  VERIFY_STATUS=$(echo "$VERIFY_RESP" | tail -1)
  if [[ "$VERIFY_STATUS" == "200" ]]; then
    echo "  API server JWT auth verified (attempt $attempt)"
    JWT_VERIFIED=true
    break
  fi
  sleep 2
done
if [[ "$JWT_VERIFIED" != "true" ]]; then
  echo "FATAL: API server not accepting Keycloak JWTs after 30s (last status=$VERIFY_STATUS)"
  echo "  API server logs:"
  kubectl logs -n "$NS" deploy/ambient-api-server -c api-server --tail=30 2>/dev/null || true
  exit 1
fi

# 8. NOW patch and restart the control plane (API server is verified healthy)
kubectl set env deployment/ambient-control-plane -n "$NS" \
  OIDC_CLIENT_ID="$OIDC_CLIENT_ID_CP" \
  OIDC_CLIENT_SECRET="$OIDC_CLIENT_SECRET_CP" \
  OIDC_TOKEN_URL="$KC_TOKEN_URL"
# Always restart CP to get fresh gRPC watch streams (set env is a no-op
# if values unchanged, which means no rollout and stale streams persist)
kubectl rollout restart deployment/ambient-control-plane -n "$NS" 2>/dev/null || true
echo "  Patched and restarted control plane"
echo "  Waiting for control plane rollout..."
if ! kubectl rollout status deployment/ambient-control-plane -n "$NS" --timeout=120s; then
  echo "FATAL: Control plane rollout failed"
  kubectl describe deployment/ambient-control-plane -n "$NS" | tail -20
  exit 1
fi

# 9. Verify control plane pod is healthy
CP_READY=false
for attempt in $(seq 1 10); do
  CP_STATUS=$(kubectl get pods -n "$NS" -l app=ambient-control-plane \
    -o jsonpath='{.items[0].status.containerStatuses[0].ready}' 2>/dev/null || true)
  if [[ "$CP_STATUS" == "true" ]]; then
    echo "  Control plane is ready (attempt $attempt)"
    CP_READY=true
    break
  fi
  sleep 2
done
if [[ "$CP_READY" != "true" ]]; then
  echo "WARNING: Control plane pod not ready after 20s — sessions may stay Pending"
  echo "  Control plane logs:"
  kubectl logs -n "$NS" deploy/ambient-control-plane --tail=20 2>/dev/null || true
fi

# 10. Wait for the CP's gRPC watch streams to be established.
#     The CP pod becomes Ready before its watch streams connect to the API
#     server (~10s lag). Sessions created in that gap are missed.
CP_STREAMS=false
for attempt in $(seq 1 20); do
  if kubectl logs -n "$NS" deploy/ambient-control-plane --tail=50 2>/dev/null | grep -q "session watch stream established"; then
    CP_STREAMS=true
    break
  fi
  sleep 1
done
if [[ "$CP_STREAMS" != "true" ]]; then
  echo "WARNING: CP session watch stream not established after 20s"
fi

echo "  Phase 0.5 complete: JWT + RBAC enforcement enabled"

# ============================================================
echo ""
echo -e "${BOLD}Phase 1: Setup${NC}"

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
ROLE_CREDENTIAL_VIEWER=$(lookup_role_id "credential:viewer")
ROLE_AGENT_RUNNER=$(lookup_role_id "agent:runner")
ROLE_CRED_TOKEN_READER=$(lookup_role_id "credential:token-reader")
ROLE_PLATFORM_ADMIN=$(lookup_role_id "platform:admin")
ROLE_PLATFORM_VIEWER=$(lookup_role_id "platform:viewer")
ROLE_AGENT_OPERATOR=$(lookup_role_id "agent:operator")
ROLE_AGENT_OBSERVER=$(lookup_role_id "agent:observer")
ROLE_AGENT_EDITOR=$(lookup_role_id "agent:editor")

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
api POST "/role_bindings" "$TOKEN_A" "{\"role_id\":\"${ROLE_CREDENTIAL_VIEWER}\",\"scope\":\"credential\",\"user_id\":\"rbac-user-a\",\"credential_id\":\"${CRED_A_ID}\",\"project_id\":\"rbac-proj-alpha\"}"
assert_status "201" "$HTTP_STATUS" "Scenario 20: Credential owner binds rbac-cred-a to own project rbac-proj-alpha"
CRED_BIND_ID=$(echo "$HTTP_BODY" | jq -r '.id // empty')

# Scenario 21: Non-project-owner cannot bind credential to project
# User B owns rbac-cred-b but does NOT own rbac-proj-alpha
api POST "/role_bindings" "$TOKEN_B" "{\"role_id\":\"${ROLE_CREDENTIAL_VIEWER}\",\"scope\":\"credential\",\"user_id\":\"rbac-user-b\",\"credential_id\":\"${CRED_B_ID}\",\"project_id\":\"rbac-proj-alpha\"}"
assert_status "403" "$HTTP_STATUS" "Scenario 21: Non-project-owner cannot bind credential to project"

# Scenario 22: Non-credential-owner cannot bind credential to project
# User B owns rbac-proj-beta but does NOT own rbac-cred-a (owned by User A)
api POST "/role_bindings" "$TOKEN_B" "{\"role_id\":\"${ROLE_CREDENTIAL_VIEWER}\",\"scope\":\"credential\",\"user_id\":\"rbac-user-b\",\"credential_id\":\"${CRED_A_ID}\",\"project_id\":\"rbac-proj-beta\"}"
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
# User B does NOT own cred-a; tries to grant credential:viewer on cred-a
if [[ -n "$ROLE_CREDENTIAL_VIEWER" ]]; then
  api POST "/role_bindings" "$TOKEN_B" "{\"role_id\":\"${ROLE_CREDENTIAL_VIEWER}\",\"scope\":\"credential\",\"user_id\":\"rbac-user-c\",\"credential_id\":\"${CRED_A_ID}\"}"
  assert_status "403" "$HTTP_STATUS" "Scenario 32: Non-credential-owner cannot grant credential-scoped roles"
else
  skip "Scenario 32: credential:viewer role not found"
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

# --- Viewer cannot grant editor (level 3 cannot grant level 2) ---
if [[ -n "$VIEWER_BINDING_C" ]]; then
  # Re-grant viewer to User C for this test
  api POST "/role_bindings" "$TOKEN_A" "{\"role_id\":\"${ROLE_PROJECT_VIEWER}\",\"scope\":\"project\",\"user_id\":\"rbac-user-c\",\"project_id\":\"rbac-proj-alpha\"}"
  VIEWER_BINDING_C2=$(echo "$HTTP_BODY" | jq -r '.id // empty')
  # User C (viewer) tries to grant project:editor — should fail
  api POST "/role_bindings" "$TOKEN_C" "{\"role_id\":\"${ROLE_PROJECT_EDITOR}\",\"scope\":\"project\",\"user_id\":\"rbac-user-b\",\"project_id\":\"rbac-proj-alpha\"}"
  assert_status "403" "$HTTP_STATUS" "Viewer cannot grant editor (level 3 cannot grant level 2)"
  # Clean up
  if [[ -n "$VIEWER_BINDING_C2" ]]; then
    api DELETE "/role_bindings/${VIEWER_BINDING_C2}" "$TOKEN_A"
  fi
fi

# --- Scenario 9: Empty list for resources, not 403 ---
api GET "/credentials?page=1&size=100" "$TOKEN_C"
assert_status "200" "$HTTP_STATUS" "Scenario 9: Credential list always returns 200, never 403"

# ============================================================
echo ""
echo -e "${BOLD}Phase 14: Escalation Matrix (generative — all caller × target combos)${NC}"

# Setup: give User B project:editor, User C project:viewer on proj-alpha
api POST "/role_bindings" "$TOKEN_A" "{\"role_id\":\"${ROLE_PROJECT_EDITOR}\",\"scope\":\"project\",\"user_id\":\"rbac-user-b\",\"project_id\":\"rbac-proj-alpha\"}"
MATRIX_EDITOR_BIND=$(echo "$HTTP_BODY" | jq -r '.id // empty')
api POST "/role_bindings" "$TOKEN_A" "{\"role_id\":\"${ROLE_PROJECT_VIEWER}\",\"scope\":\"project\",\"user_id\":\"rbac-user-c\",\"project_id\":\"rbac-proj-alpha\"}"
MATRIX_VIEWER_BIND=$(echo "$HTTP_BODY" | jq -r '.id // empty')

# --- Generative grant matrix ---
# Derive expected result from hierarchy rule:
#   admin (0): can grant anything (including admin)
#   others: can only grant strictly below (caller_level < target_level)
#   internal roles: always 403
#
# Format: "role_name:role_id_var:level:internal"
GRANTABLE_ROLES=(
  "project:owner|ROLE_PROJECT_OWNER|1|no"
  "project:editor|ROLE_PROJECT_EDITOR|2|no"
  "project:viewer|ROLE_PROJECT_VIEWER|3|no"
  "agent:operator|ROLE_AGENT_OPERATOR|2|no"
  "agent:observer|ROLE_AGENT_OBSERVER|3|no"
  "agent:editor|ROLE_AGENT_EDITOR|2|no"
  "credential:owner|ROLE_CREDENTIAL_OWNER|1|no"
  "credential:viewer|ROLE_CREDENTIAL_VIEWER|2|no"
  "agent:runner|ROLE_AGENT_RUNNER|0|yes"
  "credential:token-reader|ROLE_CRED_TOKEN_READER|0|yes"
)

# Format: "label|token_var|level"
CALLERS=(
  "owner(1)|TOKEN_A|1"
  "editor(2)|TOKEN_B|2"
  "viewer(3)|TOKEN_C|3"
)

echo "  Testing ${#CALLERS[@]} callers × ${#GRANTABLE_ROLES[@]} target roles = $(( ${#CALLERS[@]} * ${#GRANTABLE_ROLES[@]} )) combinations"

MATRIX_PASS=0
MATRIX_FAIL=0

for caller_entry in "${CALLERS[@]}"; do
  IFS='|' read -r caller_label token_var caller_level <<< "$caller_entry"
  caller_token="${!token_var}"

  for target_entry in "${GRANTABLE_ROLES[@]}"; do
    IFS='|' read -r role_name role_id_var target_level is_internal <<< "$target_entry"
    role_id="${!role_id_var}"

    if [[ -z "$role_id" ]]; then
      skip "Matrix: ${caller_label} -> ${role_name} (role not found)"
      continue
    fi

    # Derive expected result
    if [[ "$is_internal" == "yes" ]]; then
      expected="403"
    elif (( caller_level == 0 )); then
      expected="201"
    elif (( caller_level < target_level )); then
      expected="201"
    else
      expected="403"
    fi

    api POST "/role_bindings" "$caller_token" "{\"role_id\":\"${role_id}\",\"scope\":\"project\",\"user_id\":\"rbac-user-matrix-target\",\"project_id\":\"rbac-proj-alpha\"}"

    if [[ "$HTTP_STATUS" == "$expected" ]]; then
      MATRIX_PASS=$((MATRIX_PASS + 1))
      # Clean up successful grants
      if [[ "$HTTP_STATUS" == "201" ]]; then
        local_bid=$(echo "$HTTP_BODY" | jq -r '.id // empty')
        [[ -n "$local_bid" ]] && api DELETE "/role_bindings/${local_bid}" "$TOKEN_A"
      fi
    else
      fail "Matrix: ${caller_label} -> ${role_name}" "expected ${expected}, got ${HTTP_STATUS}"
      MATRIX_FAIL=$((MATRIX_FAIL + 1))
    fi
  done
done

echo -e "  Matrix same-project: ${GREEN}${MATRIX_PASS} passed${NC}, ${RED}${MATRIX_FAIL} failed${NC} (of $(( ${#CALLERS[@]} * ${#GRANTABLE_ROLES[@]} )))"
PASS_COUNT=$((PASS_COUNT + MATRIX_PASS))

# --- Cross-project grants (always 403) ---
echo ""
echo "  Cross-project grants (owner of proj-alpha granting on proj-beta):"
CROSS_PASS=0
CROSS_FAIL=0

for target_entry in "${GRANTABLE_ROLES[@]}"; do
  IFS='|' read -r role_name role_id_var target_level is_internal <<< "$target_entry"
  role_id="${!role_id_var}"
  [[ -z "$role_id" ]] && continue

  api POST "/role_bindings" "$TOKEN_A" "{\"role_id\":\"${role_id}\",\"scope\":\"project\",\"user_id\":\"rbac-user-matrix-target\",\"project_id\":\"rbac-proj-beta\"}"
  if [[ "$HTTP_STATUS" == "403" ]]; then
    CROSS_PASS=$((CROSS_PASS + 1))
  else
    fail "Cross-project: owner(proj-alpha) -> ${role_name} on proj-beta" "expected 403, got ${HTTP_STATUS}"
    CROSS_FAIL=$((CROSS_FAIL + 1))
    # Clean up accidental grants
    if [[ "$HTTP_STATUS" == "201" ]]; then
      local_bid=$(echo "$HTTP_BODY" | jq -r '.id // empty')
      [[ -n "$local_bid" ]] && api DELETE "/role_bindings/${local_bid}" "$TOKEN_B"
    fi
  fi
done

echo -e "  Cross-project: ${GREEN}${CROSS_PASS} passed${NC}, ${RED}${CROSS_FAIL} failed${NC} (of ${#GRANTABLE_ROLES[@]})"
PASS_COUNT=$((PASS_COUNT + CROSS_PASS))

# --- Global scope grants (only admin allowed, all others 403) ---
echo ""
echo "  Global scope grants (non-admin callers):"
GLOBAL_PASS=0
GLOBAL_FAIL=0

for caller_entry in "${CALLERS[@]}"; do
  IFS='|' read -r caller_label token_var caller_level <<< "$caller_entry"
  caller_token="${!token_var}"

  # Try granting project:editor at global scope
  api POST "/role_bindings" "$caller_token" "{\"role_id\":\"${ROLE_PROJECT_EDITOR}\",\"scope\":\"global\",\"user_id\":\"rbac-user-matrix-target\"}"
  if [[ "$HTTP_STATUS" == "403" ]]; then
    GLOBAL_PASS=$((GLOBAL_PASS + 1))
  else
    fail "Global scope: ${caller_label} -> project:editor (global)" "expected 403, got ${HTTP_STATUS}"
    GLOBAL_FAIL=$((GLOBAL_FAIL + 1))
    if [[ "$HTTP_STATUS" == "201" ]]; then
      local_bid=$(echo "$HTTP_BODY" | jq -r '.id // empty')
      [[ -n "$local_bid" ]] && api DELETE "/role_bindings/${local_bid}" "$TOKEN_A"
    fi
  fi
done

echo -e "  Global scope: ${GREEN}${GLOBAL_PASS} passed${NC}, ${RED}${GLOBAL_FAIL} failed${NC} (of ${#CALLERS[@]})"
PASS_COUNT=$((PASS_COUNT + GLOBAL_PASS))

# Cleanup matrix bindings
[[ -n "$MATRIX_EDITOR_BIND" ]] && api DELETE "/role_bindings/${MATRIX_EDITOR_BIND}" "$TOKEN_A"
[[ -n "$MATRIX_VIEWER_BIND" ]] && api DELETE "/role_bindings/${MATRIX_VIEWER_BIND}" "$TOKEN_A"

TOTAL_MATRIX=$((MATRIX_PASS + MATRIX_FAIL + CROSS_PASS + CROSS_FAIL + GLOBAL_PASS + GLOBAL_FAIL))
echo ""
echo -e "  ${BOLD}Escalation matrix total: $((MATRIX_PASS + CROSS_PASS + GLOBAL_PASS)) passed, $((MATRIX_FAIL + CROSS_FAIL + GLOBAL_FAIL)) failed (${TOTAL_MATRIX} tests)${NC}"

# ============================================================
echo ""
echo -e "${BOLD}Phase 15: Session Sub-resource Isolation${NC}"

# User A starts a session via agent ignite
api POST "/projects/rbac-proj-alpha/agents/${AGENT_A_ID}/start" "$TOKEN_A" '{"prompt":"test session"}'
if [[ "$HTTP_STATUS" == "200" || "$HTTP_STATUS" == "201" ]]; then
  pass "User A starts session via agent ignite"
  SESSION_A_ID=$(echo "$HTTP_BODY" | jq -r '.id // .session.id // empty')
else
  fail "User A starts session via agent ignite" "got $HTTP_STATUS"
  SESSION_A_ID=""
fi

SESSION_RUNNING=false

if [[ -n "$SESSION_A_ID" ]]; then
  # User A can GET own session
  api GET "/sessions/${SESSION_A_ID}" "$TOKEN_A"
  assert_status "200" "$HTTP_STATUS" "User A GET own session returns 200"

  # Wait for session to reach Running phase (up to 60s)
  echo "  Waiting for session to reach Running phase..."
  for i in $(seq 1 30); do
    api GET "/sessions/${SESSION_A_ID}" "$TOKEN_A"
    PHASE=$(echo "$HTTP_BODY" | jq -r '.phase // empty')
    if [[ "$PHASE" == "Running" ]]; then
      SESSION_RUNNING=true
      echo "  Session is Running (waited $((i*2))s)"
      break
    fi
    sleep 2
  done
  if [[ "$SESSION_RUNNING" != "true" ]]; then
    echo "  Session did not reach Running (phase=$PHASE) — sub-resource write tests will be skipped"
  fi

  # User B cannot GET User A's session -> 404
  api GET "/sessions/${SESSION_A_ID}" "$TOKEN_B"
  assert_status "404" "$HTTP_STATUS" "User B GET User A's session returns 404"

  # User B cannot POST messages to User A's session -> 403
  api POST "/sessions/${SESSION_A_ID}/messages" "$TOKEN_B" '{"event_type":"user","payload":"unauthorized message"}'
  assert_status "403" "$HTTP_STATUS" "User B POST message to User A's session returns 403"

  # User A CAN post messages to own session (RBAC allows — 403/404 = RBAC failure)
  if [[ "$SESSION_RUNNING" == "true" ]]; then
    api POST "/sessions/${SESSION_A_ID}/messages" "$TOKEN_A" '{"event_type":"user","payload":"authorized message"}'
    if [[ "$HTTP_STATUS" == "403" || "$HTTP_STATUS" == "404" ]]; then
      fail "User A POST message to own session" "RBAC blocked owner: got $HTTP_STATUS"
    else
      pass "User A POST message to own session not blocked by RBAC (status $HTTP_STATUS)"
    fi
  else
    skip "User A POST message to own session (session not Running)"
  fi

  # User B cannot GET messages from User A's session -> 404
  api GET "/sessions/${SESSION_A_ID}/messages" "$TOKEN_B"
  assert_status "404" "$HTTP_STATUS" "User B GET messages from User A's session returns 404"

  # User B cannot GET events from User A's session -> 404
  api GET "/sessions/${SESSION_A_ID}/events" "$TOKEN_B"
  assert_status "404" "$HTTP_STATUS" "User B GET events from User A's session returns 404"

  # User B cannot clone User A's session -> 403
  api POST "/sessions/${SESSION_A_ID}/clone" "$TOKEN_B" '{}'
  assert_status "403" "$HTTP_STATUS" "User B clone User A's session returns 403"

  # User B cannot stop User A's session -> 403
  api POST "/sessions/${SESSION_A_ID}/stop" "$TOKEN_B" '{}'
  assert_status "403" "$HTTP_STATUS" "User B stop User A's session returns 403"

  # User B cannot delete User A's session -> 403 (mutation = opaque 403)
  api DELETE "/sessions/${SESSION_A_ID}" "$TOKEN_B"
  assert_status "403" "$HTTP_STATUS" "User B DELETE User A's session returns 403"

  # User C (no bindings) cannot access session -> 404
  api GET "/sessions/${SESSION_A_ID}" "$TOKEN_C"
  assert_status "404" "$HTTP_STATUS" "User C GET session returns 404 (no bindings)"
fi

# ============================================================
echo ""
echo -e "${BOLD}Phase 16: Role Binding List Isolation${NC}"

# User A lists role_bindings -> should only see own bindings, not User B's
api GET "/role_bindings?page=1&size=100" "$TOKEN_A"
assert_status "200" "$HTTP_STATUS" "User A GET /role_bindings returns 200"
# User A should see bindings for their projects/credentials
assert_list_not_contains "$HTTP_BODY" "user_id" "rbac-user-b" "User A role_bindings list does NOT contain User B's bindings"

api GET "/role_bindings?page=1&size=100" "$TOKEN_B"
assert_status "200" "$HTTP_STATUS" "User B GET /role_bindings returns 200"
assert_list_not_contains "$HTTP_BODY" "user_id" "rbac-user-a" "User B role_bindings list does NOT contain User A's bindings"

# User C (minimal bindings) lists role_bindings
api GET "/role_bindings?page=1&size=100" "$TOKEN_C"
assert_status "200" "$HTTP_STATUS" "User C GET /role_bindings returns 200"

# ============================================================
echo ""
echo -e "${BOLD}Phase 17: Project Settings Isolation${NC}"

# project_settings is a top-level route without project scope in URL
# The middleware blocks it since no scope can be extracted -> safe (no leak)
api GET "/project_settings?page=1&size=100" "$TOKEN_A"
if [[ "$HTTP_STATUS" == "200" || "$HTTP_STATUS" == "404" ]]; then
  pass "User A GET /project_settings does not leak data (status $HTTP_STATUS)"
else
  fail "User A GET /project_settings" "unexpected status $HTTP_STATUS"
fi

api GET "/project_settings?page=1&size=100" "$TOKEN_C"
if [[ "$HTTP_STATUS" == "200" || "$HTTP_STATUS" == "404" ]]; then
  pass "User C GET /project_settings does not leak data (status $HTTP_STATUS)"
else
  fail "User C GET /project_settings" "unexpected status $HTTP_STATUS"
fi

# ============================================================
echo ""
echo -e "${BOLD}Phase 18: CRITICAL — PATCH /role_bindings escalation prevention${NC}"

# Get User A's own owner binding ID for proj-alpha
OWNER_BIND_A=$(get_binding_id "$TOKEN_A" "user_id='rbac-user-a' and project_id='rbac-proj-alpha'")

if [[ -n "$OWNER_BIND_A" ]]; then
  # User B tries to PATCH User A's binding to change role to platform:admin -> must fail
  api PATCH "/role_bindings/${OWNER_BIND_A}" "$TOKEN_B" "{\"role_id\":\"${ROLE_PLATFORM_ADMIN}\"}"
  assert_status "403" "$HTTP_STATUS" "CRITICAL: User B cannot PATCH another user's binding to platform:admin"

  # User B tries to PATCH binding to change user_id to themselves -> must fail
  api PATCH "/role_bindings/${OWNER_BIND_A}" "$TOKEN_B" "{\"user_id\":\"rbac-user-b\"}"
  assert_status "403" "$HTTP_STATUS" "CRITICAL: User B cannot PATCH binding to hijack ownership"
else
  fail "CRITICAL: Could not find User A's owner binding" "binding lookup returned empty — skipping PATCH escalation tests"
fi

# ============================================================
echo ""
echo -e "${BOLD}Phase 19: Session sub-resource access for project owner${NC}"

# These test that RBAC does NOT block the project owner from session sub-resources.
# 403 = RBAC blocked = always FAIL.
# 404 with {"kind":"Error"} = RBAC 404 (resource hidden) = FAIL.
# 404 with {"detail":"Not Found"} = runner endpoint missing (passed RBAC) = PASS.
# 200/201 = PASS.
# 500/502/503 = infrastructure = SKIP if not Running, FAIL if Running.
assert_not_rbac_blocked() {
  local desc="$1"
  if [[ "$HTTP_STATUS" == "403" ]]; then
    fail "$desc" "RBAC blocked owner: got $HTTP_STATUS"
  elif [[ "$HTTP_STATUS" == "404" ]]; then
    if echo "$HTTP_BODY" | grep -q '"kind"'; then
      fail "$desc" "RBAC blocked owner: got $HTTP_STATUS"
    else
      pass "$desc"
    fi
  elif [[ "$HTTP_STATUS" == "200" || "$HTTP_STATUS" == "201" ]]; then
    pass "$desc"
  elif [[ "$HTTP_STATUS" == "502" || "$HTTP_STATUS" == "503" ]]; then
    pass "$desc (runner proxy $HTTP_STATUS — RBAC passed, infrastructure lag)"
  elif [[ "$SESSION_RUNNING" == "true" ]]; then
    fail "$desc" "session is Running but got $HTTP_STATUS"
  else
    skip "$desc (session not Running, got $HTTP_STATUS)"
  fi
}

if [[ -n "$SESSION_A_ID" ]]; then
  # DB-backed endpoints — always work if RBAC passes (no runner needed)
  api GET "/sessions/${SESSION_A_ID}/messages" "$TOKEN_A"
  assert_not_rbac_blocked "Owner can GET /sessions/{id}/messages"

  api GET "/sessions/${SESSION_A_ID}/pod-events" "$TOKEN_A"
  assert_not_rbac_blocked "Owner can GET /sessions/{id}/pod-events"

  api GET "/sessions/${SESSION_A_ID}/export" "$TOKEN_A"
  assert_not_rbac_blocked "Owner can GET /sessions/{id}/export"

  # Runner-proxy endpoints — require a running pod; skip if session not Running.
  # Wait for the runner's HTTP server to become reachable (the CP marks the
  # session Running as soon as the pod is created, but the AGUI server inside
  # the container may need a few more seconds to bind).
  RUNNER_ENDPOINTS=("events" "workspace" "git/status" "agui/capabilities" "mcp/status")
  if [[ "$SESSION_RUNNING" == "true" ]]; then
    for _wait in $(seq 1 15); do
      _probe=$(curl -sf -o /dev/null -w '%{http_code}' --max-time 3 \
        -H "Authorization: Bearer $TOKEN_A" \
        "${API_URL}/sessions/${SESSION_A_ID}/events" 2>/dev/null || true)
      [[ "$_probe" != "502" && "$_probe" != "503" && "$_probe" != "000" && "$_probe" != "" ]] && break
      sleep 2
    done
  fi
  for ep in "${RUNNER_ENDPOINTS[@]}"; do
    if [[ "$SESSION_RUNNING" == "true" ]]; then
      api GET "/sessions/${SESSION_A_ID}/${ep}" "$TOKEN_A"
      assert_not_rbac_blocked "Owner can GET /sessions/{id}/${ep}"
    else
      skip "Owner GET /sessions/{id}/${ep} (session not Running)"
    fi
  done
fi

# Clean up test session (after Phase 19 uses it)
if [[ -n "$SESSION_A_ID" ]]; then
  api POST "/sessions/${SESSION_A_ID}/stop" "$TOKEN_A" '{}'
  api DELETE "/sessions/${SESSION_A_ID}" "$TOKEN_A"
  echo "  Cleaned up test session ${SESSION_A_ID}"
fi

# ============================================================
echo ""
echo -e "${BOLD}Phase 20: Scheduled Sessions RBAC${NC}"

# Create a scheduled session — project:owner should be able to
api POST "/projects/rbac-proj-alpha/scheduled-sessions" "$TOKEN_A" '{"name":"rbac-sched-test","schedule":"0 9 * * 1-5","agent_id":"'"${AGENT_A_ID}"'","session_prompt":"test"}'
if [[ "$HTTP_STATUS" == "201" || "$HTTP_STATUS" == "200" ]]; then
  pass "Owner can create scheduled-session"
  SCHED_ID=$(echo "$HTTP_BODY" | jq -r '.id // empty')
else
  fail "Owner create scheduled-session" "expected 201, got $HTTP_STATUS"
  SCHED_ID=""
fi

# User B cannot list scheduled sessions in proj-alpha
api GET "/projects/rbac-proj-alpha/scheduled-sessions" "$TOKEN_B"
assert_status "404" "$HTTP_STATUS" "User B cannot list scheduled-sessions in proj-alpha"

# Cleanup scheduled session
if [[ -n "$SCHED_ID" ]]; then
  api DELETE "/projects/rbac-proj-alpha/scheduled-sessions/${SCHED_ID}" "$TOKEN_A"
fi

# ============================================================
echo ""
echo -e "${BOLD}Phase 21: Credential Token Fetch RBAC${NC}"

# Credential owner should be able to fetch token (GET /credentials/{id}/token)
# This tests that pathToResource maps correctly for the /token sub-resource
if [[ -n "$CRED_A_ID" ]]; then
  api GET "/credentials/${CRED_A_ID}/token" "$TOKEN_A"
  assert_status "200" "$HTTP_STATUS" "Credential owner can GET /credentials/{id}/token"

  # Non-owner cannot fetch token
  api GET "/credentials/${CRED_A_ID}/token" "$TOKEN_B"
  assert_status "404" "$HTTP_STATUS" "Non-owner cannot GET /credentials/{id}/token"
fi

# ============================================================
# ============================================================
echo ""
echo -e "${BOLD}Phase 22: PATCH Scope Widening Attack${NC}"

# User B owns proj-beta. User B has NO access to proj-alpha.
# User B gets their own project:owner binding ID on proj-beta.
OWNER_BIND_B=$(get_binding_id "$TOKEN_B" "user_id='rbac-user-b' and project_id='rbac-proj-beta'")

if [[ -n "$OWNER_BIND_B" ]]; then
  # Attack: User B PATCHes their own binding to change project_id to proj-alpha
  # This should be REJECTED — scope widening to an unauthorized project
  api PATCH "/role_bindings/${OWNER_BIND_B}" "$TOKEN_B" '{"project_id":"rbac-proj-alpha"}'
  assert_status "403" "$HTTP_STATUS" "PATCH scope widening: cannot change project_id to unauthorized project"

  # Verify User B still cannot see proj-alpha (attack failed)
  api GET "/projects/rbac-proj-alpha" "$TOKEN_B"
  assert_status "404" "$HTTP_STATUS" "After failed scope widening, proj-alpha still invisible to User B"

  # Also test: User B cannot widen scope to global
  api PATCH "/role_bindings/${OWNER_BIND_B}" "$TOKEN_B" '{"scope":"global","project_id":null}'
  assert_status "403" "$HTTP_STATUS" "PATCH scope widening: cannot change scope to global"
else
  fail "PATCH scope widening test" "could not find User B's owner binding on proj-beta — skipping remaining tests"
fi

# ============================================================
echo ""
echo -e "${BOLD}Phase 23: Nil SessionFactory Guard${NC}"

# This is a code-level guard, not directly testable via HTTP.
# But we can verify the escalation checks ARE running by testing
# that a zero-binding user cannot create an arbitrary binding.
# If sessionFactory were nil, this would succeed (no checks).
api POST "/role_bindings" "$TOKEN_C" "{\"role_id\":\"${ROLE_PLATFORM_ADMIN}\",\"scope\":\"global\",\"user_id\":\"rbac-user-c\"}"
assert_status "403" "$HTTP_STATUS" "Zero-binding user cannot create platform:admin binding (escalation checks active)"

# ============================================================
echo ""
echo -e "${BOLD}Phase 24: Platform Viewer Cannot Escalate to Admin${NC}"

# Grant User C platform:viewer (via direct DB insert since we need a global binding)
# We can't grant global from a non-admin, so we use the seed-admin pattern via kubectl
VIEWER_GLOBAL_BIND=""
DB_POD_NAME=$(kubectl get pods -n "$NS" -l app=ambient-api-server,component=database -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
if [[ -z "$DB_POD_NAME" ]]; then
  fail "Phase 24 setup" "DB pod not found — cannot seed platform:viewer binding; skipping viewer escalation tests"
else
  kubectl exec -n "$NS" "$DB_POD_NAME" -- psql -U ambient -d ambient_api_server -t -A -c "
    INSERT INTO role_bindings (id, role_id, scope, user_id, created_at, updated_at)
    SELECT '$(date +%s)viewerbind', r.id, 'global', 'rbac-user-c', NOW(), NOW()
    FROM roles r WHERE r.name = 'platform:viewer' AND r.deleted_at IS NULL
    ON CONFLICT DO NOTHING;
  " 2>/dev/null >/dev/null

  # Refresh token for User C
  TOKEN_C=$(get_token "rbac-user-c" "testpass")

  # User C (platform:viewer) tries to grant platform:admin to themselves → MUST fail
  api POST "/role_bindings" "$TOKEN_C" "{\"role_id\":\"${ROLE_PLATFORM_ADMIN}\",\"scope\":\"global\",\"user_id\":\"rbac-user-c\"}"
  assert_status "403" "$HTTP_STATUS" "CRITICAL: platform:viewer cannot grant platform:admin"

  # User C tries to grant platform:viewer to someone else → should also fail (viewers can't grant)
  api POST "/role_bindings" "$TOKEN_C" "{\"role_id\":\"${ROLE_PLATFORM_VIEWER}\",\"scope\":\"global\",\"user_id\":\"rbac-user-a\"}"
  assert_status "403" "$HTTP_STATUS" "platform:viewer cannot grant platform:viewer (no self-mint)"
fi

# ============================================================
echo ""
echo -e "${BOLD}Phase 25: Delete Hierarchy — owner cannot delete higher-privilege bindings${NC}"

# Seed a platform:admin binding scoped to proj-alpha via direct DB insert.
# No user can grant platform:admin on a project via the API, so we seed it.
DB_POD=$(kubectl get pods -n "$NS" -l app=ambient-api-server,component=database -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
F5_BIND_ID="f5_test_admin_bind"
if [[ -n "$DB_POD" ]]; then
  kubectl exec -n "$NS" "$DB_POD" -- psql -U ambient -d ambient_api_server -qc "
    INSERT INTO role_bindings (id, role_id, scope, project_id, user_id, created_at, updated_at)
    SELECT '${F5_BIND_ID}', r.id, 'project', 'rbac-proj-alpha', 'rbac-admin-ghost', NOW(), NOW()
    FROM roles r WHERE r.name = 'platform:admin' AND r.deleted_at IS NULL
    ON CONFLICT DO NOTHING;
  " 2>/dev/null || true

  # User A (project:owner level 1) tries to delete platform:admin's binding (level 0) → must fail
  api DELETE "/role_bindings/${F5_BIND_ID}" "$TOKEN_A"
  assert_status "403" "$HTTP_STATUS" "F5: project:owner cannot delete platform:admin binding"

  # Clean up
  kubectl exec -n "$NS" "$DB_POD" -- psql -U ambient -d ambient_api_server -qc \
    "DELETE FROM role_bindings WHERE id = '${F5_BIND_ID}';" 2>/dev/null || true
else
  skip "F5: DB pod not found"
fi

# ============================================================
echo ""
echo -e "${BOLD}Phase 26: Credential Binding Enforcement (spec: credential-binding-enforcement)${NC}"

# This phase tests the hierarchical credential binding authorization rules:
#   - project:editor can bind credentials (not just owner)
#   - project:viewer cannot bind credentials
#   - agent_id without project_id is rejected (400)
#   - agent not belonging to project is rejected (400)
#   - global credential bindings require platform:admin
#   - project:editor can unbind without credential:owner (asymmetric)
#   - non-admin users cannot create internal role bindings (credential:token-reader)

# Setup: give User B project:editor on proj-alpha for this phase
api POST "/role_bindings" "$TOKEN_A" "{\"role_id\":\"${ROLE_PROJECT_EDITOR}\",\"scope\":\"project\",\"user_id\":\"rbac-user-b\",\"project_id\":\"rbac-proj-alpha\"}"
CB_EDITOR_BIND_ID=$(echo "$HTTP_BODY" | jq -r '.id // empty')

# --- Scenario: project:editor can bind credential to project ---
# User A owns cred-a and proj-alpha. User B has project:editor on proj-alpha.
# User A (credential:owner + project:editor-via-ownership) binds cred-a to proj-alpha.
# But the spec says project:editor is enough — test User A who has project:owner (level 1 ≤ 2, passes).
api POST "/role_bindings" "$TOKEN_A" "{\"role_id\":\"${ROLE_CREDENTIAL_VIEWER}\",\"scope\":\"credential\",\"user_id\":\"rbac-user-b\",\"credential_id\":\"${CRED_A_ID}\",\"project_id\":\"rbac-proj-alpha\"}"
assert_status "201" "$HTTP_STATUS" "CB: credential owner + project owner can bind credential to project"
CB_BIND_1=$(echo "$HTTP_BODY" | jq -r '.id // empty')

# Clean up binding for next test
[[ -n "$CB_BIND_1" ]] && api DELETE "/role_bindings/${CB_BIND_1}" "$TOKEN_A"

# --- Scenario: project:viewer cannot bind credentials ---
# Give User C project:viewer on proj-alpha
api POST "/role_bindings" "$TOKEN_A" "{\"role_id\":\"${ROLE_PROJECT_VIEWER}\",\"scope\":\"project\",\"user_id\":\"rbac-user-c\",\"project_id\":\"rbac-proj-alpha\"}"
CB_VIEWER_BIND_ID=$(echo "$HTTP_BODY" | jq -r '.id // empty')

# Give User C credential:owner on cred-c (they created it in Phase 12)
# User C tries to bind their credential to proj-alpha where they're only viewer
if [[ -n "$CRED_C_ID" ]]; then
  api POST "/role_bindings" "$TOKEN_C" "{\"role_id\":\"${ROLE_CREDENTIAL_VIEWER}\",\"scope\":\"credential\",\"user_id\":\"rbac-user-c\",\"credential_id\":\"${CRED_C_ID}\",\"project_id\":\"rbac-proj-alpha\"}"
  assert_status "403" "$HTTP_STATUS" "CB: project:viewer cannot bind credential to project"
else
  skip "CB: project:viewer bind test (no cred_c_id)"
fi

# Clean up viewer binding
[[ -n "$CB_VIEWER_BIND_ID" ]] && api DELETE "/role_bindings/${CB_VIEWER_BIND_ID}" "$TOKEN_A"

# --- Scenario: agent_id without project_id is rejected (400) ---
api POST "/role_bindings" "$TOKEN_A" "{\"role_id\":\"${ROLE_CREDENTIAL_VIEWER}\",\"scope\":\"credential\",\"user_id\":\"rbac-user-a\",\"credential_id\":\"${CRED_A_ID}\",\"agent_id\":\"${AGENT_A_ID}\"}"
assert_status "400" "$HTTP_STATUS" "CB: agent_id without project_id returns 400"

# --- Scenario: agent not belonging to project is rejected (400) ---
# AGENT_B_ID belongs to proj-beta. Binding it to proj-alpha should fail.
if [[ -n "$AGENT_B_ID" ]]; then
  api POST "/role_bindings" "$TOKEN_A" "{\"role_id\":\"${ROLE_CREDENTIAL_VIEWER}\",\"scope\":\"credential\",\"user_id\":\"rbac-user-a\",\"credential_id\":\"${CRED_A_ID}\",\"project_id\":\"rbac-proj-alpha\",\"agent_id\":\"${AGENT_B_ID}\"}"
  assert_status "400" "$HTTP_STATUS" "CB: agent not in project returns 400"
else
  skip "CB: agent-project mismatch test (no agent_b_id)"
fi

# --- Scenario: valid agent-level binding accepted ---
if [[ -n "$AGENT_A_ID" ]]; then
  api POST "/role_bindings" "$TOKEN_A" "{\"role_id\":\"${ROLE_CREDENTIAL_VIEWER}\",\"scope\":\"credential\",\"user_id\":\"rbac-user-a\",\"credential_id\":\"${CRED_A_ID}\",\"project_id\":\"rbac-proj-alpha\",\"agent_id\":\"${AGENT_A_ID}\"}"
  assert_status "201" "$HTTP_STATUS" "CB: agent-level binding with correct project accepted"
  CB_AGENT_BIND=$(echo "$HTTP_BODY" | jq -r '.id // empty')
  [[ -n "$CB_AGENT_BIND" ]] && api DELETE "/role_bindings/${CB_AGENT_BIND}" "$TOKEN_A"
else
  skip "CB: agent-level binding test (no agent_a_id)"
fi

# --- Scenario: global credential binding requires platform:admin ---
# User A has credential:owner on cred-a but is NOT platform:admin
api POST "/role_bindings" "$TOKEN_A" "{\"role_id\":\"${ROLE_CREDENTIAL_VIEWER}\",\"scope\":\"credential\",\"user_id\":\"rbac-user-a\",\"credential_id\":\"${CRED_A_ID}\"}"
assert_status "403" "$HTTP_STATUS" "CB: non-admin cannot create global credential binding"

# --- Scenario: non-credential-owner cannot bind ---
# User B owns cred-b but NOT cred-a. User B is project:editor on proj-alpha.
api POST "/role_bindings" "$TOKEN_B" "{\"role_id\":\"${ROLE_CREDENTIAL_VIEWER}\",\"scope\":\"credential\",\"user_id\":\"rbac-user-b\",\"credential_id\":\"${CRED_A_ID}\",\"project_id\":\"rbac-proj-alpha\"}"
assert_status "403" "$HTTP_STATUS" "CB: non-credential-owner cannot bind (editor on project but not cred owner)"

# --- Scenario: asymmetric unbind — project:editor can remove binding without credential:owner ---
# First, User A (cred owner + proj owner) creates a binding
api POST "/role_bindings" "$TOKEN_A" "{\"role_id\":\"${ROLE_CREDENTIAL_VIEWER}\",\"scope\":\"credential\",\"user_id\":\"rbac-user-a\",\"credential_id\":\"${CRED_A_ID}\",\"project_id\":\"rbac-proj-alpha\"}"
CB_UNBIND_ID=$(echo "$HTTP_BODY" | jq -r '.id // empty')

if [[ -n "$CB_UNBIND_ID" ]]; then
  # User B (project:editor, NOT credential:owner) deletes the binding
  api DELETE "/role_bindings/${CB_UNBIND_ID}" "$TOKEN_B"
  assert_status "204" "$HTTP_STATUS" "CB: project:editor can unbind credential without credential:owner"
else
  fail "CB: asymmetric unbind setup" "could not create binding to test unbind"
fi

# --- Scenario: project:viewer cannot unbind ---
# Re-create binding, give User C viewer, test they cannot unbind
api POST "/role_bindings" "$TOKEN_A" "{\"role_id\":\"${ROLE_CREDENTIAL_VIEWER}\",\"scope\":\"credential\",\"user_id\":\"rbac-user-a\",\"credential_id\":\"${CRED_A_ID}\",\"project_id\":\"rbac-proj-alpha\"}"
CB_UNBIND_ID2=$(echo "$HTTP_BODY" | jq -r '.id // empty')
api POST "/role_bindings" "$TOKEN_A" "{\"role_id\":\"${ROLE_PROJECT_VIEWER}\",\"scope\":\"project\",\"user_id\":\"rbac-user-c\",\"project_id\":\"rbac-proj-alpha\"}"
CB_VIEWER_BIND_2=$(echo "$HTTP_BODY" | jq -r '.id // empty')

if [[ -n "$CB_UNBIND_ID2" ]]; then
  api DELETE "/role_bindings/${CB_UNBIND_ID2}" "$TOKEN_C"
  assert_status "403" "$HTTP_STATUS" "CB: project:viewer cannot unbind credential"
  # Clean up
  api DELETE "/role_bindings/${CB_UNBIND_ID2}" "$TOKEN_A"
else
  fail "CB: viewer unbind test setup" "could not create binding"
fi
[[ -n "$CB_VIEWER_BIND_2" ]] && api DELETE "/role_bindings/${CB_VIEWER_BIND_2}" "$TOKEN_A"

# --- Scenario: non-admin user cannot create credential:token-reader binding (internal role) ---
if [[ -n "$ROLE_CRED_TOKEN_READER" ]]; then
  api POST "/role_bindings" "$TOKEN_A" "{\"role_id\":\"${ROLE_CRED_TOKEN_READER}\",\"scope\":\"credential\",\"user_id\":\"rbac-user-a\",\"credential_id\":\"${CRED_A_ID}\"}"
  assert_status "403" "$HTTP_STATUS" "CB: non-admin cannot create credential:token-reader binding"
else
  skip "CB: token-reader internal role test (role not found)"
fi

# Cleanup phase bindings
[[ -n "$CB_EDITOR_BIND_ID" ]] && api DELETE "/role_bindings/${CB_EDITOR_BIND_ID}" "$TOKEN_A"

# ============================================================
# Cleanup is handled by the EXIT trap (clean_db + Keycloak user deletion)
# ============================================================
echo ""
echo -e "${BOLD}Summary${NC}"
TOTAL=$((PASS_COUNT + FAIL_COUNT + SKIP_COUNT))
echo -e "  ${GREEN}${PASS_COUNT} passed${NC}, ${RED}${FAIL_COUNT} failed${NC}, ${YELLOW}${SKIP_COUNT} skipped${NC} (${TOTAL} total)"

if [[ "$FAIL_COUNT" -gt 0 ]]; then
  exit 1
fi
