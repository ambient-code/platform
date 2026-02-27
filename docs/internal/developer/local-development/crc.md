# OpenShift Local (CRC) Development

This guide covers using OpenShift Local (CRC) for local development of the Ambient Code Platform.

> **🎉 STATUS: FULLY WORKING** - Project creation, authentication, full OpenShift features

## Overview

**OpenShift Local (CRC)** provides a complete OpenShift cluster on your local machine, including:
- ✅ Full OpenShift features (Routes, BuildConfigs, etc.)
- ✅ OAuth authentication
- ✅ OpenShift console
- ✅ Production-like environment

## Quick Start

### 1. Install Prerequisites
```bash
# macOS
brew install crc

# Get Red Hat pull secret (free account):
# 1. Visit: https://console.redhat.com/openshift/create/local  
# 2. Download to ~/.crc/pull-secret.json
# That's it! The script handles crc setup and configuration automatically.
```

### 2. Start Development Environment
```bash
make dev-start
```
*First run: ~5-10 minutes. Subsequent runs: ~2-3 minutes.*

### 3. Access Your Environment
- **Frontend**: https://vteam-frontend-vteam-dev.apps-crc.testing
- **Backend**: https://vteam-backend-vteam-dev.apps-crc.testing/health  
- **Console**: https://console-openshift-console.apps-crc.testing

### 4. Verify Everything Works
```bash
make dev-test  # Should show 11/12 tests passing
```

## Hot-Reloading Development

```bash
# Terminal 1: Start with development mode
DEV_MODE=true make dev-start

# Terminal 2: Enable file sync  
make dev-sync
```

## Essential Commands

```bash
# Day-to-day workflow
make dev-start          # Start environment
make dev-test           # Run tests
make dev-stop           # Stop (keep CRC running)
make dev-clean          # Full cleanup

# Logs
make dev-logs           # All logs
make dev-logs-backend   # Backend only
make dev-logs-frontend  # Frontend only
make dev-logs-operator  # Operator only

# Operator management
make dev-restart-operator  # Restart operator
make dev-operator-status   # Check operator status
```

## Installation Details

### Platform-Specific Installation

**macOS:**
```bash
# Option 1: Homebrew (Recommended)
brew install crc

# Option 2: Manual Download
curl -LO https://mirror.openshift.com/pub/openshift-v4/clients/crc/latest/crc-macos-amd64.tar.xz
tar -xf crc-macos-amd64.tar.xz
sudo cp crc-macos-*/crc /usr/local/bin/
chmod +x /usr/local/bin/crc
```

**Linux (Fedora/RHEL/CentOS):**
```bash
curl -LO https://mirror.openshift.com/pub/openshift-v4/clients/crc/latest/crc-linux-amd64.tar.xz
tar -xf crc-linux-amd64.tar.xz
sudo cp crc-linux-*/crc /usr/local/bin/
sudo chmod +x /usr/local/bin/crc
```

**Ubuntu/Debian:**
```bash
# Install dependencies
sudo apt-get update
sudo apt-get install qemu-kvm libvirt-daemon libvirt-daemon-system network-manager

# Download and install CRC
curl -LO https://mirror.openshift.com/pub/openshift-v4/clients/crc/latest/crc-linux-amd64.tar.xz
tar -xf crc-linux-amd64.tar.xz
sudo cp crc-linux-*/crc /usr/local/bin/
sudo chmod +x /usr/local/bin/crc
```

### Get Red Hat Pull Secret

1. Visit: https://console.redhat.com/openshift/create/local
2. Sign in (or create free account)
3. Download pull secret
4. Save to `~/.crc/pull-secret.json`

The `make dev-start` script will automatically use this pull secret.

## Features

### ✅ Full OpenShift Features
- Routes (not just Ingress)
- BuildConfigs for local image builds
- OpenShift console
- OAuth authentication
- Production-like environment

### ✅ Development Workflow
- Hot-reloading with `DEV_MODE=true`
- File sync with `make dev-sync`
- Quick operator restarts
- Component-specific log viewing

### ✅ Testing
- Automated test suite
- Operator integration tests
- Full platform validation

## When to Use CRC

**Use CRC when:**
- ✅ You need full OpenShift features (Routes, BuildConfigs)
- ✅ You want production-like environment
- ✅ You're testing OAuth integration
- ✅ You need OpenShift console access

**Use Kind/Minikube when:**
- ✅ You want faster startup
- ✅ You're running E2E tests
- ✅ You don't need OpenShift-specific features

See [Local Development Comparison](README.md) for detailed comparison.

## Troubleshooting

### CRC Won't Start

```bash
# Check CRC status
crc status

# View detailed logs
crc logs

# Reset if needed
crc delete
make dev-start
```

### Pods Not Starting

```bash
# Check pod status
oc get pods -n vteam-dev

# View pod logs
oc logs -n vteam-dev <pod-name>

# Describe pod for events
oc describe pod -n vteam-dev <pod-name>
```

### Routes Not Accessible

```bash
# Check routes
oc get routes -n vteam-dev

# Verify CRC networking
crc ip
ping $(crc ip)

# Check /etc/hosts
grep apps-crc.testing /etc/hosts
```

### BuildConfig Failures

```bash
# Check build logs
oc logs -n vteam-dev bc/vteam-backend -f

# Restart build
oc start-build vteam-backend -n vteam-dev
```

## Advanced Configuration

### Resource Allocation

```bash
# Configure CRC resources before first start
crc config set cpus 6
crc config set memory 16384  # 16GB
crc config set disk-size 100  # 100GB

# Then start
make dev-start
```

### Custom Registry

```bash
# Use external registry instead of internal
export CONTAINER_REGISTRY=quay.io/your-username
make dev-start
```

## Cleanup

```bash
# Stop but keep CRC running
make dev-stop

# Stop and shutdown CRC
make dev-stop-cluster

# Full cleanup (deletes CRC cluster)
make dev-clean
crc delete
```

## See Also

- [Local Development Comparison](README.md) - CRC vs Kind vs Minikube
- [Kind Development](kind.md) - Alternative local environment
- [Hybrid Development](hybrid.md) - Run components locally
- [CLAUDE.md](../../../CLAUDE.md) - Development standards

## References

- **OpenShift Local Documentation**: https://crc.dev/crc/
- **Red Hat OpenShift**: https://www.redhat.com/en/technologies/cloud-computing/openshift
- **CRC GitHub**: https://github.com/crc-org/crc
