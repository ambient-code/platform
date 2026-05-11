## Summary

<!-- What changed and why. Focus on the WHY — the code shows the WHAT. -->

## Related Issues

<!-- Jira: RHOAIENG-XXXXX | GitHub: #issue -->

## Type of Change

<!-- Check one -->
- [ ] `feat` — new feature
- [ ] `fix` — bug fix
- [ ] `refactor` — behavior-preserving restructure
- [ ] `test` — tests only
- [ ] `docs` — documentation only
- [ ] `chore` — tooling, deps, config

## Component(s)

<!-- Check all that apply -->
- [ ] backend
- [ ] frontend
- [ ] operator
- [ ] runner
- [ ] cli
- [ ] manifests
- [ ] docs

---

## Commit Discipline (Constitution Principle X)

- [ ] Each commit is atomic and independently revertable
- [ ] Conventional commit format used: `type(scope): description`
- [ ] Commit messages explain WHY, not WHAT
- [ ] No WIP commits (squashed before submission)
- [ ] Line counts within thresholds (bugfix ≤150, feat-small ≤300, feat-medium ≤500, refactor ≤400)
- [ ] PR total is under 600 lines — or justification below:

<!-- If PR exceeds 600 lines or a threshold, explain why it cannot be split: -->

## Quality

- [ ] Tests added or updated for all changes
- [ ] `make lint` passes
- [ ] `npm run build` passes (frontend)
- [ ] `go vet ./...` passes (Go components)
- [ ] `python -m pytest tests/` passes (runner)
- [ ] No `any` types in frontend TypeScript (frontend)
- [ ] No `panic()` in Go production code (Go components)
- [ ] OwnerReferences set on any new child Kubernetes resources (if applicable)
- [ ] User-facing ops use `GetK8sClientsForRequest()`, not the service account (if applicable)

## Documentation

- [ ] `CLAUDE.md` / `AGENTS.md` updated if conventions changed
- [ ] API changes documented
- [ ] Breaking changes noted in PR description

## Security

- [ ] No secrets, tokens, or API keys committed
- [ ] `SecurityContext` set on any new container specs (`runAsNonRoot`, drop `ALL` caps, `readOnlyRootFilesystem`)
