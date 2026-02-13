# Local Development Environments

**Bottom line:** Use Kind. Run `make kind-up` and access at `http://localhost:8080`. Everything else on this page is for edge cases.

## Approaches

| Approach | When to Use | Startup | Guide |
|----------|-------------|---------|-------|
| **Kind** | All development and testing (default) | ~30 sec | [kind.md](kind.md) |
| Hybrid | Rapid single-component iteration with IDE debugging | ~30 sec + manual | [hybrid.md](hybrid.md) |
| CRC | OpenShift-specific features (Routes, OAuth, BuildConfigs) | ~5-10 min | [crc.md](crc.md) |
| Minikube | Legacy fallback (deprecated) | ~2-3 min | [minikube.md](minikube.md) |

### Kind (Recommended)

Matches CI exactly. Fastest startup. Lowest resource usage.

```bash
make kind-up           # Deploy with Quay.io images
make kind-port-forward # In another terminal
make test-e2e          # Run tests
make kind-down         # Cleanup
```

### Hybrid

Run one component locally (with IDE breakpoints) while Kind provides the cluster.

```bash
make kind-up
cd components/backend && go run .
```

See [hybrid.md](hybrid.md).

### CRC (OpenShift Local)

Only needed for OpenShift Routes, BuildConfigs, or OAuth integration testing.

```bash
make dev-start
# Access at https://vteam-frontend-vteam-dev.apps-crc.testing
```

See [crc.md](crc.md).

### Minikube (Deprecated)

Still supported but Kind is recommended. See [minikube.md](minikube.md) for migration instructions.

## Comparison

| Feature | Kind | Hybrid | CRC | Minikube |
|---------|------|--------|-----|----------|
| Matches CI | Yes | No | No | No |
| Code iteration | Moderate | Instant | Fast (hot-reload) | Slow |
| IDE debugging | No | Yes | No | No |
| OpenShift features | No | No | Yes | No |
| Memory usage | Low | Lowest | High | Medium |
| Platform | Linux/macOS | All | macOS/Linux | All |

## See Also

- [Kind Development Guide](kind.md) - Full reference
- [E2E Testing](../../testing/e2e-guide.md) - Test suite documentation
