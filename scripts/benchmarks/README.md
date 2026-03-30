## Component Benchmark Harness

Shell-based benchmark harness for developer inner-loop timing.

### Goals

- Measure **truly cold** setup/install time per component
- Measure **warm** rebuild time per component
- Compare current branch (`HEAD`) against a baseline ref
- Emit output that works well for both humans and agents

### Commands

```bash
# Human-friendly local summary
make benchmark

# Agent-friendly / pipe-friendly output
make benchmark FORMAT=tsv

# Single component
make benchmark COMPONENT=frontend MODE=cold

# Explicit refs
make benchmark BASELINE=origin/main CANDIDATE=HEAD

# CI mode
make benchmark-ci
```

### Agent Fast Path

Use these defaults unless you have a specific reason not to:

```bash
# Cheapest useful signal for agents
make benchmark FORMAT=tsv COMPONENT=backend MODE=warm REPEATS=1

# Frontend contributor setup budget check
make benchmark FORMAT=tsv COMPONENT=frontend MODE=cold REPEATS=1

# Multi-component comparison after a broad refactor
make benchmark FORMAT=tsv COMPONENT=backend,operator,public-api MODE=warm REPEATS=1
```

Decision guide:

- **Need one quick benchmark datapoint?** Start with a single component and `REPEATS=1`
- **Need contributor setup budget?** Run `MODE=cold`
- **Need incremental compiler/build signal?** Run `MODE=warm`
- **Need machine/agent consumption?** Use `FORMAT=tsv`
- **Need human-readable scanability?** Use default `human`

### Output Modes

- `human`: default for TTY; uses repo conventions (`▶`, `✓`, `✗`, section dividers)
- `tsv`: default when piped; preferred for agents and automation
- `json`: machine-readable full result object

Guidance:

- **Humans**: use `make benchmark`
- **Agents / scripts**: use `make benchmark FORMAT=tsv`
- **Downstream tools**: consume `results.json`

### Cost-Aware Benchmark Strategy

To keep agent runs efficient:

1. Benchmark the **smallest relevant scope first**
2. Use **single-component warm** runs before full sweeps
3. Only increase `REPEATS` after a suspicious or decision-relevant result
4. Use full all-component sweeps sparingly; they are intentionally expensive

Good examples:

- Backend change: `make benchmark FORMAT=tsv COMPONENT=backend MODE=warm REPEATS=1`
- Frontend setup UX: `make benchmark FORMAT=tsv COMPONENT=frontend MODE=cold REPEATS=1`
- SDK generator change: `make benchmark FORMAT=tsv COMPONENT=sdk MODE=warm REPEATS=1`

Avoid by default:

- `make benchmark MODE=both REPEATS=3` on all components during exploratory work
- Interpreting `warm` as real browser-observed HMR latency
- Using `human` output when an agent or script will parse the result

### Cold vs Warm Semantics

Cold:

- Uses isolated worktrees
- Uses isolated caches (`GOMODCACHE`, `GOCACHE`, `npm_config_cache`, `UV_CACHE_DIR`, `PIP_CACHE_DIR`)
- Removes repo-local build/install artifacts
- Intended to approximate a first contributor setup experience

Warm:

- Reuses the same isolated cache root prepared by the harness
- Measures the timed rebuild **after** untimed setup is complete
- Intended to approximate a follow-up incremental compile/build

Important:

- Current `warm` numbers are **build/rebuild proxies**, not true “save file -> browser refreshed” or “save file -> process restarted” hot-reload latency
- For frontend, `warm` currently uses `npm run build`, not a browser-observed HMR measurement

### Component Prerequisites

- `frontend`: Node.js 20+ and npm
- `backend`, `operator`, `public-api`, `api-server`, `cli`, `sdk`: Go 1.21+
- `runner`: Python 3.11+ plus `uv` or `python3 -m venv`
- `api-server`, `cli`, `sdk`: `make`

The harness now preflights these before worktree setup so failures happen fast.

### Known Efficiency Lessons

- Frontend benchmarking is highly sensitive to Node version; use Node 20+ or it will fail fast
- Use `FORMAT=tsv` for agent consumption to minimize context-token cost
- If `reports/benchmarks/` is not writable in the current environment, the harness falls back to a temp directory and prints a warning
- Warm benchmarks only stay warm if the setup phase and timed phase share the same isolated cache env; the harness now does that explicitly
- Session benchmarking is **contract-only** in v1 (`bench_session_*` stubs in `bench-manifest.sh`)
- Full warm sweeps across all components are slow because each component still performs untimed setup before the measured rebuild; use them intentionally, not as the default first move
- A failing preflight is a useful result; treat it as an environment readiness signal rather than forcing the benchmark to continue

### Files

- `scripts/benchmarks/component-bench.sh` - main harness
- `scripts/benchmarks/bench-manifest.sh` - component definitions and session stubs
- `tests/bench-test.sh` - harness self-tests

