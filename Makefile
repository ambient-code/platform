.PHONY: help setup build-all build-frontend build-backend build-operator build-runner build-state-sync build-public-api build-control-plane build-cli deploy clean check-architecture deploy-openshift
.PHONY: local-up local-down local-clean local-status local-rebuild local-reload-backend local-reload-frontend local-reload-operator local-reload-api-server local-sync-version
.PHONY: local-dev-token
.PHONY: local-logs local-logs-backend local-logs-frontend local-logs-operator local-shell local-shell-frontend
.PHONY: local-test local-test-dev local-test-quick test-all local-url local-troubleshoot local-port-forward local-stop-port-forward
.PHONY: push-all registry-login setup-hooks remove-hooks lint check-minikube check-kind check-kubectl check-local-context dev-bootstrap kind-rebuild kind-status kind-login
.PHONY: e2e-test e2e-setup e2e-clean deploy-langfuse-openshift
.PHONY: unleash-port-forward unleash-status
.PHONY: setup-minio minio-console minio-logs minio-status
.PHONY: validate-makefile lint-makefile check-shell makefile-health
.PHONY: _create-operator-config _auto-port-forward _show-access-info _build-and-load _kind-load-images

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
CONTROL_PLANE_IMAGE ?= ambient_control_plane:$(IMAGE_TAG)

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
LOCAL_RUNNER ?= false
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
	@echo '  $(COLOR_GREEN)make local-up$(COLOR_RESET)            Start local development environment'
	@echo '  $(COLOR_GREEN)make local-status$(COLOR_RESET)        Check status of local environment'
	@echo '  $(COLOR_GREEN)make local-logs$(COLOR_RESET)          View logs from all components'
	@echo '  $(COLOR_GREEN)make local-down$(COLOR_RESET)          Stop local environment'
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
	@echo '  make kind-up LOCAL_IMAGES=true    Build from source and deploy to kind (requires podman)'
	@echo '  make kind-rebuild                 Rebuild and reload all components in kind'
	@echo '  make kind-status                  Show all kind clusters and their ports'
	@echo '  make local-up CONTAINER_ENGINE=docker'
	@echo '  make local-reload-backend'
	@echo '  make build-all PLATFORM=linux/arm64'

##@ Building

build-all: build-frontend build-backend build-operator build-runner build-state-sync build-public-api build-api-server build-control-plane ## Build all container images

build-frontend: ## Build frontend image
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Building frontend with $(CONTAINER_ENGINE)..."
	@cd components/frontend && $(CONTAINER_ENGINE) build $(PLATFORM_FLAG) $(BUILD_FLAGS) \
		-t $(FRONTEND_IMAGE) .
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Frontend built: $(FRONTEND_IMAGE)"

build-backend: ## Build backend image
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Building backend with $(CONTAINER_ENGINE)..."
	@cd components/backend && $(CONTAINER_ENGINE) build $(PLATFORM_FLAG) $(BUILD_FLAGS) \
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

build-control-plane: ## Build ambient-control-plane image
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Building ambient-control-plane with $(CONTAINER_ENGINE)..."
	@$(CONTAINER_ENGINE) build $(PLATFORM_FLAG) $(BUILD_FLAGS) \
		-t $(CONTROL_PLANE_IMAGE) \
		-f components/ambient-control-plane/Dockerfile \
		components/
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Control plane built: $(CONTROL_PLANE_IMAGE)"

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
	@for image in $(FRONTEND_IMAGE) $(BACKEND_IMAGE) $(OPERATOR_IMAGE) $(RUNNER_IMAGE) $(STATE_SYNC_IMAGE) $(PUBLIC_API_IMAGE) $(API_SERVER_IMAGE) $(CONTROL_PLANE_IMAGE); do \
		echo "  Tagging and pushing $$image..."; \
		$(CONTAINER_ENGINE) tag $$image $(REGISTRY)/$$image && \
		$(CONTAINER_ENGINE) push $(REGISTRY)/$$image; \
	done
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) All images pushed"

deploy-openshift: ## Deploy to OpenShift cluster. Use LOCAL_RUNNER=true to build+push runner from source to internal registry.
	@echo "$(COLOR_BOLD)Deploying to OpenShift cluster$(COLOR_RESET)"
	@oc whoami >/dev/null 2>&1 || (echo "$(COLOR_RED)✗$(COLOR_RESET) Not logged in to OpenShift. Run: oc login" && exit 1)
	@command -v ocm >/dev/null 2>&1 || (echo "$(COLOR_RED)✗$(COLOR_RESET) ocm CLI not found. Install from https://console.redhat.com/openshift/downloads and run: ocm login" && exit 1)
	@ocm token >/dev/null 2>&1 || (echo "$(COLOR_RED)✗$(COLOR_RESET) ocm token unavailable. Run: ocm login" && exit 1)
	@REGISTRY_HOST=$$(oc get route default-route -n openshift-image-registry --template='{{ .spec.host }}' 2>/dev/null) && \
	INTERNAL_REG="image-registry.openshift-image-registry.svc:5000/ambient-code" && \
	INTERNAL_RUNNER="$$INTERNAL_REG/vteam_claude_runner:latest" && \
	echo "$(COLOR_BLUE)▶$(COLOR_RESET) Registry: $$REGISTRY_HOST" && \
	oc whoami -t | $(CONTAINER_ENGINE) login --tls-verify=false -u kubeadmin --password-stdin "$$REGISTRY_HOST" && \
	echo "$(COLOR_BLUE)▶$(COLOR_RESET) Building and pushing control plane..." && \
	$(MAKE) --no-print-directory build-control-plane && \
	$(CONTAINER_ENGINE) tag $(CONTROL_PLANE_IMAGE) $$REGISTRY_HOST/ambient-code/$(CONTROL_PLANE_IMAGE) && \
	$(CONTAINER_ENGINE) push --tls-verify=false $$REGISTRY_HOST/ambient-code/$(CONTROL_PLANE_IMAGE) && \
	echo "$(COLOR_GREEN)✓$(COLOR_RESET) Pushed $(CONTROL_PLANE_IMAGE)" && \
	if [ "$(LOCAL_RUNNER)" = "true" ]; then \
		echo "$(COLOR_BLUE)▶$(COLOR_RESET) LOCAL_RUNNER=true: building and pushing runner from source..." && \
		$(MAKE) --no-print-directory build-runner && \
		$(CONTAINER_ENGINE) tag $(RUNNER_IMAGE) $$REGISTRY_HOST/ambient-code/$(RUNNER_IMAGE) && \
		$(CONTAINER_ENGINE) push --tls-verify=false $$REGISTRY_HOST/ambient-code/$(RUNNER_IMAGE) && \
		echo "$(COLOR_GREEN)✓$(COLOR_RESET) Pushed $(RUNNER_IMAGE)"; \
	else \
		echo "$(COLOR_BLUE)▶$(COLOR_RESET) Using quay.io runner image (pass LOCAL_RUNNER=true to build from source)"; \
		INTERNAL_RUNNER="quay.io/ambient_code/vteam_claude_runner:latest"; \
	fi && \
	echo "$(COLOR_BLUE)▶$(COLOR_RESET) Applying production manifests..." && \
	kubectl kustomize components/manifests/overlays/production/ | kubectl apply --validate=false -f - && \
	echo "$(COLOR_BLUE)▶$(COLOR_RESET) Patching runner image to: $$INTERNAL_RUNNER" && \
	kubectl set env deployment/ambient-control-plane -n ambient-code RUNNER_IMAGE="$$INTERNAL_RUNNER" && \
	REGISTRY_JSON=$$(oc get configmap ambient-agent-registry -n ambient-code -o jsonpath='{.data.agent-registry\.json}') && \
	UPDATED_JSON=$$(echo "$$REGISTRY_JSON" | sed "s|quay\.io/ambient_code/vteam_claude_runner:[^\"]*|$$INTERNAL_RUNNER|g") && \
	kubectl patch configmap ambient-agent-registry -n ambient-code --type=merge \
		-p "{\"data\":{\"agent-registry.json\":$$(echo "$$UPDATED_JSON" | python3 -c 'import sys,json; print(json.dumps(sys.stdin.read()))')}}" && \
	echo "$(COLOR_GREEN)✓$(COLOR_RESET) Agent registry updated" && \
	echo "$(COLOR_BLUE)▶$(COLOR_RESET) Provisioning control plane RH SSO token..." && \
	oc delete secret ambient-control-plane-token -n ambient-code 2>/dev/null || true && \
	oc create secret generic ambient-control-plane-token -n ambient-code \
		--from-literal=token="$$(ocm token)" && \
	echo "$(COLOR_BLUE)▶$(COLOR_RESET) Restarting control-plane deployment..." && \
	oc rollout restart deployment/ambient-control-plane -n ambient-code && \
	oc rollout status deployment/ambient-control-plane -n ambient-code --timeout=120s && \
	echo "$(COLOR_GREEN)✓$(COLOR_RESET) OpenShift deployment complete"

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

##@ Local Development (Minikube)

local-up: check-minikube check-kubectl ## Start local development environment (minikube)
	@echo "$(COLOR_BOLD)🚀 Starting Ambient Code Platform Local Environment$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Step 1/8: Starting minikube..."
	@if [ "$(CONTAINER_ENGINE)" = "docker" ]; then \
		minikube start --driver=docker --memory=4096 --cpus=2 $(QUIET_REDIRECT) || \
			(minikube status >/dev/null 2>&1 && echo "$(COLOR_GREEN)✓$(COLOR_RESET) Minikube already running") || \
			(echo "$(COLOR_RED)✗$(COLOR_RESET) Failed to start minikube" && exit 1); \
	else \
		minikube start --driver=podman --memory=4096 --cpus=2 --kubernetes-version=v1.35.0 --container-runtime=cri-o $(QUIET_REDIRECT) || \
			(minikube status >/dev/null 2>&1 && echo "$(COLOR_GREEN)✓$(COLOR_RESET) Minikube already running") || \
			(echo "$(COLOR_RED)✗$(COLOR_RESET) Failed to start minikube" && exit 1); \
	fi
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Step 2/8: Enabling addons..."
	@minikube addons enable ingress $(QUIET_REDIRECT) || true
	@minikube addons enable storage-provisioner $(QUIET_REDIRECT) || true
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Step 3/8: Building images..."
	@$(MAKE) --no-print-directory _build-and-load
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Step 4/8: Creating namespace..."
	@kubectl create namespace $(NAMESPACE) --dry-run=client -o yaml | kubectl apply -f - $(QUIET_REDIRECT)
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Step 5/8: Applying CRDs and RBAC..."
	@kubectl apply -f components/manifests/base/crds/ $(QUIET_REDIRECT) || true
	@kubectl apply -f components/manifests/base/rbac/ $(QUIET_REDIRECT) || true
	@kubectl apply -f components/manifests/minikube/local-dev-rbac.yaml $(QUIET_REDIRECT) || true
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Step 6/8: Creating storage..."
	@kubectl apply -f components/manifests/base/workspace-pvc.yaml -n $(NAMESPACE) $(QUIET_REDIRECT) || true
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Step 6.5/8: Configuring operator..."
	@$(MAKE) --no-print-directory _create-operator-config
	@$(MAKE) --no-print-directory local-sync-version
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Step 7/8: Deploying services..."
	@kubectl apply -f components/manifests/minikube/backend-deployment.yaml $(QUIET_REDIRECT)
	@kubectl apply -f components/manifests/minikube/backend-service.yaml $(QUIET_REDIRECT)
	@kubectl apply -f components/manifests/minikube/frontend-deployment.yaml $(QUIET_REDIRECT)
	@kubectl apply -f components/manifests/minikube/frontend-service.yaml $(QUIET_REDIRECT)
	@kubectl apply -f components/manifests/minikube/operator-deployment.yaml $(QUIET_REDIRECT)
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Step 8/8: Setting up ingress..."
	@kubectl wait --namespace ingress-nginx --for=condition=ready pod \
		--selector=app.kubernetes.io/component=controller --timeout=90s >/dev/null 2>&1 || true
	@kubectl apply -f components/manifests/minikube/ingress.yaml $(QUIET_REDIRECT) || true
	@echo ""
	@echo "$(COLOR_GREEN)✓ Ambient Code Platform is starting up!$(COLOR_RESET)"
	@echo ""
	@$(MAKE) --no-print-directory _show-access-info
	@$(MAKE) --no-print-directory _auto-port-forward
	@echo ""
	@echo "$(COLOR_YELLOW)⚠  Next steps:$(COLOR_RESET)"
	@echo "  • Wait ~30s for pods to be ready"
	@echo "  • Run: $(COLOR_BOLD)make local-status$(COLOR_RESET) to check deployment"
	@echo "  • Run: $(COLOR_BOLD)make local-logs$(COLOR_RESET) to view logs"

local-down: check-kubectl check-local-context ## Stop Ambient Code Platform (keep minikube running)
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Stopping Ambient Code Platform..."
	@$(MAKE) --no-print-directory local-stop-port-forward
	@kubectl delete namespace $(NAMESPACE) --ignore-not-found=true --timeout=60s
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Ambient Code Platform stopped (minikube still running)"
	@echo "  To stop minikube: $(COLOR_BOLD)make local-clean$(COLOR_RESET)"

local-clean: check-minikube ## Delete minikube cluster completely
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Deleting minikube cluster..."
	@$(MAKE) --no-print-directory local-stop-port-forward
	@minikube delete
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Minikube cluster deleted"

local-status: check-kubectl ## Show status of local deployment
	@echo "$(COLOR_BOLD)📊 Ambient Code Platform Status$(COLOR_RESET)"
	@echo ""
	@if $(if $(filter podman,$(CONTAINER_ENGINE)),KIND_EXPERIMENTAL_PROVIDER=podman) kind get clusters 2>/dev/null | grep -q '^$(KIND_CLUSTER_NAME)$$'; then \
		echo "$(COLOR_BOLD)Kind:$(COLOR_RESET)"; \
		echo "$(COLOR_GREEN)✓$(COLOR_RESET) Cluster '$(KIND_CLUSTER_NAME)' running"; \
	elif command -v minikube >/dev/null 2>&1; then \
		echo "$(COLOR_BOLD)Minikube:$(COLOR_RESET)"; \
		minikube status 2>/dev/null || echo "$(COLOR_RED)✗$(COLOR_RESET) Minikube not running"; \
	else \
		echo "$(COLOR_RED)✗$(COLOR_RESET) No local cluster found (kind or minikube)"; \
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
	else \
		$(MAKE) --no-print-directory _show-access-info; \
	fi

local-sync-version: ## Sync version from git to local deployment manifests
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Syncing version from git..."
	@VERSION=$$(git describe --tags --always 2>/dev/null || echo "dev") && \
	echo "  Using version: $$VERSION" && \
	sed -i.bak "s|value: \"v.*\"|value: \"$$VERSION\"|" \
	components/manifests/minikube/frontend-deployment.yaml && \
	rm -f components/manifests/minikube/frontend-deployment.yaml.bak && \
	echo "  $(COLOR_GREEN)✓$(COLOR_RESET) Version synced to $$VERSION"

local-rebuild: check-local-context ## Rebuild and reload all components
	@echo "$(COLOR_BOLD)🔄 Rebuilding all components...$(COLOR_RESET)"
	@$(MAKE) --no-print-directory _build-and-load
	@$(MAKE) --no-print-directory _restart-all
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) All components rebuilt and reloaded"

local-reload-backend: check-local-context ## Rebuild and reload backend only
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Rebuilding backend..."
	@cd components/backend && $(CONTAINER_ENGINE) build -t $(BACKEND_IMAGE) . >/dev/null 2>&1
	@$(CONTAINER_ENGINE) tag $(BACKEND_IMAGE) localhost/$(BACKEND_IMAGE) 2>/dev/null || true
	@$(CONTAINER_ENGINE) save -o /tmp/backend-reload.tar localhost/$(BACKEND_IMAGE)
	@minikube image load /tmp/backend-reload.tar >/dev/null 2>&1
	@rm -f /tmp/backend-reload.tar
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Restarting backend..."
	@kubectl rollout restart deployment/backend-api -n $(NAMESPACE) >/dev/null 2>&1
	@kubectl rollout status deployment/backend-api -n $(NAMESPACE) --timeout=60s
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Backend reloaded"
	@OS=$$(uname -s); \
	if [ "$$OS" = "Darwin" ] && [ "$(CONTAINER_ENGINE)" = "podman" ]; then \
		echo "$(COLOR_BLUE)▶$(COLOR_RESET) Restarting backend port forward..."; \
		if [ -f /tmp/ambient-code/port-forward-backend.pid ]; then \
			kill $$(cat /tmp/ambient-code/port-forward-backend.pid) 2>/dev/null || true; \
		fi; \
		kubectl port-forward -n $(NAMESPACE) svc/backend-service 8080:8080 > /tmp/ambient-code/port-forward-backend.log 2>&1 & \
		echo $$! > /tmp/ambient-code/port-forward-backend.pid; \
		sleep 2; \
		echo "$(COLOR_GREEN)✓$(COLOR_RESET) Backend port forward restarted"; \
	fi

local-reload-frontend: check-local-context ## Rebuild and reload frontend only
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Rebuilding frontend..."
	@cd components/frontend && $(CONTAINER_ENGINE) build -t $(FRONTEND_IMAGE) . >/dev/null 2>&1
	@$(CONTAINER_ENGINE) tag $(FRONTEND_IMAGE) localhost/$(FRONTEND_IMAGE) 2>/dev/null || true
	@$(CONTAINER_ENGINE) save -o /tmp/frontend-reload.tar localhost/$(FRONTEND_IMAGE)
	@minikube image load /tmp/frontend-reload.tar >/dev/null 2>&1
	@rm -f /tmp/frontend-reload.tar
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Restarting frontend..."
	@kubectl rollout restart deployment/frontend -n $(NAMESPACE) >/dev/null 2>&1
	@kubectl rollout status deployment/frontend -n $(NAMESPACE) --timeout=60s
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Frontend reloaded"
	@OS=$$(uname -s); \
	if [ "$$OS" = "Darwin" ] && [ "$(CONTAINER_ENGINE)" = "podman" ]; then \
		echo "$(COLOR_BLUE)▶$(COLOR_RESET) Restarting frontend port forward..."; \
		if [ -f /tmp/ambient-code/port-forward-frontend.pid ]; then \
			kill $$(cat /tmp/ambient-code/port-forward-frontend.pid) 2>/dev/null || true; \
		fi; \
		kubectl port-forward -n $(NAMESPACE) svc/frontend-service 3000:3000 > /tmp/ambient-code/port-forward-frontend.log 2>&1 & \
		echo $$! > /tmp/ambient-code/port-forward-frontend.pid; \
		sleep 2; \
		echo "$(COLOR_GREEN)✓$(COLOR_RESET) Frontend port forward restarted"; \
	fi


local-reload-operator: check-local-context ## Rebuild and reload operator only
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Rebuilding operator..."
	@cd components/operator && $(CONTAINER_ENGINE) build -t $(OPERATOR_IMAGE) . >/dev/null 2>&1
	@$(CONTAINER_ENGINE) tag $(OPERATOR_IMAGE) localhost/$(OPERATOR_IMAGE) 2>/dev/null || true
	@$(CONTAINER_ENGINE) save -o /tmp/operator-reload.tar localhost/$(OPERATOR_IMAGE)
	@minikube image load /tmp/operator-reload.tar >/dev/null 2>&1
	@rm -f /tmp/operator-reload.tar
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Restarting operator..."
	@kubectl rollout restart deployment/agentic-operator -n $(NAMESPACE) >/dev/null 2>&1
	@kubectl rollout status deployment/agentic-operator -n $(NAMESPACE) --timeout=60s
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Operator reloaded"

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

makefile-health: check-minikube check-kubectl ## Run comprehensive Makefile health check
	@echo "$(COLOR_BOLD)🏥 Makefile Health Check$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BOLD)Prerequisites:$(COLOR_RESET)"
	@minikube version >/dev/null 2>&1 && echo "$(COLOR_GREEN)✓$(COLOR_RESET) minikube available" || echo "$(COLOR_RED)✗$(COLOR_RESET) minikube missing"
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

local-test-quick: check-kubectl ## Quick smoke test of local environment (kind or minikube)
	@echo "$(COLOR_BOLD)🧪 Quick Smoke Test$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Detecting cluster type..."
	@if kind get clusters 2>/dev/null | grep -q .; then \
		echo "$(COLOR_GREEN)✓$(COLOR_RESET) Kind cluster running"; \
		CLUSTER_TYPE=kind; \
	elif command -v minikube >/dev/null 2>&1 && minikube status >/dev/null 2>&1; then \
		echo "$(COLOR_GREEN)✓$(COLOR_RESET) Minikube running"; \
		CLUSTER_TYPE=minikube; \
	else \
		echo "$(COLOR_RED)✗$(COLOR_RESET) No local cluster found (kind or minikube)"; exit 1; \
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

local-url: check-minikube ## Display access URLs
	@$(MAKE) --no-print-directory _show-access-info

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

kind-up: check-kind check-kubectl ## Start kind cluster (LOCAL_IMAGES=true to build from source, requires podman)
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

kind-status: ## Show all kind clusters and their port assignments
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

##@ Internal Helpers (do not call directly)

check-minikube: ## Check if minikube is installed
	@command -v minikube >/dev/null 2>&1 || \
		(echo "$(COLOR_RED)✗$(COLOR_RESET) minikube not found. Install: https://minikube.sigs.k8s.io/docs/start/" && exit 1)

check-kind: ## Check if kind is installed
	@command -v kind >/dev/null 2>&1 || \
		(echo "$(COLOR_RED)✗$(COLOR_RESET) kind not found. Install: https://kind.sigs.k8s.io/docs/user/quick-start/" && exit 1)

check-kubectl: ## Check if kubectl is installed
	@command -v kubectl >/dev/null 2>&1 || \
		(echo "$(COLOR_RED)✗$(COLOR_RESET) kubectl not found. Install: https://kubernetes.io/docs/tasks/tools/" && exit 1)

check-local-context: ## Verify kubectl context points to a local cluster (kind or minikube)
ifneq ($(SKIP_CONTEXT_CHECK),true)
	@ctx=$$(kubectl config current-context 2>/dev/null || echo ""); \
	if echo "$$ctx" | grep -qE '^(kind-|minikube$$)'; then \
		: ; \
	else \
		echo "$(COLOR_RED)✗$(COLOR_RESET) Current kubectl context '$$ctx' does not look like a local cluster."; \
		echo "  Expected a context starting with 'kind-' or named 'minikube'."; \
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
	@for img in $(BACKEND_IMAGE) $(FRONTEND_IMAGE) $(OPERATOR_IMAGE) $(RUNNER_IMAGE) $(STATE_SYNC_IMAGE) $(PUBLIC_API_IMAGE) $(API_SERVER_IMAGE) $(CONTROL_PLANE_IMAGE); do \
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

_build-and-load: ## Internal: Build and load images
	@echo "  Building backend ($(PLATFORM))..."
	@$(CONTAINER_ENGINE) build $(PLATFORM_FLAG) -t $(BACKEND_IMAGE) components/backend $(QUIET_REDIRECT)
	@echo "  Building frontend ($(PLATFORM))..."
	@$(CONTAINER_ENGINE) build $(PLATFORM_FLAG) -t $(FRONTEND_IMAGE) components/frontend $(QUIET_REDIRECT)
	@echo "  Building operator ($(PLATFORM))..."
	@$(CONTAINER_ENGINE) build $(PLATFORM_FLAG) -t $(OPERATOR_IMAGE) components/operator $(QUIET_REDIRECT)
	@echo "  Building runner ($(PLATFORM))..."
	@$(CONTAINER_ENGINE) build $(PLATFORM_FLAG) -t $(RUNNER_IMAGE) -f components/runners/ambient-runner/Dockerfile components/runners $(QUIET_REDIRECT)
	@echo "  Building api-server ($(PLATFORM))..."
	@$(CONTAINER_ENGINE) build $(PLATFORM_FLAG) -t $(API_SERVER_IMAGE) components/ambient-api-server $(QUIET_REDIRECT)
	@echo "  Tagging images with localhost prefix..."
	@$(CONTAINER_ENGINE) tag $(BACKEND_IMAGE) localhost/$(BACKEND_IMAGE) 2>/dev/null || true
	@$(CONTAINER_ENGINE) tag $(FRONTEND_IMAGE) localhost/$(FRONTEND_IMAGE) 2>/dev/null || true
	@$(CONTAINER_ENGINE) tag $(OPERATOR_IMAGE) localhost/$(OPERATOR_IMAGE) 2>/dev/null || true
	@$(CONTAINER_ENGINE) tag $(RUNNER_IMAGE) localhost/$(RUNNER_IMAGE) 2>/dev/null || true
	@$(CONTAINER_ENGINE) tag $(API_SERVER_IMAGE) localhost/$(API_SERVER_IMAGE) 2>/dev/null || true
	@echo "  Loading images into minikube..."
	@mkdir -p /tmp/minikube-images
	@$(CONTAINER_ENGINE) save -o /tmp/minikube-images/backend.tar localhost/$(BACKEND_IMAGE)
	@$(CONTAINER_ENGINE) save -o /tmp/minikube-images/frontend.tar localhost/$(FRONTEND_IMAGE)
	@$(CONTAINER_ENGINE) save -o /tmp/minikube-images/operator.tar localhost/$(OPERATOR_IMAGE)
	@$(CONTAINER_ENGINE) save -o /tmp/minikube-images/runner.tar localhost/$(RUNNER_IMAGE)
	@$(CONTAINER_ENGINE) save -o /tmp/minikube-images/api-server.tar localhost/$(API_SERVER_IMAGE)
	@minikube image load /tmp/minikube-images/backend.tar $(QUIET_REDIRECT)
	@minikube image load /tmp/minikube-images/frontend.tar $(QUIET_REDIRECT)
	@minikube image load /tmp/minikube-images/operator.tar $(QUIET_REDIRECT)
	@minikube image load /tmp/minikube-images/runner.tar $(QUIET_REDIRECT)
	@minikube image load /tmp/minikube-images/api-server.tar $(QUIET_REDIRECT)
	@rm -rf /tmp/minikube-images
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Images built and loaded"

_restart-all: ## Internal: Restart all deployments
	@kubectl rollout restart deployment -n $(NAMESPACE) >/dev/null 2>&1
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Waiting for deployments to be ready..."
	@kubectl rollout status deployment -n $(NAMESPACE) --timeout=90s >/dev/null 2>&1 || true

_show-access-info: ## Internal: Show access information
	@echo "$(COLOR_BOLD)🌐 Access URLs:$(COLOR_RESET)"
	@OS=$$(uname -s); \
	if [ "$$OS" = "Darwin" ] && [ "$(CONTAINER_ENGINE)" = "podman" ]; then \
		echo "  $(COLOR_YELLOW)Note:$(COLOR_RESET) Port forwarding will start automatically"; \
		echo "  Once pods are ready, access at:"; \
		echo "     Frontend: $(COLOR_BLUE)http://localhost:3000$(COLOR_RESET)"; \
		echo "     Backend:  $(COLOR_BLUE)http://localhost:8080$(COLOR_RESET)"; \
		echo ""; \
		echo "  $(COLOR_BOLD)To manage port forwarding:$(COLOR_RESET)"; \
		echo "    Stop:    $(COLOR_BOLD)make local-stop-port-forward$(COLOR_RESET)"; \
		echo "    Restart: $(COLOR_BOLD)make local-port-forward$(COLOR_RESET)"; \
	else \
		MINIKUBE_IP=$$(minikube ip 2>/dev/null) && \
			echo "  Frontend: $(COLOR_BLUE)http://$$MINIKUBE_IP:30030$(COLOR_RESET)" && \
			echo "  Backend:  $(COLOR_BLUE)http://$$MINIKUBE_IP:30080$(COLOR_RESET)" || \
			echo "  $(COLOR_RED)✗$(COLOR_RESET) Cannot get minikube IP"; \
		echo ""; \
		echo "$(COLOR_BOLD)Alternative:$(COLOR_RESET) Port forward for localhost access"; \
		echo "  Run: $(COLOR_BOLD)make local-port-forward$(COLOR_RESET)"; \
		echo "  Then access:"; \
		echo "    Frontend: $(COLOR_BLUE)http://localhost:3000$(COLOR_RESET)"; \
		echo "    Backend:  $(COLOR_BLUE)http://localhost:8080$(COLOR_RESET)"; \
	fi
	@echo ""
	@echo "$(COLOR_YELLOW)⚠  SECURITY NOTE:$(COLOR_RESET) Authentication is DISABLED for local development."

local-dev-token: check-kubectl ## Print a TokenRequest token for local-dev-user (for local dev API calls)
	@kubectl get serviceaccount local-dev-user -n $(NAMESPACE) >/dev/null 2>&1 || \
		(echo "$(COLOR_RED)✗$(COLOR_RESET) local-dev-user ServiceAccount not found in namespace $(NAMESPACE). Run 'make local-up' first." && exit 1)
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
		kubectl wait --for=condition=ready pod -l app=backend -n $(NAMESPACE) --timeout=60s 2>/dev/null || true; \
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
