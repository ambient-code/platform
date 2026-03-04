# Verify Code Quality

Run all static analysis checks: go vet, format verification, and linting.

## Instructions

### Step 1: Run Verification

```bash
make verify
```

This runs:
- `go vet ./cmd/... ./pkg/... ./test/...` — checks for suspicious constructs
- `gofmt` formatting check — warns if any files are not properly formatted

### Step 2: Run Linter

```bash
make lint
```

This runs `golangci-lint` against the codebase (ignores `unused` warnings).

### Step 3: Fix Issues

- **Formatting issues**: Run `gofmt -w .` to auto-fix
- **Vet errors**: These indicate real bugs (unreachable code, incorrect printf args, etc.) — fix manually
- **Lint warnings**: Address based on severity; some may be intentional

### Step 4: Build Confirmation

```bash
make binary
```

## When to Run This

- Before committing changes
- After generating a new Kind
- As part of a CI-like local check before pushing
- When the user asks to "clean up" or "check the code"
