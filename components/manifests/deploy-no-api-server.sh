#!/bin/bash

# OpenShift Deployment Script for vTeam Ambient (excluding API server)
# Usage: ./deploy-no-api-server.sh
# 
# This script deploys all components EXCEPT the ambient-api-server, which should 
# already be deployed with TLS fixes from a different branch.

set -e

# Always run from the script's directory (manifests root)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 vTeam Ambient - OpenShift Deployment (No API Server)${NC}"
echo -e "${BLUE}======================================================${NC}"
echo -e "${YELLOW}Note: This deploys all components EXCEPT ambient-api-server${NC}"
echo -e "${YELLOW}      API server should already be running with TLS fixes${NC}"
echo ""

# Configuration  
NAMESPACE="${NAMESPACE:-ambient-code}"
echo -e "Namespace: ${GREEN}${NAMESPACE}${NC}"
echo ""

# Check if API server is already running
echo -e "${YELLOW}Checking existing API server deployment...${NC}"
if ! oc get deployment ambient-api-server -n ${NAMESPACE} >/dev/null 2>&1; then
    echo -e "${RED}❌ ambient-api-server deployment not found in ${NAMESPACE}${NC}"
    echo -e "${YELLOW}Please deploy the API server first with TLS fixes${NC}"
    exit 1
fi

API_SERVER_IMAGE=$(oc get deployment ambient-api-server -n ${NAMESPACE} -o jsonpath='{.spec.template.spec.containers[?(@.name=="api-server")].image}')
echo -e "${GREEN}✅ Found ambient-api-server deployment${NC}"
echo -e "API Server Image: ${GREEN}${API_SERVER_IMAGE}${NC}"
echo ""

# Check prerequisites
echo -e "${YELLOW}Checking prerequisites...${NC}"
if ! command -v oc >/dev/null 2>&1; then
    echo -e "${RED}❌ OpenShift CLI (oc) not found${NC}"
    exit 1
fi

if ! command -v kustomize >/dev/null 2>&1; then
    echo -e "${RED}❌ Kustomize not found${NC}"
    exit 1
fi

if ! oc whoami >/dev/null 2>&1; then
    echo -e "${RED}❌ Not logged in to OpenShift${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Prerequisites check passed${NC}"
echo -e "${GREEN}✅ Authenticated as: $(oc whoami)${NC}"
echo ""

# Deploy using the no-api-server overlay
echo -e "${YELLOW}Deploying components using no-api-server overlay...${NC}"

cd overlays/no-api-server

# Set namespace if different from default
if [ "$NAMESPACE" != "ambient-code" ]; then
    echo -e "${BLUE}Setting custom namespace: ${NAMESPACE}${NC}"
    kustomize edit set namespace "$NAMESPACE"
fi

# Build and apply manifests
echo -e "${BLUE}Building and applying manifests...${NC}"
kustomize build . | oc apply -f -

# Return to manifests root
cd ../..

# Switch to the target namespace
echo -e "${BLUE}Switching to namespace ${NAMESPACE}...${NC}"
oc project ${NAMESPACE}

echo ""
echo -e "${GREEN}✅ Deployment completed!${NC}"
echo ""

# Wait for new deployments to be ready (excluding API server)
echo -e "${YELLOW}Waiting for new deployments to be ready...${NC}"
oc rollout status deployment/backend-api --namespace=${NAMESPACE} --timeout=300s || echo -e "${YELLOW}⚠️ backend-api not ready${NC}"
oc rollout status deployment/agentic-operator --namespace=${NAMESPACE} --timeout=300s || echo -e "${YELLOW}⚠️ agentic-operator not ready${NC}" 
oc rollout status deployment/frontend --namespace=${NAMESPACE} --timeout=300s || echo -e "${YELLOW}⚠️ frontend not ready${NC}"
oc rollout status deployment/ambient-control-plane --namespace=${NAMESPACE} --timeout=300s || echo -e "${YELLOW}⚠️ ambient-control-plane not ready${NC}"

# Show deployment status
echo ""
echo -e "${BLUE}Pod Status:${NC}"
oc get pods -n ${NAMESPACE}
echo ""

echo -e "${BLUE}Services:${NC}"  
oc get services -n ${NAMESPACE}
echo ""

echo -e "${BLUE}Routes:${NC}"
oc get routes -n ${NAMESPACE}
echo ""

# Test control plane connection to API server
echo -e "${YELLOW}Testing control plane connection to API server...${NC}"
if oc logs deployment/ambient-control-plane -n ${NAMESPACE} --tail=10 | grep -q "watch stream established"; then
    echo -e "${GREEN}✅ Control plane successfully connected to API server${NC}"
else
    echo -e "${YELLOW}⚠️ Control plane connection status unclear - check logs${NC}"
fi

echo ""
echo -e "${GREEN}🎉 Deployment successful!${NC}"
echo -e "${GREEN}========================${NC}"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo -e "1. Check control plane logs:"
echo -e "   ${BLUE}oc logs -f deployment/ambient-control-plane -n ${NAMESPACE}${NC}"
echo -e "2. Test e2e functionality:"
echo -e "   ${BLUE}./test-e2e-control-plane.sh${NC}"
echo -e "3. Monitor all pods:"
echo -e "   ${BLUE}oc get pods -n ${NAMESPACE} -w${NC}"
echo ""

# Restore kustomization if we modified it
if [ "$NAMESPACE" != "ambient-code" ]; then
    cd overlays/no-api-server
    kustomize edit set namespace ambient-code
    cd ../..
fi

echo -e "${GREEN}🚀 Ready to test ambient-control-plane with existing API server!${NC}"