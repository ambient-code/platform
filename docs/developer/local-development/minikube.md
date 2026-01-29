# Minikube Local Development

> ⚠️ **Note:** Minikube is an older approach. We recommend using [Kind](kind.md) for faster iteration and CI/CD compatibility.
> 
> Minikube is still supported but considered deprecated for new development.

## When to Use Minikube

**Use Minikube only if:**
- 💻 You're on Windows (Kind doesn't work well on Windows)
- 🆘 Kind doesn't work on your machine
- 📚 You already have a Minikube workflow established

**Otherwise, use Kind:** [kind.md](kind.md)

## Quick Start

See [QUICK_START.md](../../../QUICK_START.md) for complete Minikube setup instructions.

```bash
make local-up
# Access at http://$(minikube ip):30030
```

## Why We Recommend Kind Instead

| Reason | Kind | Minikube |
|--------|------|----------|
| **Startup** | 30 seconds | 2-3 minutes |
| **Memory** | Lower | Higher |
| **CI/CD Match** | ✅ Exact match | ❌ Different |
| **Iteration Speed** | Faster | Slower |
| **Industry Standard** | ✅ Official K8s project | Older approach |

## Migration from Minikube to Kind

Switching from Minikube to Kind is straightforward:

```bash
# Stop Minikube
make local-down
minikube delete

# Start Kind
make kind-up

# Access at http://localhost:8080 (not minikube ip)
```

**Key Differences:**
- **Access:** `localhost:8080` instead of `$(minikube ip):30030`
- **Commands:** `make kind-up` instead of `make local-up`
- **Testing:** Same commands work in both environments

## Full Minikube Documentation

For complete Minikube setup and usage, see:
- [QUICK_START.md](../../../QUICK_START.md) - 5-minute Minikube setup
- [LOCAL_DEVELOPMENT.md](../../LOCAL_DEVELOPMENT.md) - Detailed Minikube guide

## See Also

- **[Kind Development](kind.md)** - Recommended approach
- **[Local Development Comparison](README.md)** - Compare all options
- **[Hybrid Development](hybrid.md)** - Run components locally
