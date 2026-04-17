<!--
Sync Impact Report - Constitution Update
Version: 1.1.0
Last Updated: 2026-04-10

Changelog (v1.1.0):
  - CLAUDE.md + BOOKMARKS.md declared authoritative for project conventions
  - Removed duplicated Development Standards (Go, Frontend, Python, Naming) — now defers to CLAUDE.md/BOOKMARKS.md
  - Removed duplicated Pre-Deployment Validation commands — now defers to CLAUDE.md
  - Updated Governance/Compliance: removed "Constitution supersedes all" language
  - Added Authority section establishing CLAUDE.md precedence for shared conventions
  - Kept all 10 core principles, Production Requirements, and spec-kit governance

Changelog (v1.0.0):
  - RATIFIED: Constitution officially ratified and adopted
  - Version bump: v0.2.0 (DRAFT) → v1.0.0 (RATIFIED)
  - Ratification Date: 2025-11-13
  - All 10 core principles now in force
  - Development standards and governance policies active

Changelog (v0.2.0):
  - Added Development Standards: Naming & Legacy Migration subsection
    * Naming guidance for legacy vTeam references
    * DO NOT update without explicit instruction: API groups, CRDs, container names, K8s resources
    * Safe to update: docs, comments, logs, UI text, new variable names

Changelog (v0.1.0):
  - Added Principle X: Commit Discipline & Code Review
    * Line count thresholds by change type (bugfix ≤150, feature ≤300/500, refactor ≤400)
    * Mandatory exceptions for generated code, migrations, dependencies
    * Conventional commit format requirements
    * PR size limits (600 lines) with justification requirements
    * Measurement guidelines (what counts vs excluded)

Changelog (v0.0.1):
  - Added Principle VIII: Context Engineering & Prompt Optimization
  - Added Principle IX: Data Access & Knowledge Augmentation
  - Enhanced Principle IV: E2E testing, coverage standards, CI/CD automation
  - Enhanced Principle VI: /metrics endpoint REQUIRED, simplified key metrics guidance
  - Simplified Principle IX: Consolidated RAG/MCP/RLHF into concise bullets
  - Removed redundant test categories section in Principle IV
  - Consolidated Development Standards: Reference principles instead of duplicating
  - Consolidated Production Requirements: Reference principles, add only unique items
  - Reduced total length by ~30 lines while maintaining clarity

Templates Status:
  ✅ plan-template.md - References constitution check dynamically
  ✅ tasks-template.md - Added Phase 3.9 for commit planning/validation (T036-T040)
  ✅ spec-template.md - No updates needed

Follow-up TODOs:
  - Implement /metrics endpoints in all components
  - Create prompt template library
  - Design RAG pipeline architecture
  - Add commit size validation tooling (pre-commit hook or CI check)
  - Update PR template to include commit discipline checklist
  - Do NOT rename Kubernetes API group (vteam.ambient-code), CRDs, or container image names without explicit instruction
-->

# ACP Constitution

## Core Principles

### I. Kubernetes-Native Architecture

All features MUST be built using Kubernetes primitives and patterns:

- Custom Resource Definitions (CRDs) for domain objects (AgenticSession, ProjectSettings, RFEWorkflow)
- Operators for reconciliation loops and lifecycle management
- Jobs for execution workloads with proper resource limits
- ConfigMaps and Secrets for configuration management
- Services and Routes for network exposure
- RBAC for authorization boundaries

**Rationale**: Kubernetes-native design ensures portability, scalability, and enterprise-grade operational tooling. Violations create operational complexity and reduce platform value.

### II. Security & Multi-Tenancy First

Security and isolation MUST be embedded in every component. User token auth, RBAC, token redaction, project-scoped isolation, least privilege, and container security hardening are all non-negotiable. See CLAUDE.md Critical Context and [Security Standards](.claude/context/security-standards.md) for implementation details.

**Rationale**: Security breaches and privilege escalation destroy trust. Multi-tenant isolation is non-negotiable for enterprise deployment.

### III. Type Safety & Error Handling (NON-NEGOTIABLE)

No panics, no `any` types, explicit error wrapping with context. See CLAUDE.md Critical Context and [Error Handling Patterns](.claude/patterns/error-handling.md) for implementation details.

**Rationale**: Runtime panics crash operator loops and kill services. Explicit error handling ensures debuggability and operational stability.

### IV. Test-Driven Development

TDD is MANDATORY for all new functionality:

- **Contract Tests**: Every API endpoint/library interface MUST have contract tests
- **Integration Tests**: Multi-component interactions MUST have integration tests
- **Unit Tests**: Business logic MUST have unit tests
- **Permission Tests**: RBAC boundary validation
- **E2E Tests**: Critical user journeys MUST have end-to-end tests
- **Red-Green-Refactor**: Tests written → Tests fail → Implementation → Tests pass → Refactor

**Coverage Standards**:

- Maintain high test coverage across all categories
- Critical paths MUST have comprehensive test coverage
- CI/CD pipeline MUST enforce test passing before merge
- Coverage reports generated automatically in CI

**Rationale**: Tests written after implementation miss edge cases and don't drive design. TDD ensures testability, catches regressions, and documents expected behavior.

### V. Component Modularity

Code MUST be organized into clear, single-responsibility modules:

- **Handlers**: HTTP/watch logic ONLY, no business logic
- **Types**: Pure data structures, no methods or business logic
- **Services**: Reusable business logic, no direct HTTP handling
- **No Cyclic Dependencies**: Package imports must form a DAG
- **Frontend Colocation**: Single-use components colocated with pages, reusable components in `/components`
- **File Size Limit**: Components over 200 lines MUST be broken down

**Rationale**: Modular architecture enables parallel development, simplifies testing, and reduces cognitive load. Cyclic dependencies create maintenance nightmares.

### VI. Observability & Monitoring

All components MUST support operational visibility:

- **Structured Logging**: Use structured logs with context (namespace, resource, operation)
- **Health Endpoints**: `/health` endpoints for all services (liveness, readiness)
- **Metrics Endpoints**: `/metrics` endpoints REQUIRED for all services (Prometheus format)
- **Status Updates**: Use `UpdateStatus` subresource for CR status changes
- **Event Emission**: Kubernetes events for operator actions
- **Error Context**: Errors must include actionable context for debugging
- **Key Metrics**: Expose latency percentiles (p50/p95/p99), error rates, throughput, and component-specific operational metrics aligned with project goals

**Metrics Standards**:

- Prometheus format on dedicated management port
- Standard labels: service, namespace, version
- Focus on metrics critical to project success (e.g., session execution time for ACP)

**Rationale**: Production systems fail. Without observability, debugging is impossible and MTTR explodes. Metrics enable proactive monitoring and capacity planning.

### VII. Resource Lifecycle Management

All child resources MUST have OwnerReferences with cascading deletes. Idempotent creation, no BlockOwnerDeletion, goroutine cleanup on parent deletion. See CLAUDE.md Critical Context for the key rules.

**Rationale**: Resource leaks waste cluster capacity and cause outages. Proper lifecycle management ensures automatic cleanup and prevents orphaned resources.

### VIII. Context Engineering & Prompt Optimization

ACP is a context engineering hub - AI output quality depends on input quality:

- **Context Budgets**: Respect token limits (200K for Claude Sonnet 4.5)
- **Context Prioritization**: System context > conversation history > examples
- **Prompt Templates**: Use standardized templates for common operations (RFE analysis, code review)
- **Context Compression**: Summarize long-running sessions to preserve history within budget
- **Agent Personas**: Maintain consistency through well-defined agent roles
- **Pre-Deployment Optimization**: ALL prompts MUST be optimized for clarity and token efficiency before deployment
- **Incremental Loading**: Build context incrementally, avoid reloading static content

**Rationale**: Poor context management causes hallucinations, inconsistent outputs, and wasted API costs. Context engineering is a first-class engineering discipline for AI platforms.

### IX. Data Access & Knowledge Augmentation

Enable agents to access external knowledge and learn from interactions:

- **RAG**: Embed and index repository contents, chunk semantically (512-1024 tokens), use consistent models, apply reranking
- **MCP**: Support MCP servers for structured data access, enforce namespace isolation, handle failures gracefully
- **RLHF**: Capture user ratings (thumbs up/down), store with session metadata, refine prompts from patterns, support A/B testing

**Rationale**: Static prompts have limited effectiveness. Platforms must continuously improve through knowledge retrieval and learning from user feedback.

### X. Commit Discipline & Code Review

Each commit MUST be atomic, reviewable, and independently testable:

**Line Count Thresholds** (excludes generated code, test fixtures, vendor/deps):

- **Bug Fix**: ≤150 lines
  - Single issue resolution
  - Includes test demonstrating the bug
  - Includes fix verification

- **Feature (Small)**: ≤300 lines
  - Single user-facing capability
  - Includes unit + contract tests
  - Updates relevant documentation

- **Feature (Medium)**: ≤500 lines
  - Multi-component feature
  - Requires design justification in commit message
  - MUST be reviewable in 30 minutes

- **Refactoring**: ≤400 lines
  - Behavior-preserving changes only
  - MUST NOT mix with feature/bug changes
  - Existing tests MUST pass unchanged

- **Documentation**: ≤200 lines
  - Pure documentation changes
  - Can be larger for initial docs

- **Test Addition**: ≤250 lines
  - Adding missing test coverage
  - MUST NOT include implementation changes

**Mandatory Exceptions** (requires justification in PR description):

- **Code Generation**: Generated CRD YAML, OpenAPI schemas, protobuf
- **Data Migration**: Database migrations, fixture updates
- **Dependency Updates**: go.mod, package.json, requirements.txt
- **Configuration**: Kubernetes manifests for new components (≤800 lines)

**Commit Requirements**:

- **Atomic**: Single logical change that can be independently reverted
- **Self-Contained**: Each commit MUST pass all tests and linters
- **Conventional Format**: `type(scope): description`
  - Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `perf`, `ci`
  - Scope: component name (backend, frontend, operator, runner)
- **Message Content**: Explain WHY, not WHAT (code shows what)
- **No WIP Commits**: Squash before PR submission

**Review Standards**:

- PR over 600 lines MUST be broken into multiple PRs
- Each commit reviewed independently (enable per-commit review in GitHub)
- Large PRs require design doc or RFC first
- Incremental delivery preferred over "big bang" merges

**Measurement** (what counts toward limits):

- ✅ Source code (`*.go`, `*.ts`, `*.tsx`, `*.py`)
- ✅ Configuration specific to feature (new YAML, JSON)
- ✅ Test code
- ❌ Generated code (CRDs, OpenAPI, mocks)
- ❌ Lock files (`go.sum`, `package-lock.json`)
- ❌ Vendored dependencies
- ❌ Binary files

**Rationale**: Large commits hide bugs, slow reviews, complicate bisecting, and create merge conflicts. Specific thresholds provide objective guidance while exceptions handle legitimate cases. Small, focused commits enable faster feedback, easier debugging (git bisect), and safer reverts.

## Development Standards

Per-language conventions (Go formatting, frontend patterns, Python tooling), build commands, pre-deployment validation, and naming/legacy migration rules are maintained in [`/CLAUDE.md`](/CLAUDE.md) and [`/BOOKMARKS.md`](/BOOKMARKS.md). Those files are the authoritative source — do not duplicate them here.

### Production Requirements

- Scan container images for vulnerabilities before deployment
- Set up centralized logging and alerting infrastructure
- Configure Horizontal Pod Autoscaling based on CPU/memory
- Set appropriate resource requests and limits
- Plan for job concurrency and queue management
- Design for multi-tenancy with shared infrastructure
- Do not use etcd as a database for unbounded objects like CRs. Use an external database like Postgres.

## Governance

### Amendment Process

1. **Proposal**: Document proposed change with rationale
2. **Review**: Evaluate impact on existing code and templates
3. **Approval**: Requires project maintainer approval
4. **Migration**: Update all dependent templates and documentation
5. **Versioning**: Increment version according to semantic versioning

### Version Policy

- **MAJOR**: Backward incompatible governance/principle removals or redefinitions
- **MINOR**: New principle/section added or materially expanded guidance
- **PATCH**: Clarifications, wording, typo fixes, non-semantic refinements

### Authority

[`/CLAUDE.md`](/CLAUDE.md) and [`/BOOKMARKS.md`](/BOOKMARKS.md) are the authoritative source of project conventions. This constitution covers spec-kit-specific governance (principles, commit discipline, amendment process) and defers to those files for shared conventions. If they conflict, CLAUDE.md wins.

### Compliance

- All pull requests MUST verify constitution compliance for spec-kit-governed concerns
- Complexity violations MUST be justified in implementation plans

**Version**: 1.1.0 | **Ratified**: 2025-11-13 | **Last Amended**: 2026-04-10
