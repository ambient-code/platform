.PHONY: help setup build-all build-frontend build-backend build-operator build-runner build-state-sync build-public-api build-cli deploy clean check-architecture
.PHONY: local-down local-status local-reload-api-server local-up local-clean local-rebuild local-reload-backend local-reload-frontend local-reload-operator
.PHONY: local-dev-token
.PHONY: local-logs local-logs-backend local-logs-frontend local-logs-operator local-shell local-shell-frontend
.PHONY: local-test local-test-dev local-test-quick test-all local-troubleshoot local-port-forward local-stop-port-forward
.PHONY: push-all registry-login setup-hooks remove-hooks lint check-minikube check-kind check-kubectl check-local-context dev-bootstrap kind-rebuild kind-reload-backend kind-reload-frontend kind-reload-operator kind-status kind-login
.PHONY: preflight-cluster preflight dev-env dev
.PHONY: e2e-test e2e-setup e2e-clean deploy-langfuse-openshift
.PHONY: unleash-port-forward unleash-status
.PHONY: setup-minio minio-console minio-logs minio-status
.PHONY: validate-makefile lint-makefile check-shell makefile-health benchmark benchmark-ci
.PHONY: _create-operator-config _auto-port-forward _show-access-info _kind-load-images

# Default target
.DEFAULT_GOAL := help

# Configuration
CONTAINER_ENGINE ?= podman

# Auto-detect host architecture for native builds
# Override with PLATFORM=linux/amd64 or PLATFORM=linux/arm64 if needed
HOST_OS := $(shell uname -s)
HOST_ARCH := $(shell uname -m)

# Map uname output to Docker platform names
ifeq ($(HOST_ARCH),arm64)
    DETECTED_PLATFORM := linux/arm64
else ifeq ($(HOST_ARCH),aarch64)
    DETECTED_PLATFORM := linux/arm64
else ifeq ($(HOST_ARCH),x86_64)
    DETECTED_PLATFORM := linux/amd64
else ifeq ($(HOST_ARCH),amd64)
    DETECTED_PLATFORM := linux/amd64
else
    DETECTED_PLATFORM := linux/amd64
    $(warning Unknown architecture $(HOST_ARCH), defaulting to linux/amd64)
endif

# Allow manual override via PLATFORM=...
PLATFORM ?= $(DETECTED_PLATFORM)
BUILD_FLAGS ?=
NAMESPACE ?= ambient-code
REGISTRY ?= quay.io/your-org
CI_MODE ?= false

# In CI we want full command output to diagnose failures. Locally we keep the Makefile quieter.
# GitHub Actions sets CI=true by default; the workflow can also pass CI_MODE=true explicitly.
ifeq ($(CI),true)
CI_MODE := true
endif

ifeq ($(CI_MODE),true)
QUIET_REDIRECT :=
else
QUIET_REDIRECT := >/dev/null 2>&1
endif

# Image tag (override with: make build-all IMAGE_TAG=v1.2.3)
IMAGE_TAG ?= latest

# Image names
FRONTEND_IMAGE ?= vteam_frontend:$(IMAGE_TAG)
BACKEND_IMAGE ?= vteam_backend:$(IMAGE_TAG)
OPERATOR_IMAGE ?= vteam_operator:$(IMAGE_TAG)
RUNNER_IMAGE ?= vteam_claude_runner:$(IMAGE_TAG)
STATE_SYNC_IMAGE ?= vteam_state_sync:$(IMAGE_TAG)
PUBLIC_API_IMAGE ?= vteam_public_api:$(IMAGE_TAG)
API_SERVER_IMAGE ?= vteam_api_server:$(IMAGE_TAG)

# Podman prefixes image names with localhost/ — kind load needs to use the same
# name so containerd can match the image reference used in the deployment spec
KIND_IMAGE_PREFIX := $(if $(filter podman,$(CONTAINER_ENGINE)),localhost/,)

# Load local developer config (KIND_HOST, etc.) — gitignored, set once per machine
-include .env.local

# Kind cluster configuration — derived from git branch for multi-worktree support
# Each worktree/branch gets a unique cluster name and ports automatically.
# Override any variable: make kind-up KIND_CLUSTER_NAME=ambient-custom KIND_FWD_FRONTEND_PORT=8080
CLUSTER_SLUG ?= $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9]/-/g' | sed 's/--*/-/g' | sed 's/^-//' | sed 's/-$$//' | cut -c1-20)
CLUSTER_SLUG := $(CLUSTER_SLUG)
KIND_CLUSTER_NAME ?= ambient-$(CLUSTER_SLUG)
KIND_CLUSTER_NAME := $(KIND_CLUSTER_NAME)
# Deterministic port offset from slug hash (0-999) — all ports derive from this
KIND_PORT_OFFSET ?= $(shell printf '%s' '$(CLUSTER_SLUG)' | cksum | awk '{print $$1 % 1000}')
KIND_PORT_OFFSET := $(KIND_PORT_OFFSET)
KIND_HTTP_PORT ?= $(shell echo $$((9000 + $(KIND_PORT_OFFSET))))
KIND_HTTP_PORT := $(KIND_HTTP_PORT)
KIND_HTTPS_PORT ?= $(shell echo $$((10000 + $(KIND_PORT_OFFSET))))
KIND_HTTPS_PORT := $(KIND_HTTPS_PORT)
KIND_FWD_FRONTEND_PORT ?= $(shell echo $$((11000 + $(KIND_PORT_OFFSET))))
KIND_FWD_FRONTEND_PORT := $(KIND_FWD_FRONTEND_PORT)
KIND_FWD_BACKEND_PORT ?= $(shell echo $$((12000 + $(KIND_PORT_OFFSET))))
KIND_FWD_BACKEND_PORT := $(KIND_FWD_BACKEND_PORT)
KIND_FWD_API_SERVER_PORT ?= $(shell echo $$((13000 + $(KIND_PORT_OFFSET))))
KIND_FWD_API_SERVER_PORT := $(KIND_FWD_API_SERVER_PORT)
# Remote kind host — set to Tailscale IP/hostname of the Linux build machine.
# When set, kubeconfig is rewritten so kubectl/port-forward work from Mac.
KIND_HOST ?=

# Vertex AI Configuration (for LOCAL_VERTEX=true)
# These inherit from environment if set, or can be overridden on command line
LOCAL_IMAGES ?= false
LOCAL_VERTEX ?= false
ANTHROPIC_VERTEX_PROJECT_ID ?= $(shell echo $$ANTHROPIC_VERTEX_PROJECT_ID)
CLOUD_ML_REGION ?= $(shell echo $$CLOUD_ML_REGION)
# Default to ADC location if not set (created by: gcloud auth application-default login)
GOOGLE_APPLICATION_CREDENTIALS ?= $(or $(shell echo $$GOOGLE_APPLICATION_CREDENTIALS),$(HOME)/.config/gcloud/application_default_credentials.json)


# Colors for output (using tput for better compatibility, with fallback to printf-compatible codes)
# Use shell assignment to evaluate tput at runtime if available
COLOR_RESET := $(shell tput sgr0 2>/dev/null || printf '\033[0m')
COLOR_BOLD := $(shell tput bold 2>/dev/null || printf '\033[1m')
COLOR_GREEN := $(shell tput setaf 2 2>/dev/null || printf '\033[32m')
COLOR_YELLOW := $(shell tput setaf 3 2>/dev/null || printf '\033[33m')
COLOR_BLUE := $(shell tput setaf 4 2>/dev/null || printf '\033[34m')
COLOR_RED := $(shell tput setaf 1 2>/dev/null || printf '\033[31m')

# Platform flag
ifneq ($(PLATFORM),)
PLATFORM_FLAG := --platform=$(PLATFORM)
else
PLATFORM_FLAG :=
endif

##@ General

help: ## Display this help message
	@echo '$(COLOR_BOLD)Ambient Code Platform - Development Makefile$(COLOR_RESET)'
	@echo ''
	@echo '$(COLOR_BOLD)Quick Start:$(COLOR_RESET)'
	@echo '  $(COLOR_GREEN)make dev$(COLOR_RESET)                  Start local dev environment (interactive)'
	@echo '  $(COLOR_GREEN)make dev COMPONENT=frontend$(COLOR_RESET)   Hot-reload frontend against kind cluster'
	@echo '  $(COLOR_GREEN)make kind-up$(COLOR_RESET)             Full cluster deploy (no hot-reload)'
	@echo '  $(COLOR_GREEN)make kind-status$(COLOR_RESET)         Check kind cluster status'
	@echo '  $(COLOR_GREEN)make kind-down$(COLOR_RESET)           Stop and delete the kind cluster'
	@echo ''
	@echo '$(COLOR_BOLD)Quality Assurance:$(COLOR_RESET)'
	@echo '  $(COLOR_GREEN)make validate-makefile$(COLOR_RESET)   Validate Makefile quality (runs in CI)'
	@echo '  $(COLOR_GREEN)make makefile-health$(COLOR_RESET)     Run comprehensive health check'
	@echo ''
	@awk 'BEGIN {FS = ":.*##"; printf "$(COLOR_BOLD)Available Targets:$(COLOR_RESET)\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  $(COLOR_BLUE)%-20s$(COLOR_RESET) %s\n", $$1, $$2 } /^##@/ { printf "\n$(COLOR_BOLD)%s$(COLOR_RESET)\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
	@echo ''
	@echo '$(COLOR_BOLD)Configuration Variables:$(COLOR_RESET)'
	@echo '  CONTAINER_ENGINE=$(CONTAINER_ENGINE)  (docker or podman)'
	@echo '  NAMESPACE=$(NAMESPACE)'
	@echo '  PLATFORM=$(PLATFORM) (detected: $(DETECTED_PLATFORM) from $(HOST_OS)/$(HOST_ARCH))'
	@echo ''
	@echo '$(COLOR_BOLD)Kind Cluster (current worktree):$(COLOR_RESET)'
	@echo '  CLUSTER_SLUG=$(CLUSTER_SLUG)'
	@echo '  KIND_CLUSTER_NAME=$(KIND_CLUSTER_NAME)'
	@echo '  Ports: frontend=$(KIND_FWD_FRONTEND_PORT) backend=$(KIND_FWD_BACKEND_PORT) http=$(KIND_HTTP_PORT) https=$(KIND_HTTPS_PORT)'
	@echo ''
	@echo '$(COLOR_BOLD)Examples:$(COLOR_RESET)'
	@echo '  make kind-up LOCAL_IMAGES=true    Build from source and deploy to kind'
	@echo '  make kind-rebuild                 Rebuild and reload all components in kind'
	@echo '  make kind-status                  Show all kind clusters and their ports'
	@echo '  make kind-up CONTAINER_ENGINE=docker'
	@echo '  make kind-rebuild'
	@echo '  make build-all PLATFORM=linux/arm64'

##@ Building

build-all: build-frontend build-backend build-operator build-runner build-state-sync build-public-api build-api-server ## Build all container images

build-frontend: ## Build frontend image
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Building frontend with $(CONTAINER_ENGINE)..."
	@cd components/frontend && $(CONTAINER_ENGINE) build $(PLATFORM_FLAG) $(BUILD_FLAGS) \
		-t $(FRONTEND_IMAGE) .
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Frontend built: $(FRONTEND_IMAGE)"

build-backend: ## Build backend image
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Building backend with $(CONTAINER_ENGINE)..."
	@cd components/backend && $(CONTAINER_ENGINE) build $(PLATFORM_FLAG) $(BUILD_FLAGS) \
		--build-arg AMBIENT_VERSION=$(shell git describe --tags --always --dirty) \
		-t $(BACKEND_IMAGE) .
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Backend built: $(BACKEND_IMAGE)"

build-operator: ## Build operator image
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Building operator with $(CONTAINER_ENGINE)..."
	@cd components/operator && $(CONTAINER_ENGINE) build $(PLATFORM_FLAG) $(BUILD_FLAGS) \
		-t $(OPERATOR_IMAGE) .
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Operator built: $(OPERATOR_IMAGE)"

build-runner: ## Build Claude Code runner image
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Building runner with $(CONTAINER_ENGINE)..."
	@cd components/runners && $(CONTAINER_ENGINE) build $(PLATFORM_FLAG) $(BUILD_FLAGS) \
		-t $(RUNNER_IMAGE) -f ambient-runner/Dockerfile .
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Runner built: $(RUNNER_IMAGE)"

build-state-sync: ## Build state-sync image for S3 persistence
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Building state-sync with $(CONTAINER_ENGINE)..."
	@cd components/runners/state-sync && $(CONTAINER_ENGINE) build $(PLATFORM_FLAG) $(BUILD_FLAGS) \
		-t $(STATE_SYNC_IMAGE) .
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) State-sync built: $(STATE_SYNC_IMAGE)"

build-public-api: ## Build public API gateway image
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Building public-api with $(CONTAINER_ENGINE)..."
	@cd components/public-api && $(CONTAINER_ENGINE) build $(PLATFORM_FLAG) $(BUILD_FLAGS) \
		-t $(PUBLIC_API_IMAGE) .
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Public API built: $(PUBLIC_API_IMAGE)"

build-api-server: ## Build ambient API server image
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Building ambient-api-server with $(CONTAINER_ENGINE)..."
	@cd components/ambient-api-server && $(CONTAINER_ENGINE) build $(PLATFORM_FLAG) $(BUILD_FLAGS) \
		-t $(API_SERVER_IMAGE) .
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) API server built: $(API_SERVER_IMAGE)"

build-cli: ## Build acpctl CLI binary
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Building acpctl CLI..."
	@cd components/ambient-cli && make build
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) CLI built: components/ambient-cli/acpctl"

lint-cli: ## Lint acpctl CLI
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Linting acpctl CLI..."
	@cd components/ambient-cli && make lint
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) CLI lint passed"

test-cli: ## Test acpctl CLI
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Testing acpctl CLI..."
	@cd components/ambient-cli && make test
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) CLI tests passed"

##@ Git Hooks & Linting

setup-hooks: ## Install pre-commit hooks (linters + branch protection)
	@./scripts/install-git-hooks.sh

remove-hooks: ## Remove pre-commit hooks
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Removing git hooks..."
	@if command -v pre-commit >/dev/null 2>&1; then \
		pre-commit uninstall && pre-commit uninstall --hook-type pre-push; \
	else \
		rm -f .git/hooks/pre-commit .git/hooks/pre-push; \
	fi
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Git hooks removed"

lint: ## Run all pre-commit linters on the entire repo
	@if ! command -v pre-commit >/dev/null 2>&1; then \
		echo "$(COLOR_RED)✗$(COLOR_RESET) pre-commit not installed. Run: make setup-hooks"; \
		exit 1; \
	fi
	pre-commit run --all-files

##@ Registry Operations

registry-login: ## Login to container registry
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Logging in to $(REGISTRY)..."
	@$(CONTAINER_ENGINE) login $(REGISTRY)

push-all: registry-login ## Push all images to registry
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Pushing images to $(REGISTRY)..."
	@for image in $(FRONTEND_IMAGE) $(BACKEND_IMAGE) $(OPERATOR_IMAGE) $(RUNNER_IMAGE) $(STATE_SYNC_IMAGE) $(PUBLIC_API_IMAGE) $(API_SERVER_IMAGE); do \
		echo "  Tagging and pushing $$image..."; \
		$(CONTAINER_ENGINE) tag $$image $(REGISTRY)/$$image && \
		$(CONTAINER_ENGINE) push $(REGISTRY)/$$image; \
	done
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) All images pushed"

##@ MinIO S3 Storage

setup-minio: ## Set up MinIO and create initial bucket
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Setting up MinIO for S3 state storage..."
	@./scripts/setup-minio.sh
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) MinIO setup complete"

minio-console: ## Open MinIO console (port-forward to localhost:9001)
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Opening MinIO console at http://localhost:9001"
	@echo "  Login: admin / changeme123 (or your configured credentials)"
	@kubectl port-forward svc/minio 9001:9001 -n $(NAMESPACE)

minio-logs: ## View MinIO logs
	@kubectl logs -f deployment/minio -n $(NAMESPACE)

minio-status: ## Check MinIO status
	@echo "$(COLOR_BOLD)MinIO Status$(COLOR_RESET)"
	@kubectl get deployment,pod,svc,pvc -l app=minio -n $(NAMESPACE)

##@ Observability

deploy-observability: ## Deploy observability (OTel + OpenShift Prometheus)
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Deploying observability stack..."
	@kubectl apply -k components/manifests/observability/
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Observability deployed (OTel + ServiceMonitor)"
	@echo "  View metrics: OpenShift Console → Observe → Metrics"
	@echo "  Optional Grafana: make add-grafana"

add-grafana: ## Add Grafana on top of observability stack
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Adding Grafana..."
	@kubectl apply -k components/manifests/observability/overlays/with-grafana/
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Grafana deployed"
	@echo "  Create route: oc create route edge grafana --service=grafana -n $(NAMESPACE)"

clean-observability: ## Remove observability components
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Removing observability..."
	@kubectl delete -k components/manifests/observability/overlays/with-grafana/ 2>/dev/null || true
	@kubectl delete -k components/manifests/observability/ 2>/dev/null || true
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Observability removed"

grafana-dashboard: ## Open Grafana (create route first)
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Opening Grafana..."
	@oc create route edge grafana --service=grafana -n $(NAMESPACE) 2>/dev/null || echo "Route already exists"
	@echo "  URL: https://$$(oc get route grafana -n $(NAMESPACE) -o jsonpath='{.spec.host}')"
	@echo "  Login: admin/admin"

##@ Local Development

local-down: check-kubectl check-local-context ## Stop Ambient Code Platform (keep cluster running)
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Stopping Ambient Code Platform..."
	@$(MAKE) --no-print-directory local-stop-port-forward
	@kubectl delete namespace $(NAMESPACE) --ignore-not-found=true --timeout=60s
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Ambient Code Platform stopped (cluster still running)"
	@echo "  To delete kind cluster: $(COLOR_BOLD)make kind-down$(COLOR_RESET)"

local-status: check-kubectl ## Show status of local deployment
	@echo "$(COLOR_BOLD)📊 Ambient Code Platform Status$(COLOR_RESET)"
	@echo ""
	@if $(if $(filter podman,$(CONTAINER_ENGINE)),KIND_EXPERIMENTAL_PROVIDER=podman) kind get clusters 2>/dev/null | grep -q '^$(KIND_CLUSTER_NAME)$$'; then \
		echo "$(COLOR_BOLD)Kind:$(COLOR_RESET)"; \
		echo "$(COLOR_GREEN)✓$(COLOR_RESET) Cluster '$(KIND_CLUSTER_NAME)' running"; \
	else \
		echo "$(COLOR_RED)✗$(COLOR_RESET) No kind cluster found. Run 'make kind-up' first."; \
	fi
	@echo ""
	@echo "$(COLOR_BOLD)Pods:$(COLOR_RESET)"
	@kubectl get pods -n $(NAMESPACE) -o wide 2>/dev/null || echo "$(COLOR_RED)✗$(COLOR_RESET) Namespace not found"
	@echo ""
	@echo "$(COLOR_BOLD)Services:$(COLOR_RESET)"
	@kubectl get svc -n $(NAMESPACE) 2>/dev/null | grep -E "NAME|NodePort" || echo "No services found"
	@echo ""
	@if $(if $(filter podman,$(CONTAINER_ENGINE)),KIND_EXPERIMENTAL_PROVIDER=podman) kind get clusters 2>/dev/null | grep -q '^$(KIND_CLUSTER_NAME)$$'; then \
		echo "$(COLOR_BOLD)Access URLs:$(COLOR_RESET)"; \
		echo "  Run in another terminal: $(COLOR_BLUE)make kind-port-forward$(COLOR_RESET)"; \
		echo "  Frontend: $(COLOR_BLUE)http://localhost:$(KIND_FWD_FRONTEND_PORT)$(COLOR_RESET)"; \
		echo "  Backend:  $(COLOR_BLUE)http://localhost:$(KIND_FWD_BACKEND_PORT)$(COLOR_RESET)"; \
	fi

local-reload-api-server: check-local-context ## Rebuild and reload ambient-api-server only
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Rebuilding ambient-api-server..."
	@$(CONTAINER_ENGINE) build $(PLATFORM_FLAG) -t $(API_SERVER_IMAGE) components/ambient-api-server >/dev/null 2>&1
	@$(CONTAINER_ENGINE) tag $(API_SERVER_IMAGE) localhost/$(API_SERVER_IMAGE) 2>/dev/null || true
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Loading image into kind cluster ($(KIND_CLUSTER_NAME))..."
	@$(CONTAINER_ENGINE) save localhost/$(API_SERVER_IMAGE) | \
		$(CONTAINER_ENGINE) exec -i $(KIND_CLUSTER_NAME)-control-plane \
		ctr --namespace=k8s.io images import -
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Restarting ambient-api-server..."
	@kubectl rollout restart deployment/ambient-api-server -n $(NAMESPACE) >/dev/null 2>&1
	@kubectl rollout status deployment/ambient-api-server -n $(NAMESPACE) --timeout=60s
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) ambient-api-server reloaded"

##@ Testing

test-all: test-cli local-test-quick local-test-dev ## Run all tests (quick + comprehensive)

##@ Quality Assurance

validate-makefile: lint-makefile check-shell ## Validate Makefile quality and syntax
	@echo "$(COLOR_GREEN)✓ Makefile validation passed$(COLOR_RESET)"

lint-makefile: ## Lint Makefile for syntax and best practices
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Linting Makefile..."
	@# Check that all targets have help text or are internal/phony
	@missing_help=$$(awk '/^[a-zA-Z_-]+:/ && !/##/ && !/^_/ && !/^\.PHONY/ && !/^\.DEFAULT_GOAL/' $(MAKEFILE_LIST)); \
	if [ -n "$$missing_help" ]; then \
		echo "$(COLOR_YELLOW)⚠$(COLOR_RESET)  Targets missing help text:"; \
		echo "$$missing_help" | head -5; \
	fi
	@# Check for common mistakes
	@if grep -n "^\t " $(MAKEFILE_LIST) | grep -v "^#" >/dev/null 2>&1; then \
		echo "$(COLOR_RED)✗$(COLOR_RESET) Found tabs followed by spaces (use tabs only for indentation)"; \
		grep -n "^\t " $(MAKEFILE_LIST) | head -3; \
		exit 1; \
	fi
	@# Check for undefined variable references (basic check)
	@if grep -E '\$$[^($$@%<^+?*]' $(MAKEFILE_LIST) | grep -v "^#" | grep -v '\$$\$$' >/dev/null 2>&1; then \
		echo "$(COLOR_YELLOW)⚠$(COLOR_RESET)  Possible unprotected variable references found"; \
	fi
	@# Verify .PHONY declarations exist
	@if ! grep -q "^\.PHONY:" $(MAKEFILE_LIST); then \
		echo "$(COLOR_RED)✗$(COLOR_RESET) No .PHONY declarations found"; \
		exit 1; \
	fi
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Makefile syntax validated"

check-shell: ## Validate shell scripts with shellcheck (if available)
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Checking shell scripts..."
	@if command -v shellcheck >/dev/null 2>&1; then \
		echo "  Running shellcheck on test scripts..."; \
		shellcheck tests/local-dev-test.sh 2>/dev/null || echo "$(COLOR_YELLOW)⚠$(COLOR_RESET)  shellcheck warnings in tests/local-dev-test.sh"; \
		if [ -d e2e/scripts ]; then \
			shellcheck e2e/scripts/*.sh 2>/dev/null || echo "$(COLOR_YELLOW)⚠$(COLOR_RESET)  shellcheck warnings in e2e scripts"; \
		fi; \
		echo "$(COLOR_GREEN)✓$(COLOR_RESET) Shell scripts checked"; \
	else \
		echo "$(COLOR_YELLOW)⚠$(COLOR_RESET)  shellcheck not installed (optional)"; \
		echo "  Install with: brew install shellcheck (macOS) or apt-get install shellcheck (Linux)"; \
	fi

makefile-health: check-kind check-kubectl ## Run comprehensive Makefile health check
	@echo "$(COLOR_BOLD)🏥 Makefile Health Check$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BOLD)Prerequisites:$(COLOR_RESET)"
	@kind version >/dev/null 2>&1 && echo "$(COLOR_GREEN)✓$(COLOR_RESET) kind available" || echo "$(COLOR_RED)✗$(COLOR_RESET) kind missing"
	@kubectl version --client >/dev/null 2>&1 && echo "$(COLOR_GREEN)✓$(COLOR_RESET) kubectl available" || echo "$(COLOR_RED)✗$(COLOR_RESET) kubectl missing"
	@command -v $(CONTAINER_ENGINE) >/dev/null 2>&1 && echo "$(COLOR_GREEN)✓$(COLOR_RESET) $(CONTAINER_ENGINE) available" || echo "$(COLOR_RED)✗$(COLOR_RESET) $(CONTAINER_ENGINE) missing"
	@echo ""
	@echo "$(COLOR_BOLD)Configuration:$(COLOR_RESET)"
	@echo "  CONTAINER_ENGINE = $(CONTAINER_ENGINE)"
	@echo "  NAMESPACE = $(NAMESPACE)"
	@echo "  PLATFORM = $(PLATFORM)"
	@echo ""
	@$(MAKE) --no-print-directory validate-makefile
	@echo ""
	@echo "$(COLOR_GREEN)✓ Makefile health check complete$(COLOR_RESET)"

local-test-dev: ## Run local developer experience tests
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Running local developer experience tests..."
	@./tests/local-dev-test.sh $(if $(filter true,$(CI_MODE)),--ci,)

local-test-quick: check-kubectl ## Quick smoke test of local environment
	@echo "$(COLOR_BOLD)🧪 Quick Smoke Test$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Detecting cluster type..."
	@if kind get clusters 2>/dev/null | grep -q .; then \
		echo "$(COLOR_GREEN)✓$(COLOR_RESET) Kind cluster running"; \
	else \
		echo "$(COLOR_RED)✗$(COLOR_RESET) No kind cluster found. Run 'make kind-up' first."; exit 1; \
	fi
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Testing namespace..."
	@kubectl get namespace $(NAMESPACE) >/dev/null 2>&1 && echo "$(COLOR_GREEN)✓$(COLOR_RESET) Namespace exists" || (echo "$(COLOR_RED)✗$(COLOR_RESET) Namespace missing" && exit 1)
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Waiting for pods to be ready..."
	@kubectl wait --for=condition=ready pod -l app=backend-api -n $(NAMESPACE) --timeout=60s >/dev/null 2>&1 && \
	kubectl wait --for=condition=ready pod -l app=frontend -n $(NAMESPACE) --timeout=60s >/dev/null 2>&1 && \
	echo "$(COLOR_GREEN)✓$(COLOR_RESET) Pods ready" || (echo "$(COLOR_RED)✗$(COLOR_RESET) Pods not ready" && exit 1)
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Testing backend health..."
	@kubectl port-forward -n $(NAMESPACE) svc/backend-service 18080:8080 >/tmp/pf-smoke-backend.log 2>&1 & PF_PID=$$!; \
	sleep 2; \
	for i in 1 2 3 4 5; do \
		curl -sf http://localhost:18080/health >/dev/null 2>&1 && { echo "$(COLOR_GREEN)✓$(COLOR_RESET) Backend healthy"; break; } || { \
			if [ $$i -eq 5 ]; then \
				kill $$PF_PID 2>/dev/null; echo "$(COLOR_RED)✗$(COLOR_RESET) Backend not responding after 5 attempts"; exit 1; \
			fi; \
			sleep 2; \
		}; \
	done; \
	kill $$PF_PID 2>/dev/null || true
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Testing frontend..."
	@kubectl port-forward -n $(NAMESPACE) svc/frontend-service 13030:3000 >/tmp/pf-smoke-frontend.log 2>&1 & PF_PID=$$!; \
	sleep 2; \
	for i in 1 2 3 4 5; do \
		curl -sf http://localhost:13030 >/dev/null 2>&1 && { echo "$(COLOR_GREEN)✓$(COLOR_RESET) Frontend accessible"; break; } || { \
			if [ $$i -eq 5 ]; then \
				kill $$PF_PID 2>/dev/null; echo "$(COLOR_RED)✗$(COLOR_RESET) Frontend not responding after 5 attempts"; exit 1; \
			fi; \
			sleep 2; \
		}; \
	done; \
	kill $$PF_PID 2>/dev/null || true
	@echo ""
	@echo "$(COLOR_GREEN)✓ Quick smoke test passed!$(COLOR_RESET)"

dev-test-operator: ## Run only operator tests
	@echo "Running operator-specific tests..."
	@bash components/scripts/local-dev/crc-test.sh 2>&1 | grep -A 1 "Operator"

##@ Development Tools

local-logs: check-kubectl ## Show logs from all components (follow mode)
	@echo "$(COLOR_BOLD)📋 Streaming logs from all components (Ctrl+C to stop)$(COLOR_RESET)"
	@kubectl logs -n $(NAMESPACE) -l 'app in (backend-api,frontend,agentic-operator)' --tail=20 --prefix=true -f 2>/dev/null || \
		echo "$(COLOR_RED)✗$(COLOR_RESET) No pods found. Run 'make local-status' to check deployment."

local-logs-backend: check-kubectl ## Show backend logs only
	@kubectl logs -n $(NAMESPACE) -l app=backend-api --tail=100 -f

local-logs-frontend: check-kubectl ## Show frontend logs only
	@kubectl logs -n $(NAMESPACE) -l app=frontend --tail=100 -f

local-logs-operator: check-kubectl ## Show operator logs only
	@kubectl logs -n $(NAMESPACE) -l app=agentic-operator --tail=100 -f

local-shell: check-kubectl ## Open shell in backend pod
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Opening shell in backend pod..."
	@kubectl exec -it -n $(NAMESPACE) $$(kubectl get pod -n $(NAMESPACE) -l app=backend-api -o jsonpath='{.items[0].metadata.name}' 2>/dev/null) -- /bin/sh 2>/dev/null || \
		echo "$(COLOR_RED)✗$(COLOR_RESET) Backend pod not found or not ready"

local-shell-frontend: check-kubectl ## Open shell in frontend pod
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Opening shell in frontend pod..."
	@kubectl exec -it -n $(NAMESPACE) $$(kubectl get pod -n $(NAMESPACE) -l app=frontend -o jsonpath='{.items[0].metadata.name}' 2>/dev/null) -- /bin/sh 2>/dev/null || \
		echo "$(COLOR_RED)✗$(COLOR_RESET) Frontend pod not found or not ready"

local-test: local-test-quick ## Alias for local-test-quick (backward compatibility)

local-port-forward: check-kubectl ## Port-forward for direct access (8080→backend, 3000→frontend)
	@echo "$(COLOR_BOLD)🔌 Setting up port forwarding$(COLOR_RESET)"
	@echo ""
	@echo "  Backend:  http://localhost:8080"
	@echo "  Frontend: http://localhost:3000"
	@echo ""
	@echo "$(COLOR_YELLOW)Press Ctrl+C to stop$(COLOR_RESET)"
	@echo ""
	@trap 'echo ""; echo "$(COLOR_GREEN)✓$(COLOR_RESET) Port forwarding stopped"; exit 0' INT; \
	(kubectl port-forward -n $(NAMESPACE) svc/backend-service 8080:8080 >/dev/null 2>&1 &); \
	(kubectl port-forward -n $(NAMESPACE) svc/frontend-service 3000:3000 >/dev/null 2>&1 &); \
	wait

local-troubleshoot: check-kubectl ## Show troubleshooting information
	@echo "$(COLOR_BOLD)🔍 Troubleshooting Information$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BOLD)Pod Status:$(COLOR_RESET)"
	@kubectl get pods -n $(NAMESPACE) -o wide 2>/dev/null || echo "$(COLOR_RED)✗$(COLOR_RESET) No pods found"
	@echo ""
	@echo "$(COLOR_BOLD)Recent Events:$(COLOR_RESET)"
	@kubectl get events -n $(NAMESPACE) --sort-by='.lastTimestamp' | tail -10 2>/dev/null || echo "No events"
	@echo ""
	@echo "$(COLOR_BOLD)Failed Pods (if any):$(COLOR_RESET)"
	@kubectl get pods -n $(NAMESPACE) --field-selector=status.phase!=Running,status.phase!=Succeeded 2>/dev/null || echo "All pods are running"
	@echo ""
	@echo "$(COLOR_BOLD)Pod Descriptions:$(COLOR_RESET)"
	@for pod in $$(kubectl get pods -n $(NAMESPACE) -o name 2>/dev/null | head -3); do \
		echo ""; \
		echo "$(COLOR_BLUE)$$pod:$(COLOR_RESET)"; \
		kubectl describe -n $(NAMESPACE) $$pod | grep -A 5 "Conditions:\|Events:" | head -10; \
	done

##@ Production Deployment

deploy: ## Deploy to production Kubernetes cluster
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Deploying to Kubernetes..."
	@cd components/manifests && ./deploy.sh
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Deployment complete"

clean: ## Clean up Kubernetes resources
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Cleaning up..."
	@cd components/manifests && ./deploy.sh clean
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Cleanup complete"

##@ Kind Local Development

# COMPONENT for dev/preflight: comma-separated frontend, backend (e.g. frontend,backend). Empty = port-forward only.
COMPONENT ?=
# When true, `make dev` runs `kind-up` without prompting if the cluster is missing.
AUTO_CLUSTER ?= false
# Backend URL for dev-env: use local go run (8080) vs port-forwarded cluster port.
DEV_BACKEND_LOCAL ?= false

preflight-cluster: ## Validate kind, kubectl, and container engine (daemon running)
	@echo "$(COLOR_BOLD)Preflight (cluster tools)$(COLOR_RESET)"
	@FAILED=0; \
	OS=$$(uname -s); \
	printf '%s\n' "---"; \
	if command -v kind >/dev/null 2>&1; then \
		KVER=$$(kind version -q 2>/dev/null || kind version 2>/dev/null | head -1); \
		echo "$(COLOR_GREEN)✓$(COLOR_RESET) kind $$KVER"; \
	else \
		echo "$(COLOR_RED)✗$(COLOR_RESET) kind not found"; \
		if [ "$$OS" = "Darwin" ]; then echo "  Install: brew install kind"; else echo "  Install: go install sigs.k8s.io/kind@latest"; fi; \
		echo "           https://kind.sigs.k8s.io/docs/user/quick-start/"; \
		FAILED=1; \
	fi; \
	if command -v kubectl >/dev/null 2>&1; then \
		echo "$(COLOR_GREEN)✓$(COLOR_RESET) kubectl $$(kubectl version --client -o yaml 2>/dev/null | grep gitVersion | head -1 | sed 's/.*: //' || kubectl version --client 2>/dev/null | head -1)"; \
	else \
		echo "$(COLOR_RED)✗$(COLOR_RESET) kubectl not found"; \
		if [ "$$OS" = "Darwin" ]; then echo "  Install: brew install kubectl"; else echo "  Install: https://kubernetes.io/docs/tasks/tools/"; fi; \
		FAILED=1; \
	fi; \
	CE="$(CONTAINER_ENGINE)"; \
	if [ "$$CE" = "podman" ]; then \
		if command -v podman >/dev/null 2>&1 && podman info >/dev/null 2>&1; then \
			echo "$(COLOR_GREEN)✓$(COLOR_RESET) podman $$(podman --version 2>/dev/null | head -1) (daemon running)"; \
		else \
			echo "$(COLOR_RED)✗$(COLOR_RESET) podman missing or daemon not running"; \
			if [ "$$OS" = "Darwin" ]; then echo "  Install: brew install podman && podman machine start"; else echo "  Install: https://podman.io/getting-started/installation"; fi; \
			FAILED=1; \
		fi; \
	else \
		if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then \
			echo "$(COLOR_GREEN)✓$(COLOR_RESET) docker $$(docker --version 2>/dev/null) (daemon running)"; \
		else \
			echo "$(COLOR_RED)✗$(COLOR_RESET) docker missing or daemon not running"; \
			if [ "$$OS" = "Darwin" ]; then echo "  Install: https://docs.docker.com/desktop/install/mac-install/"; else echo "  Install: https://docs.docker.com/engine/install/"; fi; \
			FAILED=1; \
		fi; \
	fi; \
	printf '%s\n' "---"; \
	if [ "$$FAILED" -ne 0 ]; then \
		echo "$(COLOR_RED)Preflight failed: fix the issues above.$(COLOR_RESET)"; \
		exit 1; \
	fi; \
	echo "$(COLOR_GREEN)✓$(COLOR_RESET) Cluster tool checks passed."

preflight: preflight-cluster ## Validate dev environment (cluster tools + optional Node/Go by COMPONENT)
	@echo "$(COLOR_BOLD)Preflight (language tools)$(COLOR_RESET)"
	@FAILED=0; \
	OS=$$(uname -s); \
	NEED_NODE=0; NEED_GO=0; \
	COMP="$(COMPONENT)"; \
	if [ -z "$$COMP" ]; then NEED_NODE=1; NEED_GO=1; \
	else \
		for piece in $$(echo "$$COMP" | tr ',' ' '); do \
			p=$$(echo "$$piece" | sed 's/^[[:space:]]*//;s/[[:space:]]*$$//'); \
			[ -z "$$p" ] && continue; \
			case "$$p" in \
				frontend) NEED_NODE=1 ;; \
				backend) NEED_GO=1 ;; \
				*) echo "$(COLOR_RED)✗$(COLOR_RESET) Unknown COMPONENT: $$p (use frontend, backend, or frontend,backend)"; FAILED=1 ;; \
			esac; \
		done; \
	fi; \
	if [ "$$NEED_NODE" -eq 1 ]; then \
		if command -v node >/dev/null 2>&1; then \
			NVER=$$(node -v 2>/dev/null | sed 's/^v//'); \
			NMAJ=$$(echo "$$NVER" | cut -d. -f1); \
			if [ "$${NMAJ:-0}" -ge 20 ] 2>/dev/null; then \
				echo "$(COLOR_GREEN)✓$(COLOR_RESET) node v$$NVER"; \
			else \
				echo "$(COLOR_RED)✗$(COLOR_RESET) node $$NVER (need >= 20)"; \
				if [ "$$OS" = "Darwin" ]; then echo "  Install: brew install node@20"; else echo "  Install: https://nodejs.org/ (LTS)"; fi; \
				FAILED=1; \
			fi; \
		else \
			echo "$(COLOR_RED)✗$(COLOR_RESET) node not found (need >= 20)"; \
			if [ "$$OS" = "Darwin" ]; then echo "  Install: brew install node@20"; else echo "  Install: https://nodejs.org/"; fi; \
			FAILED=1; \
		fi; \
		if command -v npm >/dev/null 2>&1; then \
			echo "$(COLOR_GREEN)✓$(COLOR_RESET) npm $$(npm -v)"; \
		else \
			echo "$(COLOR_RED)✗$(COLOR_RESET) npm not found"; \
			FAILED=1; \
		fi; \
	fi; \
	if [ "$$NEED_GO" -eq 1 ]; then \
		if command -v go >/dev/null 2>&1; then \
			GVER=$$(go env GOVERSION 2>/dev/null | sed 's/^go//'); \
			GMAJ=$$(echo "$$GVER" | cut -d. -f1); \
			GMIN=$$(echo "$$GVER" | cut -d. -f2); \
			if [ "$${GMAJ:-0}" -gt 1 ] || { [ "$${GMAJ:-0}" -eq 1 ] && [ "$${GMIN:-0}" -ge 21 ]; }; then \
				echo "$(COLOR_GREEN)✓$(COLOR_RESET) go $$GVER"; \
			else \
				echo "$(COLOR_RED)✗$(COLOR_RESET) go $$GVER (need >= 1.21)"; \
				if [ "$$OS" = "Darwin" ]; then echo "  Install: brew install go"; else echo "  Install: https://go.dev/dl/"; fi; \
				FAILED=1; \
			fi; \
		else \
			echo "$(COLOR_RED)✗$(COLOR_RESET) go not found (need >= 1.21)"; \
			if [ "$$OS" = "Darwin" ]; then echo "  Install: brew install go"; else echo "  Install: https://go.dev/dl/"; fi; \
			FAILED=1; \
		fi; \
	fi; \
	if [ "$$FAILED" -ne 0 ]; then \
		echo "$(COLOR_RED)Preflight failed: fix the issues above.$(COLOR_RESET)"; \
		exit 1; \
	fi; \
	echo "$(COLOR_GREEN)✓$(COLOR_RESET) Language tool checks passed."

dev-env: check-kubectl check-local-context ## Generate components/frontend/.env.local from cluster state (DEV_BACKEND_LOCAL=true for local backend on :8080)
	@set -e; \
	BACKEND_URL="http://localhost:$(KIND_FWD_BACKEND_PORT)/api"; \
	if [ "$(DEV_BACKEND_LOCAL)" = "true" ]; then BACKEND_URL="http://localhost:8080/api"; fi; \
	TOKEN=$$(kubectl get secret test-user-token -n $(NAMESPACE) -o jsonpath='{.data.token}' 2>/dev/null | base64 -d 2>/dev/null || true); \
	if [ -z "$$TOKEN" ]; then \
		echo "$(COLOR_YELLOW)⚠$(COLOR_RESET) test-user-token not found — OC_TOKEN left empty (run kind-up if cluster is new)"; \
	fi; \
	ENV_FILE="components/frontend/.env.local"; \
	{ \
		echo "# Generated by make dev-env — do not commit"; \
		echo "BACKEND_URL=$$BACKEND_URL"; \
		echo "ENABLE_OC_WHOAMI=0"; \
		if [ -n "$$TOKEN" ]; then echo "OC_TOKEN=$$TOKEN"; else echo "OC_TOKEN="; fi; \
	} > "$$ENV_FILE.tmp"; \
	if [ -f "$$ENV_FILE" ] && cmp -s "$$ENV_FILE.tmp" "$$ENV_FILE"; then \
		rm -f "$$ENV_FILE.tmp"; \
		echo "$(COLOR_GREEN)✓$(COLOR_RESET) $$ENV_FILE unchanged"; \
	else \
		mv "$$ENV_FILE.tmp" "$$ENV_FILE"; \
		echo "$(COLOR_GREEN)✓$(COLOR_RESET) Wrote $$ENV_FILE"; \
	fi

dev: ## Local dev: preflight, cluster, dev-env, port-forwards; COMPONENT=frontend|backend|frontend,backend for hot-reload
	@if [ -z "$(COMPONENT)" ]; then $(MAKE) --no-print-directory preflight-cluster; else $(MAKE) --no-print-directory preflight; fi
	@set -e; \
	if [ "$(CONTAINER_ENGINE)" = "podman" ]; then export KIND_EXPERIMENTAL_PROVIDER=podman; fi; \
	CLUSTER_RUNNING=0; \
	if kind get clusters 2>/dev/null | grep -q "^$(KIND_CLUSTER_NAME)$$"; then CLUSTER_RUNNING=1; fi; \
	if [ "$$CLUSTER_RUNNING" -eq 0 ]; then \
		if [ "$(AUTO_CLUSTER)" = "true" ]; then \
			echo "$(COLOR_BLUE)▶$(COLOR_RESET) AUTO_CLUSTER=true — running kind-up..."; \
			$(MAKE) kind-up CONTAINER_ENGINE=$(CONTAINER_ENGINE); \
		elif [ -t 0 ]; then \
			printf "Kind cluster '$(KIND_CLUSTER_NAME)' is not running. Run 'make kind-up' now? [y/N] "; \
			read -r _ans; \
			case "$$_ans" in y|Y|yes|YES) $(MAKE) kind-up CONTAINER_ENGINE=$(CONTAINER_ENGINE) ;; \
			*) echo "$(COLOR_RED)✗$(COLOR_RESET) Start the cluster first: $(COLOR_BOLD)make kind-up$(COLOR_RESET)"; exit 1 ;; esac; \
		else \
			echo "$(COLOR_RED)✗$(COLOR_RESET) Kind cluster '$(KIND_CLUSTER_NAME)' is not running."; \
			echo "  Run: $(COLOR_BOLD)make kind-up$(COLOR_RESET) or $(COLOR_BOLD)make dev AUTO_CLUSTER=true$(COLOR_RESET)"; \
			exit 1; \
		fi; \
	fi; \
	if [ "$(CONTAINER_ENGINE)" = "podman" ]; then \
		KIND_EXPERIMENTAL_PROVIDER=podman kubectl config use-context kind-$(KIND_CLUSTER_NAME) 2>/dev/null || \
			kubectl config use-context kind-$(KIND_CLUSTER_NAME); \
	else \
		kubectl config use-context kind-$(KIND_CLUSTER_NAME); \
	fi; \
	COMP="$(COMPONENT)"; \
	HAS_FRONT=0; HAS_BACK=0; \
	for piece in $$(echo "$$COMP" | tr ',' ' '); do \
		p=$$(echo "$$piece" | sed 's/^[[:space:]]*//;s/[[:space:]]*$$//'); \
		case "$$p" in frontend) HAS_FRONT=1 ;; backend) HAS_BACK=1 ;; esac; \
	done; \
	DEV_LOCAL=0; \
	if [ "$$HAS_FRONT" -eq 1 ] && [ "$$HAS_BACK" -eq 1 ]; then DEV_LOCAL=1; \
	elif [ "$$HAS_BACK" -eq 1 ] && [ "$$HAS_FRONT" -eq 0 ]; then DEV_LOCAL=1; \
	fi; \
	if [ -z "$$COMP" ]; then \
		$(MAKE) dev-env DEV_BACKEND_LOCAL=false; \
	else \
		$(MAKE) dev-env DEV_BACKEND_LOCAL=$$( [ "$$DEV_LOCAL" -eq 1 ] && echo true || echo false ); \
	fi; \
	echo ""; \
	echo "$(COLOR_BOLD)Access:$(COLOR_RESET)"; \
	echo "  Frontend: http://localhost:$(KIND_FWD_FRONTEND_PORT)"; \
	echo "  Backend:  http://localhost:$(KIND_FWD_BACKEND_PORT)"; \
	echo ""; \
	PF_PIDS=""; \
	cleanup() { \
		for pid in $$PF_PIDS; do kill "$$pid" 2>/dev/null || true; done; \
		echo ""; echo "$(COLOR_GREEN)✓$(COLOR_RESET) Stopped port-forward(s)."; \
	}; \
	trap cleanup INT TERM; \
	if [ -z "$$COMP" ]; then \
		echo "$(COLOR_BLUE)▶$(COLOR_RESET) Port-forwarding frontend + backend (Ctrl+C to stop)..."; \
		kubectl port-forward -n $(NAMESPACE) svc/frontend-service $(KIND_FWD_FRONTEND_PORT):3000 >/tmp/acp-dev-pf-frontend.log 2>&1 & PF_PIDS="$$PF_PIDS $$!"; \
		kubectl port-forward -n $(NAMESPACE) svc/backend-service $(KIND_FWD_BACKEND_PORT):8080 >/tmp/acp-dev-pf-backend.log 2>&1 & PF_PIDS="$$PF_PIDS $$!"; \
		echo "$(COLOR_GREEN)✓$(COLOR_RESET) Port-forwards running."; \
		echo "$(COLOR_YELLOW)Press Ctrl+C to stop.$(COLOR_RESET)"; \
		wait; \
	elif [ "$$HAS_FRONT" -eq 1 ] && [ "$$HAS_BACK" -eq 0 ]; then \
		echo "$(COLOR_BLUE)▶$(COLOR_RESET) Port-forward backend only; starting frontend dev..."; \
		kubectl port-forward -n $(NAMESPACE) svc/backend-service $(KIND_FWD_BACKEND_PORT):8080 >/tmp/acp-dev-pf-backend.log 2>&1 & PF_PIDS=$$!; \
		sleep 1; \
		cd components/frontend && npm run dev; \
	elif [ "$$HAS_BACK" -eq 1 ] && [ "$$HAS_FRONT" -eq 0 ]; then \
		echo "$(COLOR_BLUE)▶$(COLOR_RESET) Port-forward frontend only; starting backend dev..."; \
		kubectl port-forward -n $(NAMESPACE) svc/frontend-service $(KIND_FWD_FRONTEND_PORT):3000 >/tmp/acp-dev-pf-frontend.log 2>&1 & PF_PIDS=$$!; \
		sleep 1; \
		cd components/backend && go run .; \
	elif [ "$$HAS_FRONT" -eq 1 ] && [ "$$HAS_BACK" -eq 1 ]; then \
		echo "$(COLOR_BLUE)▶$(COLOR_RESET) Hot-reload: backend + frontend (local)..."; \
		(cd components/backend && go run .) & GO_PID=$$!; \
		(cd components/frontend && npm run dev) & NPM_PID=$$!; \
		trap 'kill $$GO_PID $$NPM_PID 2>/dev/null; cleanup' INT TERM; \
		wait $$GO_PID $$NPM_PID; \
	fi

##@ Benchmarking

benchmark: ## Run component benchmarks (COMPONENT=frontend MODE=cold|warm|both REPEATS=3)
	@bash scripts/benchmarks/component-bench.sh \
		$(if $(COMPONENT),--components $(COMPONENT)) \
		$(if $(MODE),--mode $(MODE)) \
		$(if $(REPEATS),--repeats $(REPEATS)) \
		$(if $(BASELINE),--baseline-ref $(BASELINE)) \
		$(if $(CANDIDATE),--candidate-ref $(CANDIDATE)) \
		$(if $(FORMAT),--format $(FORMAT))

benchmark-ci: ## Run component benchmarks in CI mode
	@bash scripts/benchmarks/component-bench.sh --ci \
		$(if $(COMPONENT),--components $(COMPONENT)) \
		$(if $(MODE),--mode $(MODE)) \
		$(if $(REPEATS),--repeats $(REPEATS)) \
		$(if $(BASELINE),--baseline-ref $(BASELINE)) \
		$(if $(CANDIDATE),--candidate-ref $(CANDIDATE)) \
		$(if $(FORMAT),--format $(FORMAT))

kind-up: preflight-cluster ## Start kind cluster and deploy the platform (LOCAL_IMAGES=true builds from source)
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Starting kind cluster '$(KIND_CLUSTER_NAME)'..."
	@cd e2e && KIND_CLUSTER_NAME=$(KIND_CLUSTER_NAME) KIND_HTTP_PORT=$(KIND_HTTP_PORT) KIND_HTTPS_PORT=$(KIND_HTTPS_PORT) KIND_HOST=$(KIND_HOST) CONTAINER_ENGINE=$(CONTAINER_ENGINE) ./scripts/setup-kind.sh
	@if [ -n "$(KIND_HOST)" ]; then \
		echo "$(COLOR_BLUE)▶$(COLOR_RESET) Rewriting kubeconfig for remote host $(KIND_HOST)..."; \
		SERVER=$$(kubectl config view -o jsonpath='{.clusters[?(@.name=="kind-$(KIND_CLUSTER_NAME)")].cluster.server}'); \
		FIXED=$$(echo "$$SERVER" | sed 's/127\.0\.0\.1/$(KIND_HOST)/; s/0\.0\.0\.0/$(KIND_HOST)/'); \
		kubectl config set-cluster kind-$(KIND_CLUSTER_NAME) --server="$$FIXED" >/dev/null; \
		echo "$(COLOR_GREEN)✓$(COLOR_RESET) API server: $$FIXED"; \
	fi
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Waiting for API server to be accessible..."
	@for i in 1 2 3 4 5 6 7 8 9 10; do \
		if kubectl cluster-info >/dev/null 2>&1; then \
			echo "$(COLOR_GREEN)✓$(COLOR_RESET) API server ready"; \
			break; \
		fi; \
		if [ $$i -eq 10 ]; then \
			echo "$(COLOR_RED)✗$(COLOR_RESET) Timeout waiting for API server"; \
			echo "   Try: kubectl cluster-info"; \
			exit 1; \
		fi; \
		sleep 3; \
	done
	@if [ "$(LOCAL_IMAGES)" = "true" ]; then \
		echo "$(COLOR_BLUE)▶$(COLOR_RESET) Building images from source..."; \
		$(MAKE) --no-print-directory build-all; \
		$(MAKE) --no-print-directory _kind-load-images; \
		echo "$(COLOR_BLUE)▶$(COLOR_RESET) Deploying with locally-built images..."; \
		kubectl apply --validate=false -k components/manifests/overlays/kind-local/; \
		echo "$(COLOR_BLUE)▶$(COLOR_RESET) Patching agent registry for local images..."; \
		REGISTRY=$$(kubectl get configmap ambient-agent-registry -n $(NAMESPACE) -o jsonpath='{.data.agent-registry\.json}'); \
		UPDATED=$$(echo "$$REGISTRY" | sed 's|quay.io/ambient_code/vteam_claude_runner:[^"]*|localhost/vteam_claude_runner:latest|g; s|quay.io/ambient_code/vteam_state_sync:[^"]*|localhost/vteam_state_sync:latest|g'); \
		kubectl patch configmap ambient-agent-registry -n $(NAMESPACE) --type=merge \
			-p "{\"data\":{\"agent-registry.json\":$$(echo "$$UPDATED" | jq -Rs .)}}"; \
		echo "$(COLOR_GREEN)✓$(COLOR_RESET) Agent registry patched for local images"; \
	else \
		echo "$(COLOR_BLUE)▶$(COLOR_RESET) Deploying with Quay.io images..."; \
		kubectl apply --validate=false -k components/manifests/overlays/kind/; \
	fi
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Waiting for pods..."
	@cd e2e && ./scripts/wait-for-ready.sh
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Initializing MinIO..."
	@cd e2e && ./scripts/init-minio.sh
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Extracting test token..."
	@cd e2e && KIND_CLUSTER_NAME=$(KIND_CLUSTER_NAME) KIND_HTTP_PORT=$(KIND_HTTP_PORT) CONTAINER_ENGINE=$(CONTAINER_ENGINE) ./scripts/extract-token.sh
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Kind cluster '$(KIND_CLUSTER_NAME)' ready!"
	@# Vertex AI setup if requested
	@if [ "$(LOCAL_VERTEX)" = "true" ]; then \
		echo "$(COLOR_BLUE)▶$(COLOR_RESET) Configuring Vertex AI..."; \
		ANTHROPIC_VERTEX_PROJECT_ID="$(ANTHROPIC_VERTEX_PROJECT_ID)" \
		CLOUD_ML_REGION="$(CLOUD_ML_REGION)" \
		GOOGLE_APPLICATION_CREDENTIALS="$(GOOGLE_APPLICATION_CREDENTIALS)" \
		./scripts/setup-vertex-kind.sh; \
	fi
	@if [ -f .dev-bootstrap.env ]; then \
		echo "$(COLOR_BLUE)▶$(COLOR_RESET) Bootstrapping developer workspace..."; \
		./scripts/bootstrap-workspace.sh || \
		echo "$(COLOR_YELLOW)⚠$(COLOR_RESET)  Bootstrap failed (non-fatal). Run 'make dev-bootstrap' manually."; \
	fi
	@echo ""
	@echo "$(COLOR_BOLD)Access the platform:$(COLOR_RESET)"
	@echo "  Cluster:  $(KIND_CLUSTER_NAME) (slug: $(CLUSTER_SLUG))"
	@echo "  Run in another terminal: $(COLOR_BLUE)make kind-port-forward$(COLOR_RESET)"
	@echo ""
	@echo "  Then access:"
	@echo "  Frontend: http://localhost:$(KIND_FWD_FRONTEND_PORT)"
	@echo "  Backend:  http://localhost:$(KIND_FWD_BACKEND_PORT)"
	@echo ""
	@echo "  Get test token: kubectl get secret test-user-token -n ambient-code -o jsonpath='{.data.token}' | base64 -d"
	@echo ""
	@echo "Run tests:"
	@echo "  make test-e2e"

kind-down: ## Stop and delete kind cluster
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Cleaning up kind cluster '$(KIND_CLUSTER_NAME)'..."
	@cd e2e && KIND_CLUSTER_NAME=$(KIND_CLUSTER_NAME) CONTAINER_ENGINE=$(CONTAINER_ENGINE) ./scripts/cleanup.sh
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Kind cluster '$(KIND_CLUSTER_NAME)' deleted"

kind-login: check-kubectl check-local-context ## Set kubectl context, port-forward services, configure acpctl, print test token
	@echo "$(COLOR_BOLD)Kind Login: $(KIND_CLUSTER_NAME)$(COLOR_RESET)"
	@echo ""
	@if [ "$(CONTAINER_ENGINE)" = "podman" ]; then \
		echo "using podman due to KIND_EXPERIMENTAL_PROVIDER"; \
		echo "enabling experimental podman provider"; \
		KIND_EXPERIMENTAL_PROVIDER=podman kubectl config use-context kind-$(KIND_CLUSTER_NAME) 2>/dev/null || \
			kubectl config use-context kind-$(KIND_CLUSTER_NAME); \
	else \
		kubectl config use-context kind-$(KIND_CLUSTER_NAME); \
	fi
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) kubeconfig set to kind-$(KIND_CLUSTER_NAME)"
	@echo ""
	@echo "Starting port-forwards..."
	@pkill -f "port-forward.*ambient-api-server-service" 2>/dev/null || true
	@pkill -f "port-forward.*frontend-service" 2>/dev/null || true
	@kubectl port-forward -n $(NAMESPACE) svc/ambient-api-server-service $(KIND_FWD_API_SERVER_PORT):8000 >/tmp/pf-api-server.log 2>&1 & \
		sleep 1; \
		echo "$(COLOR_GREEN)✓$(COLOR_RESET) ambient-api-server → http://localhost:$(KIND_FWD_API_SERVER_PORT)"
	@kubectl port-forward -n $(NAMESPACE) svc/frontend-service $(KIND_FWD_FRONTEND_PORT):3000 >/tmp/pf-frontend.log 2>&1 & \
		sleep 1; \
		echo "$(COLOR_GREEN)✓$(COLOR_RESET) frontend          → http://localhost:$(KIND_FWD_FRONTEND_PORT)"
	@echo ""
	@echo "Configuring acpctl..."
	@TOKEN=$$(kubectl get secret test-user-token -n $(NAMESPACE) -o jsonpath='{.data.token}' 2>/dev/null | base64 -d 2>/dev/null); \
	if [ -z "$$TOKEN" ]; then \
		echo "$(COLOR_YELLOW)Warning: test-user-token not found — acpctl not configured$(COLOR_RESET)"; \
	else \
		components/ambient-cli/acpctl login --url http://localhost:$(KIND_FWD_API_SERVER_PORT) --token "$$TOKEN" 2>/dev/null || \
			./acpctl login --url http://localhost:$(KIND_FWD_API_SERVER_PORT) --token "$$TOKEN" 2>/dev/null || \
			echo "$(COLOR_YELLOW)Warning: acpctl not built — run 'make build-cli' first$(COLOR_RESET)"; \
		echo "$(COLOR_GREEN)✓$(COLOR_RESET) acpctl configured: http://localhost:$(KIND_FWD_API_SERVER_PORT)"; \
		echo ""; \
		echo "Test token:"; \
		echo "$$TOKEN"; \
	fi

kind-port-forward: check-kubectl check-local-context ## Port-forward kind services (for remote Podman)
	@echo "$(COLOR_BOLD)Port forwarding kind services ($(KIND_CLUSTER_NAME))$(COLOR_RESET)"
	@echo ""
	@echo "  Frontend: http://localhost:$(KIND_FWD_FRONTEND_PORT)"
	@echo "  Backend:  http://localhost:$(KIND_FWD_BACKEND_PORT)"
	@echo ""
	@echo "$(COLOR_YELLOW)Press Ctrl+C to stop$(COLOR_RESET)"
	@echo ""
	@trap 'echo ""; echo "$(COLOR_GREEN)✓$(COLOR_RESET) Port forwarding stopped"; exit 0' INT; \
	(kubectl port-forward -n ambient-code svc/frontend-service $(KIND_FWD_FRONTEND_PORT):3000 >/dev/null 2>&1 &); \
	(kubectl port-forward -n ambient-code svc/backend-service $(KIND_FWD_BACKEND_PORT):8080 >/dev/null 2>&1 &); \
	wait

dev-bootstrap: check-kubectl check-local-context ## Bootstrap developer workspace with API key and integrations
	@./scripts/bootstrap-workspace.sh

##@ E2E Testing (Portable)

test-e2e: ## Run e2e tests against current CYPRESS_BASE_URL
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Running e2e tests..."
	@if [ ! -f e2e/.env.test ] && [ -z "$(CYPRESS_BASE_URL)" ] && [ -z "$(TEST_TOKEN)" ]; then \
		echo "$(COLOR_RED)✗$(COLOR_RESET) No .env.test found and environment variables not set"; \
		echo "   Option 1: Run 'make kind-up' first (creates .env.test)"; \
		echo "   Option 2: Set environment variables:"; \
		echo "     TEST_TOKEN=\$$(kubectl get secret test-user-token -n ambient-code -o jsonpath='{.data.token}' | base64 -d) \\"; \
		echo "     CYPRESS_BASE_URL=http://localhost:3000 \\"; \
		echo "     make test-e2e"; \
		exit 1; \
	fi
	cd e2e && CYPRESS_BASE_URL="$(CYPRESS_BASE_URL)" TEST_TOKEN="$(TEST_TOKEN)" ./scripts/run-tests.sh

test-e2e-local: ## Run complete e2e test suite with kind (setup, deploy, test, cleanup)
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Running e2e tests with kind (local)..."
	@$(MAKE) kind-up CONTAINER_ENGINE=$(CONTAINER_ENGINE)
	@cd e2e && trap 'KIND_CLUSTER_NAME=$(KIND_CLUSTER_NAME) CONTAINER_ENGINE=$(CONTAINER_ENGINE) ./scripts/cleanup.sh' EXIT; ./scripts/run-tests.sh

e2e-test: test-e2e-local ## Alias for test-e2e-local (backward compatibility)

test-e2e-setup: ## Install e2e test dependencies
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Installing e2e test dependencies..."
	cd e2e && npm install

e2e-setup: test-e2e-setup ## Alias for test-e2e-setup (backward compatibility)

kind-rebuild: check-kind check-kubectl check-local-context build-all ## Rebuild, reload, and restart all components in kind
	@$(if $(filter podman,$(CONTAINER_ENGINE)),KIND_EXPERIMENTAL_PROVIDER=podman) kind get clusters 2>/dev/null | grep -q '^$(KIND_CLUSTER_NAME)$$' || \
		(echo "$(COLOR_RED)✗$(COLOR_RESET) Kind cluster '$(KIND_CLUSTER_NAME)' not found. Run 'make kind-up LOCAL_IMAGES=true' first." && exit 1)
	@$(MAKE) --no-print-directory _kind-load-images
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Applying kind-local manifests..."
	@kubectl apply --validate=false -k components/manifests/overlays/kind-local/ $(QUIET_REDIRECT)
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Restarting deployments..."
	@kubectl rollout restart deployment -n $(NAMESPACE) $(QUIET_REDIRECT)
	@kubectl rollout status deployment -n $(NAMESPACE) --timeout=120s $(QUIET_REDIRECT)
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) All components rebuilt and restarted"

kind-reload-backend: check-kind check-kubectl check-local-context ## Rebuild and reload backend only (kind)
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Rebuilding backend..."
	@cd components/backend && $(CONTAINER_ENGINE) build $(PLATFORM_FLAG) \
		--build-arg AMBIENT_VERSION=$(shell git describe --tags --always --dirty) \
		-t $(BACKEND_IMAGE) . $(QUIET_REDIRECT)
	@$(CONTAINER_ENGINE) tag $(BACKEND_IMAGE) localhost/$(BACKEND_IMAGE) 2>/dev/null || true
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Loading image into kind cluster ($(KIND_CLUSTER_NAME))..."
	@$(CONTAINER_ENGINE) save localhost/$(BACKEND_IMAGE) | \
		$(CONTAINER_ENGINE) exec -i $(KIND_CLUSTER_NAME)-control-plane \
		ctr --namespace=k8s.io images import -
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Restarting backend..."
	@kubectl rollout restart deployment/backend-api -n $(NAMESPACE) $(QUIET_REDIRECT)
	@kubectl rollout status deployment/backend-api -n $(NAMESPACE) --timeout=60s
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Backend reloaded"

kind-reload-frontend: check-kind check-kubectl check-local-context ## Rebuild and reload frontend only (kind)
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Rebuilding frontend..."
	@cd components/frontend && $(CONTAINER_ENGINE) build $(PLATFORM_FLAG) \
		-t $(FRONTEND_IMAGE) . $(QUIET_REDIRECT)
	@$(CONTAINER_ENGINE) tag $(FRONTEND_IMAGE) localhost/$(FRONTEND_IMAGE) 2>/dev/null || true
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Loading image into kind cluster ($(KIND_CLUSTER_NAME))..."
	@$(CONTAINER_ENGINE) save localhost/$(FRONTEND_IMAGE) | \
		$(CONTAINER_ENGINE) exec -i $(KIND_CLUSTER_NAME)-control-plane \
		ctr --namespace=k8s.io images import -
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Restarting frontend..."
	@kubectl rollout restart deployment/frontend -n $(NAMESPACE) $(QUIET_REDIRECT)
	@kubectl rollout status deployment/frontend -n $(NAMESPACE) --timeout=60s
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Frontend reloaded"

kind-reload-operator: check-kind check-kubectl check-local-context ## Rebuild and reload operator only (kind)
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Rebuilding operator..."
	@cd components/operator && $(CONTAINER_ENGINE) build $(PLATFORM_FLAG) \
		-t $(OPERATOR_IMAGE) . $(QUIET_REDIRECT)
	@$(CONTAINER_ENGINE) tag $(OPERATOR_IMAGE) localhost/$(OPERATOR_IMAGE) 2>/dev/null || true
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Loading image into kind cluster ($(KIND_CLUSTER_NAME))..."
	@$(CONTAINER_ENGINE) save localhost/$(OPERATOR_IMAGE) | \
		$(CONTAINER_ENGINE) exec -i $(KIND_CLUSTER_NAME)-control-plane \
		ctr --namespace=k8s.io images import -
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Restarting operator..."
	@kubectl rollout restart deployment/agentic-operator -n $(NAMESPACE) $(QUIET_REDIRECT)
	@kubectl rollout status deployment/agentic-operator -n $(NAMESPACE) --timeout=60s
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Operator reloaded"

kind-status: check-kind ## Show all kind clusters and their port assignments
	@echo "$(COLOR_BOLD)Kind Cluster Status$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BOLD)Current worktree:$(COLOR_RESET)"
	@echo "  Slug:     $(CLUSTER_SLUG)"
	@echo "  Cluster:  $(KIND_CLUSTER_NAME)"
	@if [ -n "$(KIND_HOST)" ]; then echo "  Host:     $(KIND_HOST) (remote)"; else echo "  Host:     localhost"; fi
	@echo "  NodePort: $(KIND_HTTP_PORT) (HTTP) / $(KIND_HTTPS_PORT) (HTTPS)"
	@echo "  Forward:  $(KIND_FWD_FRONTEND_PORT) (frontend) / $(KIND_FWD_BACKEND_PORT) (backend)"
	@echo ""
	@CLUSTERS=$$($(if $(filter podman,$(CONTAINER_ENGINE)),KIND_EXPERIMENTAL_PROVIDER=podman) kind get clusters 2>/dev/null); \
	if [ -z "$$CLUSTERS" ]; then \
		echo "$(COLOR_YELLOW)No kind clusters running$(COLOR_RESET)"; \
	else \
		echo "$(COLOR_BOLD)Running clusters:$(COLOR_RESET)"; \
		echo "$$CLUSTERS" | while read -r cluster; do \
			if [ "$$cluster" = "$(KIND_CLUSTER_NAME)" ]; then \
				echo "  $(COLOR_GREEN)* $$cluster$(COLOR_RESET) (this worktree)"; \
			else \
				echo "    $$cluster"; \
			fi; \
		done; \
	fi

kind-clean: kind-down ## Alias for kind-down

e2e-clean: kind-down ## Alias for kind-down (backward compatibility)

deploy-langfuse-openshift: ## Deploy Langfuse to OpenShift/ROSA cluster
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Deploying Langfuse to OpenShift cluster..."
	@cd e2e && ./scripts/deploy-langfuse.sh --openshift

##@ Unleash Feature Flags
# Note: Unleash is deployed automatically via 'make deploy' as part of the platform manifests.
# Before deploying, create the unleash-credentials secret from the example:
#   cp components/manifests/base/unleash-credentials-secret.yaml.example unleash-credentials-secret.yaml
#   # Edit the file to set your credentials
#   kubectl apply -f unleash-credentials-secret.yaml -n ambient-code

unleash-port-forward: check-kubectl ## Port-forward Unleash (localhost:4242)
	@echo "$(COLOR_BOLD)🔌 Port forwarding Unleash$(COLOR_RESET)"
	@echo ""
	@echo "  Unleash UI: http://localhost:4242"
	@echo "  Login: admin / unleash4all"
	@echo ""
	@echo "$(COLOR_YELLOW)Press Ctrl+C to stop$(COLOR_RESET)"
	@kubectl port-forward svc/unleash 4242:4242 -n $${NAMESPACE:-ambient-code}

unleash-status: check-kubectl ## Show Unleash deployment status
	@echo "$(COLOR_BOLD)Unleash Status$(COLOR_RESET)"
	@kubectl get deployment,pod,svc -l 'app.kubernetes.io/name in (unleash,postgresql)' -n $${NAMESPACE:-ambient-code} 2>/dev/null || \
		echo "$(COLOR_RED)✗$(COLOR_RESET) Unleash not found. Run 'make deploy' first."

##@ Deprecated Aliases
# These targets preserve backward compatibility with the old minikube-based
# workflow.  Each prints a deprecation notice and delegates to the kind
# equivalent.  A follow-up issue tracks updating docs that still reference
# the old names.

local-up: ## Deprecated: use kind-up
	@echo "$(COLOR_YELLOW)Warning:$(COLOR_RESET) '$@' is deprecated. Use 'make kind-up' instead."
	@$(MAKE) --no-print-directory kind-up

local-clean: ## Deprecated: use kind-down
	@echo "$(COLOR_YELLOW)Warning:$(COLOR_RESET) '$@' is deprecated. Use 'make kind-down' instead."
	@$(MAKE) --no-print-directory kind-down

local-rebuild: ## Deprecated: use kind-rebuild
	@echo "$(COLOR_YELLOW)Warning:$(COLOR_RESET) '$@' is deprecated. Use 'make kind-rebuild' instead."
	@$(MAKE) --no-print-directory kind-rebuild

local-reload-backend: ## Deprecated: use kind-reload-backend
	@echo "$(COLOR_YELLOW)Warning:$(COLOR_RESET) '$@' is deprecated. Use 'make kind-reload-backend' instead."
	@$(MAKE) --no-print-directory kind-reload-backend

local-reload-frontend: ## Deprecated: use kind-reload-frontend
	@echo "$(COLOR_YELLOW)Warning:$(COLOR_RESET) '$@' is deprecated. Use 'make kind-reload-frontend' instead."
	@$(MAKE) --no-print-directory kind-reload-frontend

local-reload-operator: ## Deprecated: use kind-reload-operator
	@echo "$(COLOR_YELLOW)Warning:$(COLOR_RESET) '$@' is deprecated. Use 'make kind-reload-operator' instead."
	@$(MAKE) --no-print-directory kind-reload-operator

##@ Internal Helpers (do not call directly)

check-minikube: ## Check if minikube is installed
	@OS=$$(uname -s); \
	if command -v minikube >/dev/null 2>&1; then \
		echo "$(COLOR_GREEN)✓$(COLOR_RESET) minikube $$(minikube version 2>/dev/null | head -1)"; \
	else \
		echo "$(COLOR_RED)✗$(COLOR_RESET) minikube not found"; \
		if [ "$$OS" = "Darwin" ]; then echo "  Install: brew install minikube"; fi; \
		echo "  Install: https://minikube.sigs.k8s.io/docs/start/"; \
		exit 1; \
	fi

check-kind: ## Check if kind is installed
	@OS=$$(uname -s); \
	if command -v kind >/dev/null 2>&1; then \
		echo "$(COLOR_GREEN)✓$(COLOR_RESET) kind $$(kind version -q 2>/dev/null || kind version 2>/dev/null | head -1)"; \
	else \
		echo "$(COLOR_RED)✗$(COLOR_RESET) kind not found"; \
		if [ "$$OS" = "Darwin" ]; then echo "  Install: brew install kind"; else echo "  Install: go install sigs.k8s.io/kind@latest"; fi; \
		echo "  https://kind.sigs.k8s.io/docs/user/quick-start/"; \
		exit 1; \
	fi

check-kubectl: ## Check if kubectl is installed
	@OS=$$(uname -s); \
	if command -v kubectl >/dev/null 2>&1; then \
		echo "$(COLOR_GREEN)✓$(COLOR_RESET) kubectl $$(kubectl version --client -o yaml 2>/dev/null | grep gitVersion | head -1 | sed 's/.*: //' || kubectl version --client 2>/dev/null | head -1)"; \
	else \
		echo "$(COLOR_RED)✗$(COLOR_RESET) kubectl not found"; \
		if [ "$$OS" = "Darwin" ]; then echo "  Install: brew install kubectl"; fi; \
		echo "  Install: https://kubernetes.io/docs/tasks/tools/"; \
		exit 1; \
	fi

check-local-context: ## Verify kubectl context points to a local kind cluster
ifneq ($(SKIP_CONTEXT_CHECK),true)
	@ctx=$$(kubectl config current-context 2>/dev/null || echo ""); \
	if echo "$$ctx" | grep -qE '^kind-'; then \
		: ; \
	else \
		echo "$(COLOR_RED)✗$(COLOR_RESET) Current kubectl context '$$ctx' does not look like a local cluster."; \
		echo "  Expected a context starting with 'kind-'."; \
		echo "  Switch context first, e.g.: kubectl config use-context kind-ambient-local"; \
		echo ""; \
		echo "  To bypass this check: make <target> SKIP_CONTEXT_CHECK=true"; \
		exit 1; \
	fi
endif

check-architecture: ## Validate build architecture matches host
	@echo "$(COLOR_BOLD)Architecture Check$(COLOR_RESET)"
	@echo "  Host: $(HOST_OS) / $(HOST_ARCH)"
	@echo "  Detected Platform: $(DETECTED_PLATFORM)"
	@echo "  Active Platform: $(PLATFORM)"
	@if [ "$(PLATFORM)" != "$(DETECTED_PLATFORM)" ]; then \
		echo ""; \
		echo "$(COLOR_YELLOW)⚠  Cross-compilation active$(COLOR_RESET)"; \
		echo "   Building $(PLATFORM) images on $(DETECTED_PLATFORM) host"; \
		echo "   This will be slower (QEMU emulation)"; \
		echo ""; \
		echo "   To use native builds:"; \
		echo "     make build-all PLATFORM=$(DETECTED_PLATFORM)"; \
	else \
		echo "$(COLOR_GREEN)✓$(COLOR_RESET) Using native architecture"; \
	fi

_kind-load-images: ## Internal: Load images into kind cluster
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Loading images into kind ($(KIND_CLUSTER_NAME))..."
	@for img in $(BACKEND_IMAGE) $(FRONTEND_IMAGE) $(OPERATOR_IMAGE) $(RUNNER_IMAGE) $(STATE_SYNC_IMAGE) $(PUBLIC_API_IMAGE) $(API_SERVER_IMAGE); do \
		echo "  Loading $(KIND_IMAGE_PREFIX)$$img..."; \
		if [ -n "$(KIND_HOST)" ] || [ "$(CONTAINER_ENGINE)" = "podman" ]; then \
			$(CONTAINER_ENGINE) save $(KIND_IMAGE_PREFIX)$$img | \
			$(CONTAINER_ENGINE) exec -i $(KIND_CLUSTER_NAME)-control-plane \
			ctr --namespace=k8s.io images import -; \
		else \
			kind load docker-image $(KIND_IMAGE_PREFIX)$$img --name $(KIND_CLUSTER_NAME); \
		fi; \
	done
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Images loaded"

_restart-all: ## Internal: Restart all deployments
	@kubectl rollout restart deployment -n $(NAMESPACE) >/dev/null 2>&1
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Waiting for deployments to be ready..."
	@kubectl rollout status deployment -n $(NAMESPACE) --timeout=90s >/dev/null 2>&1 || true

_show-access-info: ## Internal: Show access information
	@echo "$(COLOR_BOLD)🌐 Access URLs:$(COLOR_RESET)"
	@echo "  Run: $(COLOR_BOLD)make kind-port-forward$(COLOR_RESET)"
	@echo "  Then access:"
	@echo "    Frontend: $(COLOR_BLUE)http://localhost:$(KIND_FWD_FRONTEND_PORT)$(COLOR_RESET)"
	@echo "    Backend:  $(COLOR_BLUE)http://localhost:$(KIND_FWD_BACKEND_PORT)$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_YELLOW)⚠  SECURITY NOTE:$(COLOR_RESET) Authentication is DISABLED for local development."

local-dev-token: check-kubectl ## Print a TokenRequest token for local-dev-user (for local dev API calls)
	@kubectl get serviceaccount local-dev-user -n $(NAMESPACE) >/dev/null 2>&1 || \
		(echo "$(COLOR_RED)✗$(COLOR_RESET) local-dev-user ServiceAccount not found in namespace $(NAMESPACE). Run 'make kind-up' first." && exit 1)
	@TOKEN=$$(kubectl -n $(NAMESPACE) create token local-dev-user 2>/dev/null); \
	if [ -z "$$TOKEN" ]; then \
		echo "$(COLOR_RED)✗$(COLOR_RESET) Failed to mint token (kubectl create token). Ensure TokenRequest is supported and kubectl is v1.24+"; \
		exit 1; \
	fi; \
	echo "$$TOKEN"

_create-operator-config: ## Internal: Create operator config from environment variables
	@VERTEX_PROJECT_ID=$${ANTHROPIC_VERTEX_PROJECT_ID:-""}; \
	VERTEX_KEY_FILE=$${GOOGLE_APPLICATION_CREDENTIALS:-""}; \
	ADC_FILE="$$HOME/.config/gcloud/application_default_credentials.json"; \
	CLOUD_REGION=$${CLOUD_ML_REGION:-"global"}; \
	USE_VERTEX="0"; \
	AUTH_METHOD="none"; \
	if [ -n "$$VERTEX_PROJECT_ID" ]; then \
		if [ -n "$$VERTEX_KEY_FILE" ] && [ -f "$$VERTEX_KEY_FILE" ]; then \
			USE_VERTEX="1"; \
			AUTH_METHOD="service-account"; \
			echo "  $(COLOR_GREEN)✓$(COLOR_RESET) Found Vertex AI config (service account)"; \
			echo "    Project: $$VERTEX_PROJECT_ID"; \
			echo "    Region: $$CLOUD_REGION"; \
			kubectl delete secret ambient-vertex -n $(NAMESPACE) 2>/dev/null || true; \
			kubectl create secret generic ambient-vertex \
				--from-file=ambient-code-key.json="$$VERTEX_KEY_FILE" \
				-n $(NAMESPACE) >/dev/null 2>&1; \
		elif [ -f "$$ADC_FILE" ]; then \
			USE_VERTEX="1"; \
			AUTH_METHOD="adc"; \
			echo "  $(COLOR_GREEN)✓$(COLOR_RESET) Found Vertex AI config (gcloud ADC)"; \
			echo "    Project: $$VERTEX_PROJECT_ID"; \
			echo "    Region: $$CLOUD_REGION"; \
			echo "    Using: Application Default Credentials"; \
			kubectl delete secret ambient-vertex -n $(NAMESPACE) 2>/dev/null || true; \
			kubectl create secret generic ambient-vertex \
				--from-file=ambient-code-key.json="$$ADC_FILE" \
				-n $(NAMESPACE) >/dev/null 2>&1; \
		else \
			echo "  $(COLOR_YELLOW)⚠$(COLOR_RESET)  ANTHROPIC_VERTEX_PROJECT_ID set but no credentials found"; \
			echo "    Run: gcloud auth application-default login"; \
			echo "    Or set: GOOGLE_APPLICATION_CREDENTIALS=/path/to/key.json"; \
			echo "    Using direct Anthropic API for now"; \
		fi; \
	else \
		echo "  $(COLOR_YELLOW)ℹ$(COLOR_RESET)  Vertex AI not configured"; \
		echo "    To enable: export ANTHROPIC_VERTEX_PROJECT_ID=your-project-id"; \
		echo "    Then run: gcloud auth application-default login"; \
		echo "    Using direct Anthropic API (provide ANTHROPIC_API_KEY in workspace settings)"; \
	fi; \
	kubectl create configmap operator-config -n $(NAMESPACE) \
		--from-literal=USE_VERTEX="$$USE_VERTEX" \
		--from-literal=CLOUD_ML_REGION="$$CLOUD_REGION" \
		--from-literal=ANTHROPIC_VERTEX_PROJECT_ID="$$VERTEX_PROJECT_ID" \
		--from-literal=GOOGLE_APPLICATION_CREDENTIALS="/app/vertex/ambient-code-key.json" \
		--dry-run=client -o yaml | kubectl apply -f - >/dev/null 2>&1

_auto-port-forward: ## Internal: Auto-start port forwarding on macOS with Podman
	@OS=$$(uname -s); \
	if [ "$$OS" = "Darwin" ] && [ "$(CONTAINER_ENGINE)" = "podman" ]; then \
		echo ""; \
		echo "$(COLOR_BLUE)▶$(COLOR_RESET) Starting port forwarding in background..."; \
		echo "  Waiting for services to be ready..."; \
		kubectl wait --for=condition=ready pod -l app=backend-api -n $(NAMESPACE) --timeout=60s 2>/dev/null || true; \
		kubectl wait --for=condition=ready pod -l app=frontend -n $(NAMESPACE) --timeout=60s 2>/dev/null || true; \
		mkdir -p /tmp/ambient-code; \
		kubectl port-forward -n $(NAMESPACE) svc/backend-service 8080:8080 > /tmp/ambient-code/port-forward-backend.log 2>&1 & \
		echo $$! > /tmp/ambient-code/port-forward-backend.pid; \
		kubectl port-forward -n $(NAMESPACE) svc/frontend-service 3000:3000 > /tmp/ambient-code/port-forward-frontend.log 2>&1 & \
		echo $$! > /tmp/ambient-code/port-forward-frontend.pid; \
		sleep 1; \
		if ps -p $$(cat /tmp/ambient-code/port-forward-backend.pid 2>/dev/null) > /dev/null 2>&1 && \
		   ps -p $$(cat /tmp/ambient-code/port-forward-frontend.pid 2>/dev/null) > /dev/null 2>&1; then \
			echo "$(COLOR_GREEN)✓$(COLOR_RESET) Port forwarding started"; \
			echo "  $(COLOR_BOLD)Access at:$(COLOR_RESET)"; \
			echo "    Frontend: $(COLOR_BLUE)http://localhost:3000$(COLOR_RESET)"; \
			echo "    Backend:  $(COLOR_BLUE)http://localhost:8080$(COLOR_RESET)"; \
		else \
			echo "$(COLOR_YELLOW)⚠$(COLOR_RESET)  Port forwarding started but may need time for pods"; \
			echo "  If connection fails, wait for pods and run: $(COLOR_BOLD)make local-port-forward$(COLOR_RESET)"; \
		fi; \
	fi

local-stop-port-forward: ## Stop background port forwarding
	@if [ -f /tmp/ambient-code/port-forward-backend.pid ]; then \
		echo "$(COLOR_BLUE)▶$(COLOR_RESET) Stopping port forwarding..."; \
		if ps -p $$(cat /tmp/ambient-code/port-forward-backend.pid 2>/dev/null) > /dev/null 2>&1; then \
			kill $$(cat /tmp/ambient-code/port-forward-backend.pid) 2>/dev/null || true; \
			echo "  Stopped backend port forward"; \
		fi; \
		if ps -p $$(cat /tmp/ambient-code/port-forward-frontend.pid 2>/dev/null) > /dev/null 2>&1; then \
			kill $$(cat /tmp/ambient-code/port-forward-frontend.pid) 2>/dev/null || true; \
			echo "  Stopped frontend port forward"; \
		fi; \
		rm -f /tmp/ambient-code/port-forward-*.pid /tmp/ambient-code/port-forward-*.log; \
		echo "$(COLOR_GREEN)✓$(COLOR_RESET) Port forwarding stopped"; \
	fi
