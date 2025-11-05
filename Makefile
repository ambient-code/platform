.PHONY: help setup-env build-all build-frontend build-backend build-operator build-runner deploy clean dev-frontend dev-backend lint test registry-login push-all dev-start dev-stop dev-test dev-logs-operator dev-restart-operator dev-operator-status dev-test-operator e2e-test e2e-setup e2e-clean setup-hooks remove-hooks deploy-langfuse-openshift

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Configuration Variables:'
	@echo '  CONTAINER_ENGINE   Container engine to use (default: docker, can be set to podman)'
	@echo '  PLATFORM           Target platform (e.g., linux/amd64, linux/arm64)'
	@echo '  BUILD_FLAGS        Additional flags to pass to build command'
	@echo '  REGISTRY           Container registry for push operations'
	@echo ''
	@echo 'Examples:'
	@echo '  make build-all CONTAINER_ENGINE=podman'
	@echo '  make build-all PLATFORM=linux/amd64'
	@echo '  make build-all BUILD_FLAGS="--no-cache --pull"'
	@echo '  make build-all CONTAINER_ENGINE=podman PLATFORM=linux/arm64'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Container engine configuration
CONTAINER_ENGINE ?= docker
PLATFORM ?= linux/amd64
BUILD_FLAGS ?= 


# Construct platform flag if PLATFORM is set
ifneq ($(PLATFORM),)
PLATFORM_FLAG := --platform=$(PLATFORM)
else
PLATFORM_FLAG := 
endif

# Docker image tags
FRONTEND_IMAGE ?= vteam_frontend:latest
BACKEND_IMAGE ?= vteam_backend:latest
OPERATOR_IMAGE ?= vteam_operator:latest
RUNNER_IMAGE ?= vteam_claude_runner:latest

# Docker registry operations (customize REGISTRY as needed)
REGISTRY ?= your-registry.com

# Build all images
build-all: build-frontend build-backend build-operator build-runner ## Build all container images

# Build individual components
build-frontend: ## Build the frontend container image
	@echo "Building frontend image with $(CONTAINER_ENGINE)..."
	cd components/frontend && $(CONTAINER_ENGINE) build $(PLATFORM_FLAG) $(BUILD_FLAGS) -t $(FRONTEND_IMAGE) .

build-backend: ## Build the backend API container image
	@echo "Building backend image with $(CONTAINER_ENGINE)..."
	cd components/backend && $(CONTAINER_ENGINE) build $(PLATFORM_FLAG) $(BUILD_FLAGS) -t $(BACKEND_IMAGE) .

build-operator: ## Build the operator container image
	@echo "Building operator image with $(CONTAINER_ENGINE)..."
	cd components/operator && $(CONTAINER_ENGINE) build $(PLATFORM_FLAG) $(BUILD_FLAGS) -t $(OPERATOR_IMAGE) .

build-runner: ## Build the Claude Code runner container image
	@echo "Building Claude Code runner image with $(CONTAINER_ENGINE)..."
	cd components/runners && $(CONTAINER_ENGINE) build $(PLATFORM_FLAG) $(BUILD_FLAGS) -t $(RUNNER_IMAGE) -f claude-code-runner/Dockerfile .

# Kubernetes deployment
deploy: ## Deploy all components to OpenShift (production overlay)
	@echo "Deploying to OpenShift..."
	cd components/manifests && ./deploy.sh

# Cleanup
clean: ## Clean up all Kubernetes resources (production overlay)
	@echo "Cleaning up Kubernetes resources..."
	cd components/manifests && ./deploy.sh clean



push-all: ## Push all images to registry
	$(CONTAINER_ENGINE) tag $(FRONTEND_IMAGE) $(REGISTRY)/$(FRONTEND_IMAGE)
	$(CONTAINER_ENGINE) tag $(BACKEND_IMAGE) $(REGISTRY)/$(BACKEND_IMAGE)
	$(CONTAINER_ENGINE) tag $(OPERATOR_IMAGE) $(REGISTRY)/$(OPERATOR_IMAGE)
	$(CONTAINER_ENGINE) tag $(RUNNER_IMAGE) $(REGISTRY)/$(RUNNER_IMAGE)
	$(CONTAINER_ENGINE) push $(REGISTRY)/$(FRONTEND_IMAGE)
	$(CONTAINER_ENGINE) push $(REGISTRY)/$(BACKEND_IMAGE)
	$(CONTAINER_ENGINE) push $(REGISTRY)/$(OPERATOR_IMAGE)
	$(CONTAINER_ENGINE) push $(REGISTRY)/$(RUNNER_IMAGE)

# Git hooks for branch protection
setup-hooks: ## Install git hooks for branch protection
	@./scripts/install-git-hooks.sh

remove-hooks: ## Remove git hooks
	@echo "Removing git hooks..."
	@rm -f .git/hooks/pre-commit
	@rm -f .git/hooks/pre-push
	@echo "✅ Git hooks removed"

# Local development with minikube
NAMESPACE ?= ambient-code

local-start: ## Start minikube and deploy vTeam
	@command -v minikube >/dev/null || (echo "❌ Please install minikube first: https://minikube.sigs.k8s.io/docs/start/" && exit 1)
	@echo "🚀 Starting minikube..."
	@minikube start --memory=4096 --cpus=2 || true
	@echo "📦 Enabling required addons..."
	@minikube addons enable ingress
	@minikube addons enable storage-provisioner
	@echo "🏗️  Building images in minikube..."
	@eval $$(minikube docker-env) && \
		docker build -t vteam-backend:latest components/backend && \
		docker build -t vteam-frontend:latest components/frontend && \
		docker build -t vteam-operator:latest components/operator
	@echo "📋 Creating namespace..."
	@kubectl create namespace $(NAMESPACE) --dry-run=client -o yaml | kubectl apply -f -
	@echo "🔧 Deploying CRDs..."
	@kubectl apply -f components/manifests/crds/ || true
	@echo "🔐 Deploying RBAC..."
	@kubectl apply -f components/manifests/rbac/ || true
	@echo "💾 Creating PVCs..."
	@kubectl apply -f components/manifests/workspace-pvc.yaml -n $(NAMESPACE) || true
	@echo "🚀 Deploying backend..."
	@kubectl apply -f components/manifests/minikube/backend-deployment.yaml
	@kubectl apply -f components/manifests/minikube/backend-service.yaml
	@echo "🌐 Deploying frontend..."
	@kubectl apply -f components/manifests/minikube/frontend-deployment.yaml
	@kubectl apply -f components/manifests/minikube/frontend-service.yaml
	@echo "🤖 Deploying operator..."
	@kubectl apply -f components/manifests/minikube/operator-deployment.yaml
	@echo "🌍 Creating ingress..."
	@echo "   Waiting for ingress controller to be ready..."
	@kubectl wait --namespace ingress-nginx --for=condition=ready pod --selector=app.kubernetes.io/component=controller --timeout=120s || true
	@kubectl apply -f components/manifests/minikube/ingress.yaml || echo "   ⚠️  Ingress creation failed (controller may still be starting)"
	@echo "🔑 Granting backend permissions..."
	@kubectl create clusterrolebinding backend-admin --clusterrole=cluster-admin --serviceaccount=$(NAMESPACE):backend-api --dry-run=client -o yaml | kubectl apply -f -
	@echo ""
	@echo "✅ Deployment complete!"
	@echo ""
	@echo "📍 Access URLs:"
	@echo "   Add to /etc/hosts: 127.0.0.1 vteam.local"
	@echo "   Frontend: http://vteam.local"
	@echo "   Backend:  http://vteam.local/api"
	@echo ""
	@echo "   Or use NodePort:"
	@echo "   Frontend: http://$$(minikube ip):30030"
	@echo "   Backend:  http://$$(minikube ip):30080"
	@echo ""
	@echo "🔍 Check status with: make local-status"

# E2E Testing with kind
e2e-test: ## Run complete e2e test suite (setup, deploy, test, cleanup)
	@echo "Running e2e tests..."
	@# Clean up any existing cluster first
	@cd e2e && CONTAINER_ENGINE=$(CONTAINER_ENGINE) ./scripts/cleanup.sh 2>/dev/null || true
	@# Setup and deploy (allows password prompt for /etc/hosts)
	cd e2e && CONTAINER_ENGINE=$(CONTAINER_ENGINE) ./scripts/setup-kind.sh
	cd e2e && CONTAINER_ENGINE=$(CONTAINER_ENGINE) ./scripts/deploy.sh
	@# Run tests with cleanup trap (no more password prompts needed)
	@cd e2e && trap 'CONTAINER_ENGINE=$(CONTAINER_ENGINE) ./scripts/cleanup.sh' EXIT; ./scripts/run-tests.sh

e2e-setup: ## Install e2e test dependencies
	@echo "Installing e2e test dependencies..."
	cd e2e && npm install

e2e-clean: ## Clean up e2e test environment
	@echo "Cleaning up e2e environment..."
	cd e2e && CONTAINER_ENGINE=$(CONTAINER_ENGINE) ./scripts/cleanup.sh

deploy-langfuse-openshift: ## Deploy Langfuse to OpenShift/ROSA cluster
	@echo "Deploying Langfuse to OpenShift cluster..."
	@cd e2e && ./scripts/deploy-langfuse.sh --openshift

# Minikube local development targets
local-stop: ## Stop vTeam (delete namespace, keep minikube running)
	@echo "🛑 Stopping vTeam..."
	@kubectl delete namespace $(NAMESPACE) --ignore-not-found=true
	@echo "✅ vTeam stopped. Minikube is still running."
	@echo "   To stop minikube: make local-delete"

local-delete: ## Delete minikube cluster completely
	@echo "🗑️  Deleting minikube cluster..."
	@minikube delete
	@echo "✅ Minikube cluster deleted."

local-status: ## Show status of local deployment
	@echo "🔍 Minikube status:"
	@minikube status || echo "❌ Minikube not running"
	@echo ""
	@echo "📦 Pods in namespace $(NAMESPACE):"
	@kubectl get pods -n $(NAMESPACE) 2>/dev/null || echo "❌ No pods found (namespace may not exist)"
	@echo ""
	@echo "🌐 Services:"
	@kubectl get svc -n $(NAMESPACE) 2>/dev/null || echo "❌ No services found"
	@echo ""
	@echo "🔗 Ingress:"
	@kubectl get ingress -n $(NAMESPACE) 2>/dev/null || echo "❌ No ingress found"

local-logs: ## Show logs from backend
	@kubectl logs -n $(NAMESPACE) -l app=backend-api --tail=50 -f

local-logs-frontend: ## Show frontend logs
	@kubectl logs -n $(NAMESPACE) -l app=frontend --tail=50 -f

local-logs-operator: ## Show operator logs
	@kubectl logs -n $(NAMESPACE) -l app=agentic-operator --tail=50 -f

local-logs-all: ## Show logs from all pods
	@kubectl logs -n $(NAMESPACE) -l 'app in (backend-api,frontend,agentic-operator)' --tail=20 --prefix=true

local-restart: ## Restart all deployments
	@echo "🔄 Restarting all deployments..."
	@kubectl rollout restart deployment -n $(NAMESPACE)
	@kubectl rollout status deployment -n $(NAMESPACE) --timeout=60s

local-restart-backend: ## Restart backend deployment
	@kubectl rollout restart deployment/backend-api -n $(NAMESPACE)
	@kubectl rollout status deployment/backend-api -n $(NAMESPACE) --timeout=60s

local-restart-frontend: ## Restart frontend deployment
	@kubectl rollout restart deployment/frontend -n $(NAMESPACE)
	@kubectl rollout status deployment/frontend -n $(NAMESPACE) --timeout=60s

local-restart-operator: ## Restart operator deployment
	@kubectl rollout restart deployment/agentic-operator -n $(NAMESPACE)
	@kubectl rollout status deployment/agentic-operator -n $(NAMESPACE) --timeout=60s

local-shell-backend: ## Open shell in backend pod
	@kubectl exec -it -n $(NAMESPACE) $$(kubectl get pod -n $(NAMESPACE) -l app=backend-api -o jsonpath='{.items[0].metadata.name}') -- /bin/sh

local-shell-frontend: ## Open shell in frontend pod
	@kubectl exec -it -n $(NAMESPACE) $$(kubectl get pod -n $(NAMESPACE) -l app=frontend -o jsonpath='{.items[0].metadata.name}') -- /bin/sh

dev-test: ## Run tests against local minikube deployment
	@echo "🧪 Testing local deployment..."
	@echo ""
	@echo "Testing backend health endpoint..."
	@curl -f http://$$(minikube ip):30080/health && echo "✅ Backend is healthy" || echo "❌ Backend health check failed"
	@echo ""
	@echo "Testing frontend..."
	@curl -f http://$$(minikube ip):30030 > /dev/null && echo "✅ Frontend is accessible" || echo "❌ Frontend check failed"
	@echo ""
	@echo "Checking pods..."
	@kubectl get pods -n $(NAMESPACE) | grep -E "(backend-api|frontend)" | grep Running && echo "✅ All pods running" || echo "❌ Some pods not running"

# Backward compatibility aliases (dev-* -> local-*)
dev-start: local-start ## Alias for local-start (backward compatibility)

dev-stop: local-stop ## Alias for local-stop (backward compatibility)

dev-logs: local-logs ## Alias for local-logs (backward compatibility)

dev-logs-backend: local-logs ## Alias for local-logs (backward compatibility)

dev-logs-frontend: local-logs-frontend ## Alias for local-logs-frontend (backward compatibility)
