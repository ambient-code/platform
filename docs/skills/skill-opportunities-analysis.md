# Skill Opportunities Analysis - Ambient Code Platform

**Date:** 2026-01-27
**Reviewer:** Claude (Cowork Mode)
**Repository:** Ambient Code Platform (formerly vTeam)

## Executive Summary

This repository shows strong potential for skills development. The platform is a Kubernetes-native AI automation system with multiple specialized domains, existing agent definitions, and well-documented patterns. I've identified **12 high-value skill opportunities** organized by priority and impact.

## Repository Context

**Size:** ~18,270 source files (Go, Python, TypeScript/TSX)
**Architecture:** Microservices (NextJS frontend, Go backend, Go operator, Python runner)
**Existing Skills:** 1 (Claude SDK Expert in `.ambient/skills/`)
**Agents:** 7 active agents, 16 in bullpen
**Documentation:** Extensive (ADRs, patterns, context files, implementation plans)

## High-Priority Skill Opportunities

### 1. Kubernetes Operator Development Expert ⭐⭐⭐
**Priority:** P0 - Critical Infrastructure
**Trigger:** kubernetes operator, CRD, controller, reconciliation loop

**Why This Matters:**
The operator is the heart of the platform, managing AgenticSession custom resources and job scheduling. Errors here cascade across the entire system.

**Skill Would Cover:**
- Operator SDK patterns (watch, reconcile, update status)
- CRD design and versioning strategies
- Controller error handling and retry logic
- Status subresource management
- RBAC for operators (ClusterRole vs Role)
- Job lifecycle management
- Leader election patterns
- Finalizers and cleanup hooks
- Resource ownership and garbage collection

**Current Gap:**
Operator code at `components/operator/` lacks centralized expertise. Developers need to reference Kubernetes docs and example repos repeatedly.

**Implementation:**
```
.ambient/skills/k8s-operator-expert/
├── SKILL.md              # Main skill file
├── crd-patterns.md       # CRD best practices
├── reconcile-loop.md     # Reconciliation strategies
├── error-handling.md     # Operator-specific error patterns
└── testing-guide.md      # envtest and integration tests
```

**Estimated Impact:** Reduces operator development time by 40%, prevents common pitfalls (infinite loops, status fights, resource leaks).

---

### 2. GitLab Integration Specialist ⭐⭐⭐
**Priority:** P0 - Active Development Area
**Trigger:** gitlab, git provider, repository integration, self-hosted

**Why This Matters:**
Recent v1.1.0 feature. Multiple documentation files exist (`gitlab-integration.md`, `gitlab-token-setup.md`, `gitlab-self-hosted.md`, `gitlab-testing-procedures.md`), signaling ongoing complexity.

**Skill Would Cover:**
- GitLab API patterns (REST vs GraphQL)
- Personal Access Token (PAT) scopes and permissions
- Self-hosted GitLab configuration (custom domains, SSH)
- Provider detection logic (GitHub vs GitLab URL patterns)
- Multi-provider project handling
- GitLab-specific error messages
- Webhook configuration (for future features)
- CI/CD integration (GitLab CI vs GitHub Actions)

**Current Gap:**
Knowledge scattered across 4+ docs. Testing procedures complex. Self-hosted configurations have edge cases.

**Implementation:**
```
.ambient/skills/gitlab-integration/
├── SKILL.md              # Main skill
├── api-patterns.md       # GitLab API usage
├── token-guide.md        # PAT setup and troubleshooting
├── testing.md            # Test scenarios
└── self-hosted.md        # Enterprise deployment
```

**Estimated Impact:** Accelerates GitLab feature development, reduces support burden, enables faster troubleshooting.

---

### 3. Go Backend API Development Expert ⭐⭐
**Priority:** P0 - Core System
**Trigger:** go backend, gin framework, kubernetes client, API handler

**Why This Matters:**
The backend (`components/backend/`) is the API gateway. Every feature touches this. It handles auth, RBAC, and K8s client management.

**Skill Would Cover:**
- Gin framework patterns (handlers, middleware, routing)
- Kubernetes client-go usage (dynamic client, typed client)
- User token authentication vs service account
- RBAC enforcement (SelfSubjectAccessReview)
- Error handling patterns (documented in `.claude/patterns/error-handling.md`)
- Request validation and sanitization
- Multi-tenant namespace isolation
- Secret management (runner secrets, Git tokens)
- Logging and observability
- Testing (unit, integration, contract tests)

**Current Gap:**
Backend complexity growing. Patterns exist but not centralized. New developers struggle with K8s client selection (user token vs SA).

**Implementation:**
```
.ambient/skills/go-backend-api/
├── SKILL.md              # Main skill
├── gin-patterns.md       # Handler structure, middleware
├── k8s-client-guide.md   # When to use what client
├── auth-rbac.md          # Token handling, authorization
├── testing-strategy.md   # Test pyramid for backend
└── api-design.md         # REST conventions, versioning
```

**Estimated Impact:** Speeds up API development by 30%, ensures consistent patterns, reduces RBAC bugs.

---

### 4. Amber Background Agent Orchestration ⭐⭐
**Priority:** P1 - Automation System
**Trigger:** amber agent, github actions, issue-to-pr, background automation

**Why This Matters:**
Amber is the automation engine (`agents/amber.md`, `docs/amber-automation.md`). It handles auto-fix, refactoring, and test coverage via GitHub Actions. Complex workflows require deep understanding.

**Skill Would Cover:**
- Amber agent personality and decision-making (from `agents/amber.md`)
- GitHub Actions workflow design (`.github/workflows/`)
- Issue templates and label-based triggering
- Safety protocols (rollback, confidence levels)
- Multi-step automation (discovery → analysis → PR creation)
- Amber config file (`.claude/amber-config.yml`)
- Integration with Claude Code CLI
- PR description templates
- Dependency synchronization (`scripts/sync-amber-dependencies.py`)
- Workflow validation (`scripts/validate-amber-workflows.sh`)

**Current Gap:**
Amber is powerful but complex. New team members struggle with workflow creation. Safety protocols scattered across docs.

**Implementation:**
```
.ambient/skills/amber-orchestration/
├── SKILL.md              # Main skill
├── workflow-design.md    # GHA workflow patterns
├── safety-protocols.md   # Rollback, confidence, risk
├── issue-templates.md    # Template creation guide
└── troubleshooting.md    # Common failure modes
```

**Estimated Impact:** Enables team members to create new Amber workflows, reduces workflow failures, improves automation safety.

---

### 5. NextJS Frontend Development (shadcn/ui) ⭐⭐
**Priority:** P1 - User Interface
**Trigger:** nextjs, react, shadcn ui, frontend, dashboard

**Why This Matters:**
The frontend (`components/frontend/`) uses NextJS 14 App Router, shadcn/ui, and React Query. Design guidelines exist (`DESIGN_GUIDELINES.md`). SpecSmith dashboard coming.

**Skill Would Cover:**
- NextJS App Router patterns (server/client components)
- shadcn/ui component library usage
- React Query patterns (documented in `.claude/patterns/react-query-usage.md`)
- Tailwind CSS conventions
- Design system (from `DESIGN_GUIDELINES.md`)
- Real-time updates (SSE, WebSocket)
- State management (Zustand + React Query)
- Accessibility (ARIA, keyboard navigation)
- Testing (Cypress, Jest, React Testing Library)

**Current Gap:**
Design guidelines exist but not embedded in workflow. React Query patterns documented but not codified as skill.

**Implementation:**
```
.ambient/skills/nextjs-frontend/
├── SKILL.md              # Main skill
├── app-router.md         # NextJS 14 patterns
├── shadcn-ui.md          # Component usage and customization
├── react-query.md        # Data fetching patterns
├── design-system.md      # Design guidelines enforcement
└── testing.md            # Frontend test strategy
```

**Estimated Impact:** Accelerates UI development by 25%, ensures design consistency, reduces prop drilling and state bugs.

---

### 6. Makefile Development & CI/CD Automation ⭐⭐
**Priority:** P1 - Developer Experience
**Trigger:** makefile, build automation, ci/cd, local development

**Why This Matters:**
The `Makefile` (1,000+ lines) is the primary developer interface. It handles builds, deploys, local dev setup, and has quality checks (`validate-makefile`, `makefile-health`).

**Skill Would Cover:**
- Makefile structure and target organization
- PHONY targets and dependency management
- Container engine abstraction (Podman vs Docker)
- Platform-specific builds (amd64, arm64)
- Build metadata injection (Git commit, version)
- Local development workflows (Minikube, CRC, kind)
- CI/CD integration (GitHub Actions)
- Error handling and logging in Make
- Help documentation generation
- Port-forwarding and access patterns

**Current Gap:**
Makefile is well-structured but lacks documented patterns for extension. New targets added inconsistently.

**Implementation:**
```
.ambient/skills/makefile-automation/
├── SKILL.md              # Main skill
├── target-patterns.md    # How to add new targets
├── build-system.md       # Container builds, platforms
├── local-dev.md          # Minikube/CRC patterns
└── ci-integration.md     # GitHub Actions usage
```

**Estimated Impact:** Improves developer onboarding, reduces Makefile bugs, enables self-service automation additions.

---

### 7. SpecSmith Specification-First Development ⭐
**Priority:** P2 - Strategic Initiative
**Trigger:** specsmith, specification-first, dashboard, command precision

**Why This Matters:**
SpecSmith (`SPECSMITH_README.md`, `specsmith-philosophy.md`) is a new strategic direction. It's a cockpit-style dashboard for specification-first development. Design philosophy is well-documented.

**Skill Would Cover:**
- Specification-first development methodology
- Command Precision design philosophy
- Inbox-driven workflow patterns
- Agent orchestration UI (Foundry view)
- Terminal integration (xterm.js)
- Code editor embedding (github.dev iframe)
- AG-UI protocol (backend integration)
- Dashboard layout patterns (three-column)
- Color as signal system (blue/amber/emerald/crimson)
- Typography as command language
- Desktop vs mobile considerations

**Current Gap:**
SpecSmith is planned but not implemented. Future developers will need this philosophy embedded.

**Implementation:**
```
.ambient/skills/specsmith-development/
├── SKILL.md              # Main skill
├── philosophy.md         # Command Precision principles
├── dashboard-patterns.md # Three-column layout, navigation
├── ag-ui-integration.md  # Backend protocol usage
└── design-system.md      # Color, typography, density
```

**Estimated Impact:** Ensures SpecSmith development adheres to vision, reduces rework, maintains design consistency.

---

### 8. Agent Definition & Orchestration ⭐
**Priority:** P2 - Core Platform Capability
**Trigger:** agent definition, multi-agent, vteam agent, agent orchestration

**Why This Matters:**
The platform has 7 active agents (`agents/`) and 16 in bullpen (`agent-bullpen/`). Each has structured YAML frontmatter and personality definitions.

**Skill Would Cover:**
- Agent definition format (YAML frontmatter)
- Agent personality and communication style
- Agent competency levels (SE → Senior SE)
- Problem space definition (questions agent answers)
- Process phases and deliverables
- Tool assignments for agents
- Agent collaboration patterns
- Bullpen management (moving agents in/out of active rotation)
- Agent testing and validation
- When to create new agents vs use existing

**Current Gap:**
Agent creation is ad-hoc. No formalized process for designing, testing, or promoting agents from bullpen to active.

**Implementation:**
```
.ambient/skills/agent-development/
├── SKILL.md              # Main skill
├── definition-format.md  # YAML structure, fields
├── personality-guide.md  # Communication style, competency
├── process-design.md     # Phases, outputs, collaboration
└── lifecycle.md          # Bullpen → active → refinement
```

**Estimated Impact:** Accelerates agent creation, ensures quality, reduces agent role overlap.

---

## Medium-Priority Skill Opportunities

### 9. E2E Testing with Cypress & Kind ⭐
**Priority:** P2 - Quality Assurance
**Trigger:** e2e testing, cypress, kind cluster, integration testing

**Why This Matters:**
E2E tests (`e2e/`) use Cypress with kind (Kubernetes in Docker). Tests run in CI. Setup is complex (kind cluster, deploy, test, cleanup).

**Skill Would Cover:**
- kind cluster setup and configuration
- Deploying vTeam stack to ephemeral clusters
- Cypress test patterns for Kubernetes apps
- Handling async operations (pod startup, CR status)
- Cleanup and teardown strategies
- CI/CD integration (GitHub Actions)
- Test data management
- Troubleshooting test failures

**Current Gap:**
E2E setup documented (`e2e/README.md`) but knowledge not embedded. New tests added inconsistently.

---

### 10. Observability & Monitoring (Langfuse) ⭐
**Priority:** P2 - Production Readiness
**Trigger:** langfuse, observability, tracing, monitoring

**Why This Matters:**
The platform integrates Langfuse for observability (`components/manifests/observability/`). Claude SDK traces are sent to Langfuse. Production monitoring critical.

**Skill Would Cover:**
- Langfuse integration patterns
- Claude SDK trace configuration
- Privacy masking (credential redaction)
- Dashboard setup and visualization
- Alert configuration
- Performance metrics (latency, throughput)
- Cost tracking (token usage)
- Error rate monitoring
- Log correlation

**Current Gap:**
Langfuse integration exists but not fully documented. Monitoring best practices not codified.

---

### 11. Python Claude Code Runner Development ⭐
**Priority:** P2 - Runner System
**Trigger:** claude code runner, python runner, agent execution

**Why This Matters:**
The runner (`components/runners/claude-code-runner/`) executes Claude Code CLI. It's Python-based, uses Claude Agent SDK, and has a comprehensive smoketest suite.

**Skill Would Cover:**
- Claude Code CLI invocation patterns
- Claude Agent SDK usage (now covered by existing skill at `.ambient/skills/claude-sdk-expert/`)
- Subprocess management and lifecycle
- AG-UI protocol translation
- Streaming response handling
- State persistence (conversation continuation)
- Error recovery (resume on failure)
- Tool execution (Read, Write, Bash, etc.)
- Testing strategy (smoketest suite)
- Deployment patterns (Kubernetes Jobs)

**Current Gap:**
Claude SDK skill exists but broader runner context (CLI, deployment, testing) not covered.

---

### 12. Git Provider Abstraction Layer ⭐
**Priority:** P3 - Architecture Pattern
**Trigger:** git provider, multi-provider, github gitlab abstraction

**Why This Matters:**
The platform supports GitHub and GitLab with automatic detection. Abstraction layer enables future providers (Bitbucket, Azure DevOps).

**Skill Would Cover:**
- Provider detection from URLs
- Abstract provider interface design
- GitHub-specific implementation
- GitLab-specific implementation
- Provider-specific error handling
- Multi-provider project support
- Adding new providers (template)
- Testing multi-provider scenarios

**Current Gap:**
Abstraction exists in code but not documented as architectural pattern. Adding new providers would require code archaeology.

---

## Additional Opportunities (Lower Priority)

### 13. OpenShift OAuth Integration
**Trigger:** openshift oauth, authentication, sso
**Docs:** `docs/OPENSHIFT_OAUTH.md`

### 14. MinIO S3 State Persistence
**Trigger:** minio, s3 storage, state persistence
**Makefile:** Targets for `setup-minio`, `minio-console`, `minio-logs`

### 15. Vertex AI Configuration
**Trigger:** vertex ai, google cloud, anthropic vertex
**Docs:** `README.md` section on Vertex AI vs Direct API

### 16. Branch Protection & Git Hooks
**Trigger:** git hooks, branch protection, pre-commit
**Scripts:** `scripts/install-git-hooks.sh`, `scripts/git-hooks/`

### 17. Repomix Analysis & Architecture Views
**Trigger:** repomix, codebase analysis, architecture view
**Docs:** `.claude/repomix-guide.md`

---

## Implementation Roadmap

### Phase 1: Core Infrastructure (P0)
**Target:** Q1 2026
**Skills:** 1-4 (Operator, GitLab, Backend, Amber)
**Rationale:** These touch every feature and have highest ROI

### Phase 2: Developer Experience (P1)
**Target:** Q2 2026
**Skills:** 5-7 (Frontend, Makefile, SpecSmith)
**Rationale:** Improve velocity and enable strategic initiatives

### Phase 3: Quality & Architecture (P2)
**Target:** Q3 2026
**Skills:** 8-12 (Agent Dev, E2E, Observability, Runner, Git Provider)
**Rationale:** Production readiness and architectural patterns

### Phase 4: Specialized Topics (P3)
**Target:** Q4 2026
**Skills:** 13-17 (OAuth, MinIO, Vertex AI, Hooks, Repomix)
**Rationale:** Nice-to-have, narrow use cases

---

## Skill Creation Template

For consistency, all skills should follow this structure:

```
.ambient/skills/<skill-name>/
├── SKILL.md              # Main skill file (Amber loads this)
├── USAGE-FOR-AMBER.md    # Optional: How Amber should use this
├── <domain-1>.md         # Sub-domain deep dives
├── <domain-2>.md
└── examples/             # Code examples, templates
    ├── example-1.go
    └── example-2.py
```

**SKILL.md Format:**
- Overview and trigger conditions
- Core concepts and terminology
- Common patterns and anti-patterns
- Decision trees ("When to use X vs Y")
- Quick reference (imports, commands, snippets)
- Troubleshooting guide
- Links to related skills and docs

---

## Success Metrics

**Developer Velocity:**
- Time to implement new feature (target: -30%)
- Time to onboard new developer (target: -50%)
- Time to troubleshoot issues (target: -40%)

**Code Quality:**
- Reduced bug rate in skill-covered areas (target: -25%)
- Pattern consistency across codebase (target: 90%+)
- Test coverage in skill-covered areas (target: 80%+)

**Knowledge Distribution:**
- Reduced "ask an expert" queries (target: -60%)
- Increased self-service problem solving (target: +50%)
- Faster PR reviews (target: -30% review time)

---

## Recommendations

1. **Start with Kubernetes Operator skill** - Highest impact, deepest complexity, core to platform
2. **Create GitLab Integration skill next** - Active development area, reduce support burden
3. **Build Go Backend API skill alongside** - Backend changes touch every feature
4. **Develop Amber skill for team enablement** - Empower non-experts to create automations
5. **Document SpecSmith philosophy before implementation** - Prevent drift from vision
6. **Extract patterns from existing `.claude/patterns/` and `.claude/context/` files** - Don't reinvent, formalize
7. **Use Amber to help create skills** - Meta: Amber can assist in documenting its own expertise
8. **Test skills with new contributors** - Validate that skills reduce onboarding time
9. **Version skills alongside platform** - Skills should evolve with codebase
10. **Create skill for skill creation** - Meta-skill using the `skill-creator` skill as foundation

---

## Existing Assets to Leverage

**Already Documented:**
- `.claude/patterns/` - Error handling, K8s client, React Query
- `.claude/context/` - Backend, frontend, security standards
- `docs/adr/` - Architectural decisions (WHY, not just WHAT)
- `agents/` - Agent definitions with personality and process
- `docs/claude-agent-sdk/` - Comprehensive SDK documentation
- `.ambient/skills/claude-sdk-expert/` - Existing skill model

**Tools & Scripts:**
- `scripts/sync-amber-dependencies.py` - Dependency management
- `scripts/validate-amber-workflows.sh` - Workflow validation
- `scripts/install-git-hooks.sh` - Git hook setup
- `Makefile` targets - `validate-makefile`, `makefile-health`

**Testing Infrastructure:**
- `e2e/` - Cypress tests with kind
- `components/runners/claude-code-runner/tests/smoketest/` - SDK smoke tests
- `components/backend/tests/integration/` - Backend integration tests

---

## Conclusion

The Ambient Code Platform has **exceptional potential for skills development**. The codebase is large, well-documented, and has clear domain boundaries. Existing patterns, agents, and documentation provide strong foundations.

**Key Strengths:**
- Existing skill model (Claude SDK Expert) proves viability
- Strong documentation culture (ADRs, patterns, context files)
- Clear architectural boundaries (operator, backend, frontend, runner)
- Active agent definitions with personality and process
- Complex domains requiring specialized expertise (K8s, GitLab, Amber)

**Quick Wins:**
- Formalize existing patterns (error-handling.md → skill)
- Extract GitLab docs (4 files → 1 skill)
- Document Amber workflows (2 docs + scripts → skill)
- Codify SpecSmith philosophy (3 docs → skill)

**Next Steps:**
1. Review this analysis with team
2. Prioritize top 3-5 skills based on current pain points
3. Create first skill (recommend: Kubernetes Operator)
4. Validate with new contributor onboarding
5. Iterate and expand

The investment in skills will compound over time, reducing friction, distributing knowledge, and enabling faster, higher-quality development.
