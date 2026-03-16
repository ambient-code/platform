# Local Development Environments

The Ambient Code Platform supports four local development approaches. **Kind is recommended** for most development and testing.

## Choose Your Approach

### 🐳 Kind (Kubernetes in Docker) - **RECOMMENDED**

**Best for:** All development, E2E testing, CI/CD

**Why Kind?**
- ⚡ **Fastest startup** (~30 seconds)
- 🎯 **Same as CI** - Tests run in Kind, develop in Kind
- 💨 **Lightweight** - Lower memory usage
- 🔄 **Quick iteration** - Fast to create/destroy clusters
- ✅ **Battle-tested** - Used by Kubernetes project itself

**Pros:**
- ⚡ Fast startup (~30 seconds)
- 🎯 Matches CI/CD environment exactly
- 💨 Lightweight and quick to reset
- 🔄 Multiple clusters easy
- ✅ Official Kubernetes project

**Cons:**
- 📚 Requires basic Docker knowledge
- 🐳 Docker must be installed

**Quick Start:**
```bash
make kind-up
# Access at http://localhost:8080
```

**Full Guide:** [kind.md](kind.md)

---

### 🔴 OpenShift Local (CRC) (Specialized Use)

**Status:** ⚠️ Use only when you need OpenShift-specific features

**Best for:** Testing OpenShift Routes, BuildConfigs, OAuth integration

**Pros:**
- ✅ Full OpenShift features (Routes, BuildConfigs, OAuth)
- ✅ Production-like environment
- ✅ OpenShift console access
- ✅ Hot-reloading development mode

**Cons:**
- ⏱️ Slower startup (~5-10 minutes first time)
- 💾 Higher resource requirements
- 🖥️ macOS and Linux only

**Quick Start:**
```bash
make local-up    # Note: CRC dev-* targets have been replaced with local-* equivalents
```

**Full Guides:** [crc.md](crc.md) | [openshift.md](openshift.md)

---

### ⚡ Hybrid Local Development

**Best for:** Rapid iteration on specific components

**What it is:** Run components (frontend, backend, operator) locally on your machine while using Kind for dependencies (CRDs, MinIO).

**Pros:**
- 🚀 Instant code reloads (no container rebuilds)
- 🐛 Direct debugging with IDE breakpoints
- ⚡ Fastest iteration cycle (seconds)

**Cons:**
- 🔧 More manual setup
- 🧩 Need to manage multiple terminals
- 💻 Not suitable for integration testing

**Quick Start:**
```bash
make kind-up
# Then run components locally (see guide)
```

**Full Guide:** [hybrid.md](hybrid.md)

---

## Quick Comparison

| Feature | **Kind (Recommended)** | CRC | Hybrid |
|---------|------------------------|-----|--------|
| **Status** | ✅ **Recommended** | ⚠️ Specialized | Advanced |
| **Startup Time** | ⚡ ~30 sec | ~5-10 min | ~30 sec + manual |
| **Memory Usage** | Lower | Highest | Lowest |
| **CI/CD Match** | ✅ **Yes (exact!)** | No | No |
| **Learning Curve** | Moderate | Moderate | Advanced |
| **Code Iteration** | Moderate | Fast (hot-reload) | ⚡ Instant |
| **Debugging** | Logs only | Logs only | ✅ IDE debugging |
| **OpenShift Features** | No | ✅ Yes | No |
| **Production-Like** | Good | ✅ Best | No |
| **Integration Testing** | ✅ **Best** | Yes | Limited |
| **E2E Testing** | ✅ **Required** | Yes | No |
| **Platform Support** | Linux/macOS | macOS/Linux | All |
| **Our CI Uses** | ✅ **Kind** | No | No |

## Which Should I Use?

### ⭐ Choose **Kind** (Recommended for 95% of use cases)
- 👋 You're new to the project → **Start with Kind**
- 🧪 You're writing or running E2E tests → **Use Kind**
- 🔄 You're working on any development → **Use Kind**
- ⚡ You value fast iteration → **Use Kind**
- 🎯 You want to match CI/CD environment → **Use Kind**

**TL;DR:** Just use Kind. It's faster, lighter, and matches our CI environment.

---

### Choose **OpenShift** only if:
- 🔴 You **specifically** need OpenShift Routes (not Ingress)
- 🏗️ You're testing OpenShift BuildConfigs
- 🔐 You're developing OpenShift OAuth integration
- 🎛️ You need the OpenShift console
- ☁️ You're deploying to production OpenShift clusters

**Note:** OpenShift is for OpenShift-specific features only. If you don't need OpenShift features, use Kind.

---

### Choose **Hybrid** if:
- 🚀 You're rapidly iterating on ONE component
- 🐛 You need to debug with IDE breakpoints
- ⚡ Container rebuild time is slowing you down
- 💪 You're very comfortable with Kubernetes

## Getting Started

### 👉 First Time Here? Use Kind!

**Our recommendation for everyone:**

```bash
# 1. Install Docker (if not already installed)
# 2. Start Kind cluster
make kind-up

# 3. Verify
make test-e2e

# Access at http://localhost:8080
```

**Full guide:** [kind.md](kind.md)

### Working on E2E Tests?
Use **Kind** - it's what CI uses:
```bash
make kind-up
make test-e2e
```

### Need OpenShift-Specific Features?
Use **CRC** for local OpenShift or **OpenShift cluster** for cloud deployment:
```bash
make local-up  # Local CRC dev
# OR deploy to OpenShift cluster (see openshift.md)
```

### Need to Debug with Breakpoints?
Use **Hybrid** to run components locally:
```bash
make kind-up
cd components/backend && go run .
```

## Additional Resources

- [Kind Quick Start](../../../QUICK_START.md) - 2-minute setup
- [Kind Development Guide](kind.md) - Using Kind for development and testing
- [CRC Development Guide](crc.md) - OpenShift Local development
- [OpenShift Cluster Guide](openshift.md) - OpenShift cluster deployment
- [Hybrid Development Guide](hybrid.md) - Running components locally
- [E2E Testing](../../testing/e2e-guide.md) - End-to-end test suite
