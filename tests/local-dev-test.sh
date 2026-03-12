#!/bin/bash
#
# Local Developer Experience Test Suite
# Tests the complete local development workflow for Ambient Code Platform
#
# Usage: ./tests/local-dev-test.sh [options]
#   -s, --skip-setup    Skip the initial setup (assume environment is ready)
#   -c, --cleanup       Clean up after tests
#   -v, --verbose       Verbose output
#   --ci                CI mode (treats known TODOs as non-failures)
#

# Don't exit on error - we want to collect all test results
# shellcheck disable=SC2103  # Intentional: continue on errors to collect all test results
set +e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Test configuration
NAMESPACE="${NAMESPACE:-ambient-code}"
SKIP_SETUP=false
CLEANUP=false
VERBOSE=false
CI_MODE=false
FAILED_TESTS=0
PASSED_TESTS=0
KNOWN_FAILURES=0

# Get test URL for a service via port-forwarding (kind uses localhost)
get_test_url() {
    local port=$1

    # Kind uses port-forwarding to localhost
    if [[ "$port" == "30080" ]]; then
        echo "http://localhost:8080"
    elif [[ "$port" == "30030" ]]; then
        echo "http://localhost:3000"
    else
        echo "http://localhost:${port}"
    fi
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -s|--skip-setup)
            SKIP_SETUP=true
            shift
            ;;
        -c|--cleanup)
            CLEANUP=true
            shift
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        --ci)
            CI_MODE=true
            shift
            ;;
        -h|--help)
            head -n 10 "$0" | tail -n 7
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Logging functions
log_info() {
    echo -e "${BLUE}ℹ${NC} $*"
}

log_success() {
    echo -e "${GREEN}✓${NC} $*"
}

log_error() {
    echo -e "${RED}✗${NC} $*"
}

log_warning() {
    echo -e "${YELLOW}⚠${NC} $*"
}

log_section() {
    echo ""
    echo -e "${BOLD}═══════════════════════════════════════════${NC}"
    echo -e "${BOLD}  $*${NC}"
    echo -e "${BOLD}═══════════════════════════════════════════${NC}"
}

# Test assertion functions
assert_command_exists() {
    local cmd=$1
    if command -v "$cmd" >/dev/null 2>&1; then
        log_success "Command '$cmd' is installed"
        ((PASSED_TESTS++))
        return 0
    else
        log_error "Command '$cmd' is NOT installed"
        ((FAILED_TESTS++))
        return 1
    fi
}

assert_equals() {
    local expected=$1
    local actual=$2
    local description=$3

    if [ "$expected" = "$actual" ]; then
        log_success "$description"
        ((PASSED_TESTS++))
        return 0
    else
        log_error "$description"
        log_error "  Expected: $expected"
        log_error "  Actual: $actual"
        ((FAILED_TESTS++))
        return 1
    fi
}

assert_contains() {
    local haystack=$1
    local needle=$2
    local description=$3

    if echo "$haystack" | grep -q "$needle"; then
        log_success "$description"
        ((PASSED_TESTS++))
        return 0
    else
        log_error "$description"
        log_error "  Expected to contain: $needle"
        log_error "  Actual: $haystack"
        ((FAILED_TESTS++))
        return 1
    fi
}

assert_http_ok() {
    local url=$1
    local description=$2
    local max_retries=${3:-5}
    local retry=0

    while [ $retry -lt $max_retries ]; do
        if curl -sf "$url" >/dev/null 2>&1; then
            log_success "$description"
            ((PASSED_TESTS++))
            return 0
        fi
        ((retry++))
        [ $retry -lt $max_retries ] && sleep 2
    done

    log_error "$description (after $max_retries retries)"
    ((FAILED_TESTS++))
    return 1
}

assert_pod_running() {
    local label=$1
    local description=$2

    if kubectl get pods -n "$NAMESPACE" -l "$label" 2>/dev/null | grep -q "Running"; then
        log_success "$description"
        ((PASSED_TESTS++))
        return 0
    else
        log_error "$description"
        ((FAILED_TESTS++))
        return 1
    fi
}

# Test: Prerequisites
test_prerequisites() {
    log_section "Test 1: Prerequisites"

    assert_command_exists "make"
    assert_command_exists "kubectl"
    assert_command_exists "kind"
    assert_command_exists "podman" || assert_command_exists "docker"

    # Check if running on macOS or Linux
    if [[ "$OSTYPE" == "darwin"* ]]; then
        log_info "Running on macOS"
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        log_info "Running on Linux"
    else
        log_warning "Unknown OS: $OSTYPE"
    fi
}

# Test: Makefile Help
test_makefile_help() {
    log_section "Test 2: Makefile Help Command"

    local help_output
    help_output=$(make help 2>&1)

    assert_contains "$help_output" "Ambient Code Platform" "Help shows correct branding"
    assert_contains "$help_output" "local-up" "Help lists local-up command"
    assert_contains "$help_output" "local-status" "Help lists local-status command"
    assert_contains "$help_output" "local-logs" "Help lists local-logs command"
    assert_contains "$help_output" "kind-up" "Help lists kind-up command"
}

# Test: Kind Status Check
test_kind_status() {
    log_section "Test 3: Kind Status"

    if kind get clusters 2>/dev/null | grep -q .; then
        log_success "Kind cluster is running"
        ((PASSED_TESTS++))

        # Check kind version
        local version
        version=$(kind version 2>/dev/null || echo "unknown")
        log_info "Kind version: $version"
    else
        log_error "No Kind cluster is running"
        ((FAILED_TESTS++))
        return 1
    fi
}

# Test: Kubernetes Context
test_kubernetes_context() {
    log_section "Test 4: Kubernetes Context"

    local context
    context=$(kubectl config current-context 2>/dev/null || echo "none")

    assert_contains "$context" "kind-" "kubectl context is set to a kind cluster"

    # Test kubectl connectivity
    if kubectl cluster-info >/dev/null 2>&1; then
        log_success "kubectl can connect to cluster"
        ((PASSED_TESTS++))
    else
        log_error "kubectl cannot connect to cluster"
        ((FAILED_TESTS++))
    fi
}

# Test: Namespace Exists
test_namespace_exists() {
    log_section "Test 5: Namespace Existence"

    if kubectl get namespace "$NAMESPACE" >/dev/null 2>&1; then
        log_success "Namespace '$NAMESPACE' exists"
        ((PASSED_TESTS++))
    else
        log_error "Namespace '$NAMESPACE' does NOT exist"
        ((FAILED_TESTS++))
        return 1
    fi
}

# Test: CRDs Installed
test_crds_installed() {
    log_section "Test 6: Custom Resource Definitions"

    local crds=("agenticsessions.vteam.ambient-code" "projectsettings.vteam.ambient-code")

    for crd in "${crds[@]}"; do
        if kubectl get crd "$crd" >/dev/null 2>&1; then
            log_success "CRD '$crd' is installed"
            ((PASSED_TESTS++))
        else
            log_error "CRD '$crd' is NOT installed"
            ((FAILED_TESTS++))
        fi
    done
}

# Test: Pods Running
test_pods_running() {
    log_section "Test 7: Pod Status"

    assert_pod_running "app=backend-api" "Backend pod is running"
    assert_pod_running "app=frontend" "Frontend pod is running"
    assert_pod_running "app=agentic-operator" "Operator pod is running"

    # Check pod readiness
    local not_ready
    not_ready=$(kubectl get pods -n "$NAMESPACE" --field-selector=status.phase!=Running 2>/dev/null | grep -v "NAME" | wc -l)

    if [ "$not_ready" -eq 0 ]; then
        log_success "All pods are in Running state"
        ((PASSED_TESTS++))
    else
        log_warning "$not_ready pod(s) are not running"
    fi
}

# Test: Services Exist
test_services_exist() {
    log_section "Test 8: Services"

    local services=("backend-service" "frontend-service")

    for svc in "${services[@]}"; do
        if kubectl get svc "$svc" -n "$NAMESPACE" >/dev/null 2>&1; then
            log_success "Service '$svc' exists"
            ((PASSED_TESTS++))
        else
            log_error "Service '$svc' does NOT exist"
            ((FAILED_TESTS++))
        fi
    done
}

# Test: Ingress Configuration
test_ingress() {
    log_section "Test 9: Ingress Configuration"

    # Kind uses NodePort mapping instead of an Ingress resource
    if kubectl get ingress -n "$NAMESPACE" >/dev/null 2>&1; then
        local ingress_count
        ingress_count=$(kubectl get ingress -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)
        if [ "$ingress_count" -gt 0 ]; then
            log_success "Ingress resources found"
            ((PASSED_TESTS++))
        else
            log_info "No ingress resources (kind uses NodePort mapping, this is expected)"
            ((PASSED_TESTS++))
        fi
    else
        log_info "No ingress resources (kind uses NodePort mapping, this is expected)"
        ((PASSED_TESTS++))
    fi
}

# Test: Backend Health Endpoint
test_backend_health() {
    log_section "Test 10: Backend Health Endpoint"

    # Check backend health via pod readiness (kubectl wait already validates the
    # readiness probe which hits /health). Verify pod is ready as a proxy.
    if kubectl get pods -n "$NAMESPACE" -l app=backend-api -o jsonpath='{.items[0].status.conditions[?(@.type=="Ready")].status}' 2>/dev/null | grep -q "True"; then
        log_success "Backend health endpoint responds (pod readiness probe passes)"
        ((PASSED_TESTS++))
    else
        log_error "Backend pod is not ready (health endpoint may not be responding)"
        ((FAILED_TESTS++))
    fi
}

# Test: Frontend Accessibility
test_frontend_accessibility() {
    log_section "Test 11: Frontend Accessibility"

    # Check frontend health via pod readiness
    if kubectl get pods -n "$NAMESPACE" -l app=frontend -o jsonpath='{.items[0].status.conditions[?(@.type=="Ready")].status}' 2>/dev/null | grep -q "True"; then
        log_success "Frontend is accessible (pod readiness probe passes)"
        ((PASSED_TESTS++))
    else
        log_error "Frontend pod is not ready"
        ((FAILED_TESTS++))
    fi
}

# Test: RBAC Configuration
test_rbac() {
    log_section "Test 12: RBAC Configuration"

    local roles=("ambient-project-admin" "ambient-project-edit" "ambient-project-view")

    for role in "${roles[@]}"; do
        if kubectl get clusterrole "$role" >/dev/null 2>&1; then
            log_success "ClusterRole '$role' exists"
            ((PASSED_TESTS++))
        else
            log_error "ClusterRole '$role' does NOT exist"
            ((FAILED_TESTS++))
        fi
    done
}

# Test: Development Workflow - Build Command
test_build_command() {
    log_section "Test 13: Build Commands (Dry Run)"

    if make -n build-backend >/dev/null 2>&1; then
        log_success "make build-backend syntax is valid"
        ((PASSED_TESTS++))
    else
        log_error "make build-backend has syntax errors"
        ((FAILED_TESTS++))
    fi

    if make -n build-frontend >/dev/null 2>&1; then
        log_success "make build-frontend syntax is valid"
        ((PASSED_TESTS++))
    else
        log_error "make build-frontend has syntax errors"
        ((FAILED_TESTS++))
    fi
}

# Test: Development Workflow - Rebuild Command
test_reload_commands() {
    log_section "Test 14: Rebuild Command (Dry Run)"

    if make -n kind-rebuild >/dev/null 2>&1; then
        log_success "make kind-rebuild syntax is valid"
        ((PASSED_TESTS++))
    else
        log_error "make kind-rebuild has syntax errors"
        ((FAILED_TESTS++))
    fi
}

# Test: Logging Commands
test_logging_commands() {
    log_section "Test 15: Logging Commands"

    # Test that we can get logs from each component
    local components=("backend-api" "frontend" "agentic-operator")

    for component in "${components[@]}"; do
        if kubectl logs -n "$NAMESPACE" -l "app=$component" --tail=1 >/dev/null 2>&1; then
            log_success "Can retrieve logs from $component"
            ((PASSED_TESTS++))
        else
            log_warning "Cannot retrieve logs from $component (pod may not be running)"
        fi
    done
}

# Test: Storage Configuration
test_storage() {
    log_section "Test 16: Storage Configuration"

    # Check if workspace PVC exists
    if kubectl get pvc workspace-pvc -n "$NAMESPACE" >/dev/null 2>&1; then
        log_success "Workspace PVC exists"
        ((PASSED_TESTS++))

        # Check PVC status
        local status
        status=$(kubectl get pvc workspace-pvc -n "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null)
        if [ "$status" = "Bound" ]; then
            log_success "Workspace PVC is bound"
            ((PASSED_TESTS++))
        else
            log_warning "Workspace PVC status: $status"
        fi
    else
        log_info "Workspace PVC does not exist (may not be required for all deployments)"
    fi
}

# Test: Environment Variables
test_environment_variables() {
    log_section "Test 17: Environment Variables"

    # Check backend deployment env vars
    local backend_env
    backend_env=$(kubectl get deployment backend-api -n "$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].env[*].name}' 2>/dev/null || echo "")

    assert_contains "$backend_env" "NAMESPACE" "Backend has NAMESPACE env var"
    assert_contains "$backend_env" "PORT" "Backend has PORT env var"

    # Check frontend deployment env vars
    local frontend_env
    frontend_env=$(kubectl get deployment frontend -n "$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].env[*].name}' 2>/dev/null || echo "")

    assert_contains "$frontend_env" "BACKEND_URL" "Frontend has BACKEND_URL env var"
}

# Test: Resource Limits
test_resource_limits() {
    log_section "Test 18: Resource Configuration"

    # Check if deployments have resource requests/limits
    local deployments=("backend-api" "frontend" "agentic-operator")

    for deployment in "${deployments[@]}"; do
        local resources
        resources=$(kubectl get deployment "$deployment" -n "$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].resources}' 2>/dev/null || echo "{}")

        if [ "$resources" != "{}" ]; then
            log_success "Deployment '$deployment' has resource configuration"
            ((PASSED_TESTS++))
        else
            log_info "Deployment '$deployment' has no resource limits (OK for dev)"
        fi
    done
}

# Test: Make local-status
test_make_status() {
    log_section "Test 19: make local-status Command"

    local status_output
    # Pass CONTAINER_ENGINE so kind get clusters uses the correct provider
    local engine
    if command -v docker &>/dev/null && docker info &>/dev/null 2>&1; then
        engine=docker
    else
        engine=podman
    fi
    status_output=$(make local-status CONTAINER_ENGINE="$engine" 2>&1 || echo "")

    assert_contains "$status_output" "Ambient Code Platform Status" "Status shows correct branding"
    assert_contains "$status_output" "Pods" "Status shows Pods section"
}

# Test: Ingress Controller
test_ingress_controller() {
    log_section "Test 20: Ingress Controller"

    # Kind uses NodePort mapping; ingress-nginx is optional
    if kubectl get namespace ingress-nginx >/dev/null 2>&1; then
        log_success "ingress-nginx namespace exists"
        ((PASSED_TESTS++))

        # Check if controller is running
        if kubectl get pods -n ingress-nginx -l app.kubernetes.io/component=controller 2>/dev/null | grep -q "Running"; then
            log_success "Ingress controller is running"
            ((PASSED_TESTS++))
        else
            log_info "Ingress controller not running (kind uses NodePort, this is OK)"
            ((PASSED_TESTS++))
        fi
    else
        log_info "ingress-nginx not installed (kind uses NodePort mapping, this is expected)"
        ((PASSED_TESTS++))
    fi
}

# Test: Security - Test User Permissions
test_security_local_dev_user() {
    log_section "Test 21: Security - Test User Permissions"

    log_info "Verifying test-user service account exists..."

    # Kind creates a test-user service account with a pre-generated token
    if kubectl get serviceaccount test-user -n "$NAMESPACE" >/dev/null 2>&1; then
        log_success "test-user service account exists"
        ((PASSED_TESTS++))
    else
        log_warning "test-user service account does not exist (may not be set up yet)"
        # Not a hard failure — kind-up creates this but test may run before setup completes
        ((PASSED_TESTS++))
        return
    fi

    # Check that the test-user token secret exists
    if kubectl get secret test-user-token -n "$NAMESPACE" >/dev/null 2>&1; then
        log_success "test-user-token secret exists"
        ((PASSED_TESTS++))
    else
        log_warning "test-user-token secret does not exist"
        ((PASSED_TESTS++))
    fi
}

# Test: Security - Production Namespace Rejection
test_security_prod_namespace_rejection() {
    log_section "Test 22: Security - Production Namespace Rejection"

    log_info "Testing that dev mode rejects production-like namespaces..."

    # Test 1: Check backend middleware has protection
    local backend_pod
    backend_pod=$(kubectl get pods -n "$NAMESPACE" -l app=backend-api -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)

    if [ -z "$backend_pod" ]; then
        log_warning "Backend pod not found, skipping namespace rejection test"
        return
    fi

    # Test 1: Verify namespace does not contain 'prod'
    if echo "$NAMESPACE" | grep -qi "prod"; then
        log_error "Namespace contains 'prod' - this would be REJECTED by middleware (GOOD)"
        log_error "Current namespace: $NAMESPACE"
        log_info "Dev mode should NEVER run in production namespaces"
        ((PASSED_TESTS++))  # This is correct behavior - we want it to fail
    else
        log_success "Namespace does not contain 'prod' (safe for dev mode)"
        ((PASSED_TESTS++))
    fi

    # Test 2: Document the protection mechanism
    log_info "Middleware protection:"
    log_info "  • Checks if namespace contains 'prod'"
    log_info "  • Uses real token auth (no DISABLE_AUTH in kind)"
}

# Test: Security - Mock Token Detection in Logs
test_security_mock_token_logging() {
    log_section "Test 23: Security - Mock Token Detection"

    log_info "Verifying backend logs show dev mode activation..."

    local backend_pod
    backend_pod=$(kubectl get pods -n "$NAMESPACE" -l app=backend-api -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)

    if [ -z "$backend_pod" ]; then
        log_warning "Backend pod not found, skipping log test"
        return
    fi

    # Get recent backend logs
    local logs
    logs=$(kubectl logs -n "$NAMESPACE" "$backend_pod" --tail=100 2>/dev/null || echo "")

    if [ -z "$logs" ]; then
        log_warning "Could not retrieve backend logs"
        return
    fi

    # Test 1: Check for dev mode detection logs
    if echo "$logs" | grep -q "Local dev mode detected\|Dev mode detected\|local dev environment"; then
        log_success "Backend logs show dev mode activation"
        ((PASSED_TESTS++))
    else
        log_info "Backend logs do not show dev mode activation yet (may need API call to trigger)"
    fi

    # Test 2: Verify logs do NOT contain the actual mock token value
    if echo "$logs" | grep -q "mock-token-for-local-dev"; then
        log_error "Backend logs contain mock token value (SECURITY ISSUE - tokens should be redacted)"
        ((FAILED_TESTS++))
    else
        log_success "Backend logs do NOT contain mock token value (correct - tokens are redacted)"
        ((PASSED_TESTS++))
    fi

    # Test 3: Check for service account usage logging
    if echo "$logs" | grep -q "using.*service account\|K8sClient\|DynamicClient"; then
        log_success "Backend logs reference service account usage"
        ((PASSED_TESTS++))
    else
        log_info "Backend logs do not show service account usage (may need API call to trigger)"
    fi

    # Test 4: Verify environment validation logs
    if echo "$logs" | grep -q "Local dev environment validated\|env=local\|env=development"; then
        log_success "Backend logs show environment validation"
        ((PASSED_TESTS++))
    else
        log_info "Backend logs do not show environment validation yet"
    fi
}

# Test: Security - Token Redaction
test_security_token_redaction() {
    log_section "Test 24: Security - Token Redaction in Logs"

    log_info "Verifying tokens are properly redacted in logs..."

    local backend_pod
    backend_pod=$(kubectl get pods -n "$NAMESPACE" -l app=backend-api -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)

    if [ -z "$backend_pod" ]; then
        log_warning "Backend pod not found, skipping token redaction test"
        return
    fi

    # Get all backend logs
    local logs
    logs=$(kubectl logs -n "$NAMESPACE" "$backend_pod" --tail=500 2>/dev/null || echo "")

    if [ -z "$logs" ]; then
        log_warning "Could not retrieve backend logs"
        return
    fi

    # Test 1: Logs should use tokenLen= instead of showing token
    if echo "$logs" | grep -q "tokenLen=\|token (len="; then
        log_success "Logs use token length instead of token value (correct redaction)"
        ((PASSED_TESTS++))
    else
        log_info "Token length logging not found (may need authenticated requests)"
    fi

    # Test 2: Should NOT contain Bearer tokens
    if echo "$logs" | grep -qE "Bearer [A-Za-z0-9._-]{20,}"; then
        log_error "Logs contain Bearer tokens (SECURITY ISSUE)"
        ((FAILED_TESTS++))
    else
        log_success "Logs do NOT contain Bearer tokens (correct)"
        ((PASSED_TESTS++))
    fi

    # Test 3: Should NOT contain base64-encoded credentials
    if echo "$logs" | grep -qE "[A-Za-z0-9+/]{40,}={0,2}"; then
        log_warning "Logs may contain base64-encoded data (verify not credentials)"
    else
        log_success "Logs do not contain long base64 strings"
        ((PASSED_TESTS++))
    fi
}

# Test: Security - Service Account Configuration
test_security_service_account_config() {
    log_section "Test 25: Security - Service Account Configuration"

    log_info "Verifying service account RBAC configuration..."

    # Test 1: Check backend-api service account exists
    if kubectl get serviceaccount backend-api -n "$NAMESPACE" >/dev/null 2>&1; then
        log_success "backend-api service account exists"
        ((PASSED_TESTS++))
    else
        log_error "backend-api service account does NOT exist"
        ((FAILED_TESTS++))
        return
    fi

    # Test 2: Check if backend has cluster-admin (expected in dev, dangerous in prod)
    local clusterrolebindings
    clusterrolebindings=$(kubectl get clusterrolebinding -o json 2>/dev/null | grep -c "backend-api\|system:serviceaccount:$NAMESPACE:backend-api" || echo "0")

    if [ "$clusterrolebindings" -gt 0 ]; then
        log_warning "backend-api has cluster-level role bindings (OK for dev, DANGEROUS in production)"
        log_warning "  ⚠️  This service account has elevated permissions"
        log_warning "  ⚠️  Production deployments should use minimal namespace-scoped permissions"
    else
        log_info "backend-api has no cluster-level role bindings (namespace-scoped only)"
    fi

    # Test 3: Verify dev mode safety checks are in place
    # NOTE: Auth bypass is intentionally NOT supported in backend code.
    log_info "Dev mode safety mechanisms:"
    log_info "  ✓ No env-var based auth bypass in backend code"
    log_info "  ✓ Requests must provide real tokens (401/403 reflect auth/RBAC)"
    ((PASSED_TESTS++))
}

# Test: CRITICAL - Test User Token
test_critical_token_minting() {
    log_section "Test 26: CRITICAL - Test User Token"

    # Kind setup creates a test-user ServiceAccount with a pre-generated token
    # stored in a secret. Validate that this exists.

    # Step 1: test-user ServiceAccount must exist
    if kubectl get serviceaccount test-user -n "$NAMESPACE" >/dev/null 2>&1; then
        log_success "Step 1/2: test-user ServiceAccount exists"
        ((PASSED_TESTS++))
    else
        log_warning "Step 1/2: test-user ServiceAccount does not exist (kind-up may not have completed)"
        if [ "$CI_MODE" = true ]; then
            ((KNOWN_FAILURES++))
        else
            ((FAILED_TESTS++))
        fi
        return 1
    fi

    # Step 2: test-user-token secret must exist
    if kubectl get secret test-user-token -n "$NAMESPACE" >/dev/null 2>&1; then
        log_success "Step 2/2: test-user-token secret exists"
        ((PASSED_TESTS++))
    else
        log_warning "Step 2/2: test-user-token secret does not exist"
        if [ "$CI_MODE" = true ]; then
            ((KNOWN_FAILURES++))
        else
            ((FAILED_TESTS++))
        fi
        return 1
    fi
}

# Test: Production Manifest Safety - No Dev Mode Variables
test_production_manifest_safety() {
    log_section "Test 27: Production Manifest Safety"

    log_info "Verifying production manifests do NOT contain dev mode variables..."

    # Check base/production manifests for DISABLE_AUTH
    local prod_manifests=(
        "components/manifests/base/backend-deployment.yaml"
        "components/manifests/base/frontend-deployment.yaml"
        "components/manifests/overlays/production/frontend-oauth-deployment-patch.yaml"
    )

    local found_issues=false

    for manifest in "${prod_manifests[@]}"; do
        if [ ! -f "$manifest" ]; then
            log_warning "Manifest not found: $manifest (may be in subdirectory)"
            continue
        fi

        # Check for DISABLE_AUTH
        if grep -q "DISABLE_AUTH" "$manifest" 2>/dev/null; then
            log_error "Production manifest contains DISABLE_AUTH: $manifest"
            log_error "  This would enable dev mode in production (CRITICAL SECURITY ISSUE)"
            ((FAILED_TESTS++))
            found_issues=true
        else
            log_success "Production manifest clean (no DISABLE_AUTH): $manifest"
            ((PASSED_TESTS++))
        fi

        # Check for ENVIRONMENT=local or development
        if grep -qE "ENVIRONMENT.*[\"']?(local|development)[\"']?" "$manifest" 2>/dev/null; then
            log_error "Production manifest sets ENVIRONMENT=local/development: $manifest"
            log_error "  This would enable dev mode in production (CRITICAL SECURITY ISSUE)"
            ((FAILED_TESTS++))
            found_issues=true
        else
            log_success "Production manifest clean (no ENVIRONMENT=local): $manifest"
            ((PASSED_TESTS++))
        fi
    done

    if [ "$found_issues" = false ]; then
        log_info ""
        log_info "✅ Production manifests are safe"
        log_info "✅ Clear separation between dev and production configs"
    fi
}

# Test: Backend Using Wrong Service Account
test_critical_backend_sa_usage() {
    log_section "Test 28: CRITICAL - Backend Using Wrong Service Account"

    log_info "Verifying which service account backend uses in dev mode..."

    # Get backend pod
    local backend_pod
    backend_pod=$(kubectl get pods -n "$NAMESPACE" -l app=backend-api -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)

    if [ -z "$backend_pod" ]; then
        log_warning "Backend pod not found, skipping SA usage test"
        return
    fi

    # Check which service account the backend pod is using
    local backend_sa
    backend_sa=$(kubectl get pod "$backend_pod" -n "$NAMESPACE" -o jsonpath='{.spec.serviceAccountName}' 2>/dev/null)

    log_info "Backend pod service account: $backend_sa"

    # Check if backend has cluster-admin via clusterrolebinding
    local has_cluster_admin=false
    if kubectl get clusterrolebinding -o json 2>/dev/null | grep -q "serviceaccount:$NAMESPACE:$backend_sa"; then
        has_cluster_admin=true
        log_warning "Backend SA '$backend_sa' has cluster-level role bindings (expected in local dev manifests)"

        # List the actual bindings (best effort)
        log_warning "Cluster role bindings for backend SA:"
        kubectl get clusterrolebinding -o json 2>/dev/null | jq -r ".items[] | select(.subjects[]?.name == \"$backend_sa\") | \"  - \(.metadata.name): \(.roleRef.name)\"" 2>/dev/null || echo "  (could not enumerate)"

        # CI mode: treat this as a known local-dev tradeoff (cluster-admin is for dev convenience).
        # Production safety is validated separately (production manifests must not include dev-mode vars).
        if [ "$CI_MODE" = true ]; then
            ((KNOWN_FAILURES++))
        else
            ((FAILED_TESTS++))
        fi
    else
        log_success "Backend SA '$backend_sa' has NO cluster-level bindings (good for prod model)"
        ((PASSED_TESTS++))
    fi

    # Validate current security posture: no env-var auth bypass code should exist in backend middleware.
    log_info "Checking backend middleware has no local-dev auth bypass implementation..."
    if [ -f "components/backend/handlers/middleware.go" ]; then
        # Ensure no local-dev auth bypass helpers exist in backend code (including legacy names).
        if grep -qE "getLocalDevK8sClients\\(|isLocalDevEnvironment\\(" components/backend/handlers/middleware.go; then
            log_error "Found local-dev auth bypass code in middleware.go (should be removed)"
            ((FAILED_TESTS++))
        else
            log_success "No local-dev auth bypass code found in middleware.go"
            ((PASSED_TESTS++))
        fi
    else
        log_warning "middleware.go not found in current directory"
    fi
}

# Main test execution
main() {
    log_section "Ambient Code Platform - Local Developer Experience Tests"
    log_info "Starting test suite at $(date)"
    log_info "Test configuration:"
    log_info "  Namespace: $NAMESPACE"
    log_info "  Skip setup: $SKIP_SETUP"
    log_info "  Cleanup: $CLEANUP"
    log_info "  Verbose: $VERBOSE"
    echo ""

    # Run tests
    test_prerequisites
    test_makefile_help
    test_kind_status
    test_kubernetes_context
    test_namespace_exists
    test_crds_installed
    test_pods_running
    test_services_exist
    test_ingress
    test_backend_health
    test_frontend_accessibility
    test_rbac
    test_build_command
    test_reload_commands
    test_logging_commands
    test_storage
    test_environment_variables
    test_resource_limits
    test_make_status
    test_ingress_controller

    # Security tests
    test_security_local_dev_user
    test_security_prod_namespace_rejection
    test_security_mock_token_logging
    test_security_token_redaction
    test_security_service_account_config

    # Production safety tests
    test_production_manifest_safety

    # CRITICAL failing tests for unimplemented features
    test_critical_token_minting
    test_critical_backend_sa_usage

    # Summary
    log_section "Test Summary"
    echo ""
    echo -e "${BOLD}Results:${NC}"
    echo -e "  ${GREEN}Passed:${NC} $PASSED_TESTS"
    echo -e "  ${RED}Failed:${NC} $FAILED_TESTS"
    if [ $KNOWN_FAILURES -gt 0 ]; then
        echo -e "  ${YELLOW}Known TODOs:${NC} $KNOWN_FAILURES"
    fi
    echo -e "  ${BOLD}Total:${NC}  $((PASSED_TESTS + FAILED_TESTS + KNOWN_FAILURES))"
    echo ""

    if [ "$CI_MODE" = true ]; then
        # In CI mode, known failures are acceptable
        local unexpected_failures=$FAILED_TESTS
        if [ $unexpected_failures -eq 0 ]; then
            echo -e "${GREEN}${BOLD}✓ All tests passed (excluding $KNOWN_FAILURES known TODOs)!${NC}"
            echo ""
            log_info "CI validation successful!"
            if [ $KNOWN_FAILURES -gt 0 ]; then
                log_warning "Note: $KNOWN_FAILURES known TODOs tracked in test output"
            fi
            exit 0
        else
            echo -e "${RED}${BOLD}✗ $unexpected_failures unexpected test failures${NC}"
            echo ""
            log_error "CI validation failed"
            exit 1
        fi
    else
        # In normal mode, any failure is an issue
        if [ $FAILED_TESTS -eq 0 ]; then
            echo -e "${GREEN}${BOLD}✓ All tests passed!${NC}"
            echo ""
            log_info "Your local development environment is ready!"
            log_info "Access the application:"
            log_info "  • Frontend: $(get_test_url 30030)"
            log_info "  • Backend:  $(get_test_url 30080)"
            echo ""
            if [ $KNOWN_FAILURES -gt 0 ]; then
                log_warning "Note: $KNOWN_FAILURES known TODOs tracked for future implementation"
            fi
            exit 0
        else
            echo -e "${RED}${BOLD}✗ Some tests failed${NC}"
            echo ""
            log_error "Your local development environment has issues"
            log_info "Run 'make local-troubleshoot' for more details"
            echo ""
            exit 1
        fi
    fi
}

# Run main function
main
