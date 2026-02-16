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

### 🚀 Minikube (Older Alternative)

**Status:** ⚠️ Still supported but Kind is recommended for new development

**Best for:** Beginners uncomfortable with Docker, Windows users

**Best for:** First-time setup, general development, stable environment

**Pros:**
- ✅ Mature and well-documented
- ✅ Works on all platforms (macOS, Linux, Windows)
- ✅ Simpler troubleshooting
- ✅ Stable driver support

**Cons:**
- ⏱️ Slower startup (~2-3 minutes)
- 💾 Higher memory usage

**Quick Start:**
```bash
make local-up
# Access at http://$(minikube ip):30030
```

**Full Guide:** [minikube.md](minikube.md)

---

### 🐳 Kind (Kubernetes in Docker)

**Best for:** E2E testing, CI/CD, experienced Kubernetes developers

**Pros:**
- ⚡ Fast startup (~30 seconds)
- 🎯 Same environment as CI/CD
- 💨 Lightweight and quick to reset
- 🔄 Multiple clusters easy

**Cons:**
- 📚 Steeper learning curve
- 🐛 Less forgiving of configuration mistakes
- 🐳 Requires Docker knowledge

**Quick Start:**
```bash
make kind-up
make test-e2e
make kind-down
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

**Full Guide:** [crc.md](crc.md)

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

| Feature | **Kind (Recommended)** | Minikube | CRC | Hybrid |
|---------|------------------------|----------|-----|--------|
| **Status** | ✅ **Recommended** | ⚠️ Older | ⚠️ Specialized | Advanced |
| **Startup Time** | ⚡ ~30 sec | ~2-3 min | ~5-10 min | ~30 sec + manual |
| **Memory Usage** | Lower | Higher | Highest | Lowest |
| **CI/CD Match** | ✅ **Yes (exact!)** | No | No | No |
| **Learning Curve** | Moderate | Easier | Moderate | Advanced |
| **Code Iteration** | Moderate | Slow (rebuild) | Fast (hot-reload) | ⚡ Instant |
| **Debugging** | Logs only | Logs only | Logs only | ✅ IDE debugging |
| **OpenShift Features** | No | No | ✅ Yes | No |
| **Production-Like** | Good | Basic | ✅ Best | No |
| **Integration Testing** | ✅ **Best** | Yes | Yes | Limited |
| **E2E Testing** | ✅ **Required** | Yes | Yes | No |
| **Platform Support** | Linux/macOS | All | macOS/Linux | All |
| **Our CI Uses** | ✅ **Kind** | No | No | No |

## Which Should I Use?

### ⭐ Choose **Kind** (Recommended for 95% of use cases)
- 👋 You're new to the project → **Start with Kind**
- 🧪 You're writing or running E2E tests → **Use Kind**
- 🔄 You're working on any development → **Use Kind**
- ⚡ You value fast iteration → **Use Kind**
- 🎯 You want to match CI/CD environment → **Use Kind**

**TL;DR:** Just use Kind. It's faster, lighter, and matches our CI environment.

---

### Choose **Minikube** only if:
- 💻 You're on Windows (Kind doesn't work well on Windows)
- 🆘 Kind doesn't work on your machine for some reason
- 📚 You already have Minikube experience

**Note:** Minikube is the older approach. We recommend migrating to Kind.

---

### Choose **CRC** only if:
- 🔴 You **specifically** need OpenShift Routes (not Ingress)
- 🏗️ You're testing OpenShift BuildConfigs
- 🔐 You're developing OpenShift OAuth integration
- 🎛️ You need the OpenShift console

**Note:** CRC is for OpenShift-specific features only. If you don't need OpenShift features, use Kind.

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
Use **CRC** only if you need Routes, BuildConfigs, etc:
```bash
make local-up  # Local dev (minikube-based)
```

### Need to Debug with Breakpoints?
Use **Hybrid** to run components locally:
```bash
make kind-up
cd components/backend && go run .
```

## Additional Resources

- [Kind Quick Start](../../../QUICK_START.md) - 2-minute setup
- [Minikube Setup](minikube.md) - Older approach (deprecated)
- [Kind Development Guide](kind.md) - Using Kind for development and testing
- [CRC Development Guide](crc.md) - OpenShift Local development
- [Hybrid Development Guide](hybrid.md) - Running components locally
- [E2E Testing](../../testing/e2e-guide.md) - End-to-end test suite
