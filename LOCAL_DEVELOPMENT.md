# Local Development Guide

This guide explains how to set up and use the minikube-based local development environment for the Ambient Code Platform.

## Complete Feature List

✅ **Authentication Disabled** - No login required  
✅ **Automatic Mock User** - Login automatically as "developer"  
✅ **Full Project Management** - Create, view, and manage projects  
✅ **Service Account Permissions** - Backend uses Kubernetes service account in dev mode  
✅ **Ingress Routing** - Access via hostname or NodePort  
✅ **All Components Running** - Frontend, backend, and operator fully functional

## Prerequisites

- Docker
- Minikube
- kubectl

### Installation

```bash
# macOS
brew install minikube kubectl

# Linux
curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
sudo install minikube-linux-amd64 /usr/local/bin/minikube
```

## Quick Start

```bash
# Start local environment
make dev-start

# Add to /etc/hosts (optional, for ingress access)
echo "127.0.0.1 vteam.local" | sudo tee -a /etc/hosts
```

## Access URLs

### Via Ingress (after /etc/hosts)
- Frontend: http://vteam.local
- Backend: http://vteam.local/api/health

### Via NodePort (no /etc/hosts needed)
- Frontend: http://$(minikube ip):30030
- Backend: http://$(minikube ip):30080/health

## Authentication

Authentication is **completely disabled** for local development:

- ✅ No OpenShift OAuth required
- ✅ Automatic login as "developer"
- ✅ Full access to all features
- ✅ Backend uses service account for Kubernetes API

### How It Works

1. **Frontend**: Sets `DISABLE_AUTH=true` environment variable
2. **Auth Handler**: Automatically provides mock credentials:
   - User: developer
   - Email: developer@localhost
   - Token: mock-token-for-local-dev

3. **Backend**: Detects mock token and uses service account credentials

## Features Tested

### ✅ Projects
- View project list
- Create new projects
- Access project details

### ✅ Backend API
- Health endpoint working
- Projects API returning data
- Service account permissions working

### ✅ Ingress
- Frontend routing works
- Backend API routing works  
- Load balancer configured

## Common Commands

```bash
# View status
make local-status

# View logs
make local-logs              # Backend
make local-logs-frontend     # Frontend
make local-logs-operator     # Operator

# Restart components
make local-restart           # All
make local-restart-backend   # Backend only

# Stop/delete
make local-stop              # Stop deployment
make local-delete            # Delete minikube cluster
```

## Development Workflow

1. Make code changes
2. Rebuild images:
   ```bash
   eval $(minikube docker-env)
   docker build -t vteam-backend:latest components/backend
   ```
3. Restart deployment:
   ```bash
   make local-restart-backend
   ```

## Troubleshooting

### Projects Not Showing
- Backend requires cluster-admin permissions
- Added via: `kubectl create clusterrolebinding backend-admin --clusterrole=cluster-admin --serviceaccount=ambient-code:backend-api`

### Frontend Auth Errors
- Frontend needs `DISABLE_AUTH=true` environment variable
- Backend middleware checks for mock token

### Ingress Not Working
- Wait for ingress controller to be ready
- Check: `kubectl get pods -n ingress-nginx`

## Technical Details

### Authentication Flow
1. Frontend sends request with `X-Forwarded-Access-Token: mock-token-for-local-dev`
2. Backend middleware checks: `if token == "mock-token-for-local-dev"`
3. Backend uses `server.K8sClient` and `server.DynamicClient` (service account)
4. No RBAC restrictions - full cluster access

### Environment Variables
- `DISABLE_AUTH=true` (Frontend & Backend)
- `MOCK_USER=developer` (Frontend)

### RBAC
- Backend service account has cluster-admin role
- All namespaces accessible
- Full Kubernetes API access

## Production Differences

| Feature | Minikube (Dev) | OpenShift (Prod) |
|---------|----------------|------------------|
| Authentication | Disabled, mock user | OpenShift OAuth |
| User Tokens | Mock token | Real OAuth tokens |
| Kubernetes Access | Service account | User token with RBAC |
| Namespace Visibility | All (cluster-admin) | User permissions |

## Changes Made

### Backend (`components/backend/handlers/middleware.go`)
```go
// In dev mode, use service account credentials for mock tokens
if token == "mock-token-for-local-dev" || os.Getenv("DISABLE_AUTH") == "true" {
    log.Printf("Dev mode detected - using service account credentials for %s", c.FullPath())
    return server.K8sClient, server.DynamicClient
}
```

### Frontend (`components/frontend/src/lib/auth.ts`)
```typescript
// If auth is disabled, provide mock credentials
if (process.env.DISABLE_AUTH === 'true') {
  const mockUser = process.env.MOCK_USER || 'developer';
  headers['X-Forwarded-User'] = mockUser;
  headers['X-Forwarded-Preferred-Username'] = mockUser;
  headers['X-Forwarded-Email'] = `${mockUser}@localhost`;
  headers['X-Forwarded-Access-Token'] = 'mock-token-for-local-dev';
  return headers;
}
```

## Success Criteria

✅ All components running  
✅ Projects create and list successfully  
✅ No authentication required  
✅ Full application functionality available  
✅ Development workflow simple and fast

