# Makefile for ambient-code-backend

.PHONY: help build test test-unit test-contract test-integration clean run container-build container-run

# Default target
help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

# Build targets
build: ## Build the backend binary
	go build -o backend .

clean: ## Clean build artifacts
	rm -f backend main
	go clean

# Test targets
test: test-unit test-contract ## Run all tests (excluding integration tests)

test-unit: ## Run unit tests using Ginkgo
	@echo "Running unit tests with Ginkgo..."
	ginkgo run --label-filter="unit" --junit-report=reports/junit.xml --json-report=reports/results.json test/unit

test-unit-go: ## Run unit tests with go test (alternative)
	go test -v -tags=test ./handlers ./types ./git -timeout=5m

test-contract: ## Run contract tests
	go test ./tests/contract/... -v

test-integration: ## Run integration tests (requires Kubernetes cluster)
	@echo "Running integration tests (requires Kubernetes cluster access)..."
	USE_REAL_CLUSTER=true CLEANUP_RESOURCES=true ginkgo run --label-filter="integration" --timeout=10m

test-integration-short: ## Run integration tests with short timeout
	go test ./tests/integration/... -v -short

test-all: test test-integration ## Run all tests including integration tests

# Ginkgo-specific test targets
test-ginkgo: ## Run all tests using Ginkgo framework
	@echo "Running all tests with Ginkgo..."
	ginkgo run --junit-report=reports/junit.xml --json-report=reports/results.json

test-ginkgo-parallel: ## Run tests in parallel
	@echo "Running tests in parallel..."
	ginkgo run -p --junit-report=reports/junit.xml --json-report=reports/results.json

test-ginkgo-verbose: ## Run tests with verbose output
	@echo "Running tests with verbose output..."
	ginkgo run -v --junit-report=reports/junit.xml --json-report=reports/results.json

test-handlers: ## Run handler tests only
	@echo "Running handler tests..."
	ginkgo run --tags=test --label-filter="handlers" -v

test-types: ## Run type tests only
	@echo "Running type tests..."
	ginkgo run --tags=test --label-filter="types" -v

test-git: ## Run git operation tests only
	@echo "Running git operation tests..."
	ginkgo run --tags=test --label-filter="git" -v

test-fast: ## Run tests excluding slow ones
	@echo "Running fast tests only..."
	SKIP_SLOW_TESTS=true ginkgo run --tags=test --label-filter="!slow"

test-auth: ## Run authentication and authorization tests
	@echo "Running auth tests..."
	ginkgo run --tags=test --label-filter="auth" -v

test-focus: ## Run specific test by pattern (usage: make test-focus FOCUS="test pattern")
	@echo "Running focused tests: $(FOCUS)"
	ginkgo run --focus="$(FOCUS)" -v

# Test with specific configuration
test-integration-local: ## Run integration tests with local configuration
	@echo "Running integration tests with local configuration..."
	TEST_NAMESPACE=ambient-code-test \
	CLEANUP_RESOURCES=true \
	go test ./tests/integration/... -v -timeout=5m

test-integration-ci: ## Run integration tests for CI (no cleanup for debugging)
	@echo "Running integration tests for CI..."
	TEST_NAMESPACE=ambient-code-ci \
	CLEANUP_RESOURCES=false \
	go test ./tests/integration/... -v -timeout=10m -json

test-permissions: ## Run permission and RBAC integration tests specifically
	@echo "Running permission boundary and RBAC tests..."
	TEST_NAMESPACE=ambient-code-test \
	CLEANUP_RESOURCES=true \
	go test ./tests/integration/ -v -run TestPermission -timeout=5m

test-permissions-verbose: ## Run permission tests with detailed output
	@echo "Running permission tests with verbose output..."
	TEST_NAMESPACE=ambient-code-test \
	CLEANUP_RESOURCES=true \
	go test ./tests/integration/ -v -run TestPermission -timeout=5m -count=1

# Coverage targets
test-coverage: ## Run tests with coverage
	go test ./tests/unit/... ./tests/contract/... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Development targets
run: ## Run the backend server locally
	go run .

dev: ## Run with live reload (requires air: go install github.com/cosmtrek/air@latest)
	air

# Container targets
CONTAINER_ENGINE ?= podman

container-build: ## Build container image
	$(CONTAINER_ENGINE) build -t ambient-code-backend .

container-run: ## Run container
	$(CONTAINER_ENGINE) run -p 8080:8080 ambient-code-backend

# Linting and formatting
fmt: ## Format Go code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

lint: ## Run golangci-lint (requires golangci-lint to be installed)
	# Lint production build
	golangci-lint run --timeout=5m
	# Lint test build (handlers unit tests use -tags=test for test-only hooks)
	golangci-lint run --timeout=5m --build-tags=test

# Dependency management
deps: ## Download dependencies
	go mod download

deps-update: ## Update dependencies
	go get -u ./...
	go mod tidy

deps-verify: ## Verify dependencies
	go mod verify

# Installation targets for development tools
install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/cosmtrek/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/onsi/ginkgo/v2/ginkgo@latest

# Kubernetes-specific targets for integration testing
k8s-setup: ## Setup local Kubernetes for testing (requires kubectl and kind)
	@echo "Setting up local Kubernetes cluster for testing..."
	kind create cluster --name ambient-test || true
	kubectl config use-context kind-ambient-test
	@echo "Installing test CRDs..."
	kubectl apply -f ../manifests/crds/ || echo "Warning: Could not install CRDs"

k8s-teardown: ## Teardown local Kubernetes test cluster
	@echo "Tearing down test cluster..."
	kind delete cluster --name ambient-test || true

# Pre-commit hooks
pre-commit: fmt vet test ## Run pre-commit checks

# Build information
version: ## Show version information
	@echo "Go version: $(shell go version)"
	@echo "Git commit: $(shell git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
	@echo "Build time: $(shell date)"

# Environment validation
check-env: ## Check environment setup for development
	@echo "Checking environment..."
	@go version >/dev/null 2>&1 || (echo "❌ Go not installed"; exit 1)
	@echo "✅ Go installed: $(shell go version)"
	@kubectl version --client >/dev/null 2>&1 || echo "⚠️  kubectl not found (needed for integration tests)"
	@podman version >/dev/null 2>&1 || echo "⚠️  Podman not found (needed for container builds)"
	@echo "Environment check complete"