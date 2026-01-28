# ADR-0006: Amber LangGraph Implementation

**Status:** Proposed
**Date:** 2026-01-27
**Author:** Jeremy Eder

## Context

Amber is the Ambient Code Platform's background agent for codebase intelligence and autonomous maintenance. Currently defined as a persona specification in `agents/amber.md`, Amber needs a production implementation that can execute its four operational modes (on-demand, background, scheduled, webhook-triggered) within ACP's Kubernetes-native architecture.

LangGraph provides a framework for building stateful, long-running agents with explicit state management, checkpointing, and error handling. This ADR proposes implementing Amber using LangGraph to enable production-ready execution, human-in-the-loop workflows, and integration with ACP's existing infrastructure (AgenticSession CRDs, Langfuse observability, GitHub API).

### Current State

- Amber exists as a comprehensive persona specification (12 pages, 450+ lines)
- No executable implementation
- ACP uses Claude Code SDK for agentic sessions via `claude-code-runner`
- AgenticSession CRD manages session lifecycle, multi-repo support, status tracking
- Backend API creates CRs, Operator spawns Jobs, Runner pods execute sessions

### Problem

Building Amber as a traditional script or single-file agent would lack:

1. **State persistence**: Background and scheduled modes need to maintain context across executions
2. **Checkpointing**: Human-in-the-loop approval requires pausing and resuming execution
3. **Error resilience**: Network failures, API timeouts, and transient errors need retry logic
4. **Mode switching**: Amber must adapt behavior based on invocation context (interactive vs autonomous)
5. **Tool orchestration**: Complex workflows (issue triage → analysis → PR creation) need explicit orchestration

LangGraph addresses these requirements through StateGraph, checkpointers, retry policies, and conditional routing.

## Decision

Implement Amber as a LangGraph-based agent deployed as a new runner component (`components/runners/amber-runner/`) following ACP's existing runner pattern. The implementation will:

1. Use LangGraph StateGraph for workflow orchestration
2. Integrate with AgenticSession CRD for lifecycle management
3. Deploy as containerized Jobs spawned by the Operator
4. Use PostgreSQL checkpointer for cross-execution state persistence
5. Integrate with GitHub API via MCP server or direct API calls
6. Support Langfuse tracing for observability

### Architecture Overview

```
User/Webhook → Backend API → AgenticSession CR → Operator → Amber Job Pod
                                                                    ↓
                                        LangGraph Agent (StateGraph)
                                                ↓           ↓
                                        PostgreSQL     Langfuse
                                        (checkpoints)  (traces)
                                                ↓
                                        GitHub API (via MCP/direct)
```

### Component Structure

```
components/runners/amber-runner/
├── amber_agent.py          # LangGraph StateGraph definition
├── nodes/
│   ├── triage.py          # Issue triage node
│   ├── analyze.py         # Root cause analysis node
│   ├── implement.py       # Auto-fix implementation node
│   ├── pr_creation.py     # PR creation and formatting node
│   └── review.py          # Code review node
├── tools/
│   ├── github_tools.py    # GitHub API integration
│   ├── k8s_tools.py       # Kubernetes API integration
│   └── codebase_tools.py  # Code search, analysis, pattern detection
├── state.py               # AmberState schema definition
├── server.py              # FastAPI wrapper (AG-UI protocol)
├── context.py             # RunnerContext integration
├── requirements.txt       # Dependencies
└── Dockerfile             # Container image

components/manifests/base/
└── amber-runner/
    ├── deployment.yaml    # Amber runner deployment
    ├── service.yaml       # Service exposure
    └── rbac.yaml          # RBAC permissions
```

## LangGraph Implementation Design

### State Schema

The `AmberState` TypedDict defines the shared state across all nodes:

```python
from typing import TypedDict, Literal, Annotated
from langgraph.types import add_messages

class AmberState(TypedDict):
    """Shared state for Amber agent execution."""

    # Execution context
    mode: Literal["on-demand", "background", "scheduled", "webhook"]
    thread_id: str
    run_id: str
    session_id: str

    # Conversation history (LangGraph's add_messages reducer)
    messages: Annotated[list, add_messages]

    # Amber-specific context
    github_issue: Optional[dict]  # Issue data for background/webhook modes
    github_pr: Optional[dict]     # PR data for review mode
    github_repo: str              # Target repository

    # Task tracking
    current_task: str
    tasks_completed: list[str]
    confidence_level: Literal["high", "medium", "low"]

    # Results
    analysis_results: Optional[dict]
    pr_created: Optional[str]     # PR URL if created
    error: Optional[str]

    # Safety & trust
    human_approval_required: bool
    rollback_instructions: Optional[str]
```

### Graph Structure

```python
from langgraph.graph import StateGraph, START, END
from langgraph.checkpoint.postgres import PostgresSaver
from langgraph.types import RetryPolicy

# Initialize checkpointer
checkpointer = PostgresSaver.from_conn_string(os.environ["POSTGRES_URL"])

# Define retry policy for external API calls
api_retry = RetryPolicy(max_attempts=3, exponential_backoff=True)

# Build graph
graph = StateGraph(AmberState)

# Add nodes
graph.add_node("determine_mode", determine_mode_node)
graph.add_node("triage_issue", triage_issue_node, retry=api_retry)
graph.add_node("analyze_root_cause", analyze_root_cause_node)
graph.add_node("plan_implementation", plan_implementation_node)
graph.add_node("request_approval", request_approval_node)  # Human-in-the-loop
graph.add_node("implement_fix", implement_fix_node, retry=api_retry)
graph.add_node("create_pr", create_pr_node, retry=api_retry)
graph.add_node("report_results", report_results_node)

# Define edges
graph.add_edge(START, "determine_mode")

# Conditional routing based on mode
graph.add_conditional_edges(
    "determine_mode",
    route_by_mode,
    {
        "on-demand": "analyze_root_cause",       # User asks question
        "background": "triage_issue",            # Issue-to-PR workflow
        "scheduled": "report_results",           # Health check report
        "webhook": "triage_issue",               # New issue webhook
    }
)

# Background/webhook flow: triage → analyze → plan → approval → implement → PR
graph.add_edge("triage_issue", "analyze_root_cause")
graph.add_edge("analyze_root_cause", "plan_implementation")
graph.add_conditional_edges(
    "plan_implementation",
    check_approval_needed,
    {
        "needs_approval": "request_approval",
        "auto_fixable": "implement_fix",
        "escalate": "report_results",
    }
)
graph.add_edge("request_approval", "implement_fix")  # Resumes after human approval
graph.add_edge("implement_fix", "create_pr")
graph.add_edge("create_pr", "report_results")
graph.add_edge("report_results", END)

# Compile with checkpointer
app = graph.compile(checkpointer=checkpointer)
```

### Node Implementation Examples

**Triage Issue Node**:

```python
from langchain_anthropic import ChatAnthropic
from langgraph.prebuilt import ToolNode

async def triage_issue_node(state: AmberState) -> dict:
    """
    Triage incoming GitHub issue: severity, component, related issues.

    Uses Claude to analyze issue content and assign labels.
    Returns partial state update with triage results.
    """
    issue = state["github_issue"]

    llm = ChatAnthropic(model="claude-sonnet-4-5-20250929")

    triage_prompt = f"""
You are Amber, the ACP codebase intelligence agent. Triage this GitHub issue:

Title: {issue['title']}
Body: {issue['body']}

Analyze:
1. Severity (P0/P1/P2/P3)
2. Component (frontend/backend/operator/runner)
3. Related issues (search for similar patterns)
4. Auto-fixable? (yes/no with confidence)

Return JSON: {{"severity": "P2", "component": "backend", "auto_fixable": true, "confidence": "high"}}
"""

    response = await llm.ainvoke([{"role": "user", "content": triage_prompt}])
    triage_results = json.loads(response.content)

    return {
        "analysis_results": triage_results,
        "confidence_level": triage_results["confidence"],
        "current_task": "triage_completed",
        "tasks_completed": state["tasks_completed"] + ["triage"],
    }
```

**Request Approval Node** (Human-in-the-Loop):

```python
from langgraph.checkpoint import Checkpoint

async def request_approval_node(state: AmberState) -> dict:
    """
    Pause execution and request human approval.

    This node creates a checkpoint and waits for user to resume with approval.
    In ACP, this translates to updating AgenticSession status to "WaitingForApproval"
    and emitting an event for the frontend to display approval UI.
    """
    plan = state["analysis_results"]

    approval_message = f"""
## Amber Requests Approval

**Proposed Fix:**
{plan['implementation_plan']}

**Confidence:** {state['confidence_level']} ({plan['confidence_score']}%)

**Risk Assessment:** {plan['risk']}

**Rollback:**
```bash
{plan['rollback_instructions']}
```

React 👍 to approve, 👎 to reject, 💬 to request changes.
"""

    # Emit approval request event (AG-UI protocol)
    # Frontend displays approval UI, waits for user response
    # Execution pauses here until graph.update_state() called with approval

    return {
        "human_approval_required": True,
        "current_task": "waiting_for_approval",
        "messages": [{"role": "assistant", "content": approval_message}],
    }
```

### Mode-Specific Behaviors

**On-Demand Mode** (Interactive Consultation):

- Entry point: User creates AgenticSession with Amber workflow
- Graph flow: START → determine_mode → analyze_root_cause → report_results → END
- No auto-implementation, only analysis and recommendations
- Returns file references, root cause, suggested fixes

**Background Mode** (Autonomous Maintenance):

- Entry point: GitHub webhook creates AgenticSession
- Graph flow: START → determine_mode → triage → analyze → plan → approval → implement → PR → END
- Auto-fixable issues proceed to implementation after approval
- Creates PRs with detailed descriptions following safety protocol

**Scheduled Mode** (Periodic Health Checks):

- Entry point: CronJob creates AgenticSession nightly/weekly
- Graph flow: START → determine_mode → report_results → END
- Generates markdown reports in `docs/amber-reports/`
- Commits report to feature branch, opens PR

**Webhook Mode** (Reactive Intelligence):

- Entry point: GitHub issue/PR webhook
- Graph flow: Same as background mode for issues, custom flow for PR reviews
- Comments on issues/PRs with analysis
- Only comments if adding unique value (high signal, low noise)

## Integration with ACP Infrastructure

### AgenticSession CRD Integration

Amber runner maps LangGraph execution to AgenticSession lifecycle:

1. **Session Creation**: Backend creates AgenticSession with `activeWorkflow.gitUrl` pointing to Amber workflow repo
2. **Job Spawning**: Operator watches CR, creates Job pod with amber-runner image
3. **Execution**: Runner initializes LangGraph graph, executes based on `spec.initialPrompt` and mode
4. **Status Updates**: Runner updates CR status via Kubernetes API:
   - `phase: Running` → `phase: WaitingForApproval` → `phase: Running` → `phase: Completed`
5. **Results**: Final state stored in `status.results`, PR URL in `status.pr_created`

### Kubernetes RBAC

Amber runner needs permissions to:

- Read/write AgenticSession CRs (status updates)
- Read ProjectSettings (API keys, configuration)
- Read/write Secrets (GitHub tokens)
- Create ConfigMaps (for reports)

RBAC defined in `components/manifests/base/amber-runner/rbac.yaml`:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: amber-runner
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: amber-runner
rules:
- apiGroups: ["vteam.ambient-code"]
  resources: ["agenticsessions", "projectsettings"]
  verbs: ["get", "list", "watch", "update", "patch"]
- apiGroups: ["vteam.ambient-code"]
  resources: ["agenticsessions/status"]
  verbs: ["update", "patch"]
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get"]
```

### GitHub Integration

Two approaches evaluated:

1. **MCP Server** (Recommended): Use GitHub MCP server for tool integration
   - Advantages: Standardized interface, built-in error handling, supports authentication
   - Disadvantages: Adds dependency, requires MCP server deployment

2. **Direct PyGithub**: Use PyGithub library directly
   - Advantages: Simpler, no additional services
   - Disadvantages: Manual error handling, authentication management, API rate limiting

**Decision**: Start with direct PyGithub for simplicity, migrate to MCP server if GitHub integration becomes complex.

### Langfuse Integration

Amber runner inherits Langfuse integration from existing claude-code-runner pattern:

```python
from langfuse import Langfuse
from langfuse.decorators import observe

# Initialize Langfuse client
langfuse = Langfuse(
    public_key=os.environ["LANGFUSE_PUBLIC_KEY"],
    secret_key=os.environ["LANGFUSE_SECRET_KEY"],
    host=os.environ["LANGFUSE_HOST"],
)

@observe(name="amber-execution")
async def execute_amber_session(state: AmberState):
    """Traced execution with Langfuse."""
    # LangGraph execution automatically traced
    # Custom metadata for Amber-specific tracking
    langfuse.trace(
        name="amber-session",
        metadata={
            "mode": state["mode"],
            "session_id": state["session_id"],
            "github_repo": state["github_repo"],
        }
    )
```

### Checkpointer Setup

Use PostgreSQL for production-grade state persistence:

```python
from langgraph.checkpoint.postgres import PostgresSaver

# PostgreSQL connection from environment
checkpointer = PostgresSaver.from_conn_string(
    os.environ.get("POSTGRES_URL", "postgresql://user:pass@localhost:5432/amber")
)

# Compile graph with checkpointer
app = graph.compile(
    checkpointer=checkpointer,
    interrupt_before=["request_approval"],  # Always pause before approval
)
```

PostgreSQL deployment:

- Option 1: Reuse existing ACP PostgreSQL instance (if available)
- Option 2: Deploy dedicated PostgreSQL StatefulSet for Amber
- Option 3: Use external managed PostgreSQL (AWS RDS, Azure Database)

**Recommendation**: Start with Option 2 (dedicated StatefulSet) for isolation, migrate to shared instance if appropriate.

## Deployment Architecture

### Container Image

Amber runner Dockerfile follows claude-code-runner pattern:

```dockerfile
FROM python:3.11-slim

WORKDIR /app

# Install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY amber_agent.py .
COPY nodes/ nodes/
COPY tools/ tools/
COPY state.py .
COPY server.py .
COPY context.py .

# Non-root user
RUN useradd -m -u 1000 amber
USER amber

EXPOSE 8000

CMD ["python", "server.py"]
```

### Job Template

Operator creates Jobs using this template:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: amber-session-<session-id>
  namespace: <project-namespace>
spec:
  template:
    spec:
      serviceAccountName: amber-runner
      containers:
      - name: amber-runner
        image: quay.io/ambient_code/amber-runner:latest
        env:
        - name: SESSION_ID
          value: "<session-id>"
        - name: POSTGRES_URL
          valueFrom:
            secretKeyRef:
              name: amber-postgres-credentials
              key: connection-string
        - name: LANGFUSE_PUBLIC_KEY
          valueFrom:
            secretKeyRef:
              name: langfuse-credentials
              key: public-key
        - name: GITHUB_TOKEN
          valueFrom:
            secretKeyRef:
              name: github-credentials
              key: token
        - name: ANTHROPIC_API_KEY
          valueFrom:
            secretKeyRef:
              name: anthropic-api-key
              key: api-key
        volumeMounts:
        - name: workspace
          mountPath: /workspace
      volumes:
      - name: workspace
        persistentVolumeClaim:
          claimName: amber-workspace-<session-id>
      restartPolicy: Never
```

## Error Handling and Safety

### Retry Policies

LangGraph RetryPolicy applied to nodes with external API calls:

```python
from langgraph.types import RetryPolicy

# GitHub API calls
github_retry = RetryPolicy(
    max_attempts=3,
    exponential_backoff=True,
    initial_interval=1.0,
    backoff_factor=2.0,
)

graph.add_node("create_pr", create_pr_node, retry=github_retry)
```

### Error Recovery

Implement fallback nodes for critical failures:

```python
graph.add_conditional_edges(
    "implement_fix",
    check_implementation_success,
    {
        "success": "create_pr",
        "failure": "escalate_to_human",  # Fallback node
    }
)

async def escalate_to_human_node(state: AmberState) -> dict:
    """Fallback: create issue comment requesting human intervention."""
    error_msg = f"""
🚨 Amber encountered an error during implementation:

**Error:** {state['error']}

**Context:** {state['current_task']}

**Confidence:** {state['confidence_level']}

I've paused execution. Please review and provide guidance.
"""
    # Post comment to GitHub issue
    # Update AgenticSession status to Failed
    return {"phase": "Failed", "error": error_msg}
```

### Safety Guardrails

Implement constitution compliance checks:

1. **Pre-commit Checks**: Run linters before PR creation (gofmt, black, golangci-lint)
2. **Test Execution**: Never skip tests (Principle IV)
3. **Commit Size**: Validate line count thresholds (Principle X)
4. **Type Safety**: Reject implementations with `panic()` or unsafe type assertions (Principle III)

```python
async def validate_fix_node(state: AmberState) -> dict:
    """Validate fix against constitution before proceeding."""
    fix_code = state["analysis_results"]["implementation"]

    violations = []

    # Check for forbidden patterns
    if "panic(" in fix_code:
        violations.append("Contains panic() (Principle III violation)")

    if len(fix_code.split("\n")) > 150:  # Bug fix threshold
        violations.append("Exceeds bug fix line limit (Principle X)")

    if violations:
        return {
            "error": "Constitution violations detected: " + ", ".join(violations),
            "phase": "Failed",
        }

    return {"current_task": "validation_passed"}
```

## Constitution Compliance

This implementation adheres to ACP Constitution principles:

- **Principle I (Kubernetes-Native)**: Deployed as Jobs, uses CRDs, RBAC, ConfigMaps
- **Principle II (Security)**: Uses service account with minimal permissions, no token logging
- **Principle III (Type Safety)**: Python type hints, explicit error handling, no unsafe operations
- **Principle IV (TDD)**: Includes test suite for nodes, tools, state management
- **Principle V (Modularity)**: Clear separation: nodes/, tools/, state.py, server.py
- **Principle VI (Observability)**: Langfuse integration, structured logging, status updates
- **Principle VII (Lifecycle)**: OwnerReferences on Jobs, automatic cleanup
- **Principle VIII (Context Engineering)**: Prompts optimized for Claude Sonnet 4.5, context budgets respected
- **Principle IX (Knowledge Augmentation)**: GitHub API integration, codebase search tools
- **Principle X (Commit Discipline)**: Amber enforces line count thresholds, conventional commits

## Testing Strategy

### Unit Tests

Test individual nodes and tools:

```python
# tests/test_triage_node.py
import pytest
from amber_agent import triage_issue_node, AmberState

@pytest.mark.asyncio
async def test_triage_node_p0_severity():
    """Test triage correctly identifies P0 issues."""
    state = AmberState(
        mode="background",
        github_issue={
            "title": "Production down: backend crashes on startup",
            "body": "Cluster-wide outage...",
        },
        messages=[],
        tasks_completed=[],
    )

    result = await triage_issue_node(state)

    assert result["analysis_results"]["severity"] == "P0"
    assert result["confidence_level"] in ["high", "medium"]
```

### Integration Tests

Test graph execution end-to-end:

```python
# tests/test_amber_graph.py
import pytest
from amber_agent import app, AmberState

@pytest.mark.asyncio
async def test_background_mode_auto_fix():
    """Test background mode auto-fixes trivial issue."""
    initial_state = AmberState(
        mode="background",
        session_id="test-session",
        github_issue={
            "title": "Fix typo in README",
            "body": "README has typo: 'kuberntes' should be 'kubernetes'",
        },
        github_repo="ambient-code/platform",
        messages=[],
        tasks_completed=[],
    )

    # Execute graph
    result = await app.ainvoke(initial_state, config={"configurable": {"thread_id": "test-thread"}})

    # Assert successful execution
    assert result["phase"] == "Completed"
    assert result["pr_created"] is not None
    assert "Fix typo" in result["pr_created"]
```

### Contract Tests

Verify AgenticSession CR integration:

```python
# tests/test_k8s_integration.py
def test_status_update_contract():
    """Verify status updates match AgenticSession CRD schema."""
    status_update = {
        "phase": "WaitingForApproval",
        "current_task": "implementation_planned",
        "approval_required": True,
    }

    # Validate against CRD OpenAPI schema
    assert validate_cr_status(status_update, "AgenticSession")
```

## Migration Path

### Phase 1: Proof of Concept (2 weeks)

1. Implement core LangGraph graph with 3 nodes: triage, analyze, report
2. Single mode: on-demand (interactive consultation)
3. No GitHub integration (mock data)
4. Deploy to dev namespace, test via AgenticSession CR

**Success Criteria**: Amber answers questions about codebase, provides file references

### Phase 2: Background Mode (3 weeks)

1. Add GitHub API integration (PyGithub)
2. Implement PR creation node
3. Add human-in-the-loop approval node
4. Deploy PostgreSQL checkpointer
5. Test issue-to-PR workflow

**Success Criteria**: Amber triages issue, creates PR with fix, requests approval

### Phase 3: Production Readiness (3 weeks)

1. Add scheduled and webhook modes
2. Implement Langfuse observability
3. Add comprehensive error handling and fallback nodes
4. Write test suite (unit, integration, contract)
5. Deploy to production namespace

**Success Criteria**: Amber operates autonomously, handles failures gracefully, integrates with CI/CD

### Phase 4: Advanced Features (4 weeks)

1. Implement upstream dependency monitoring
2. Add pattern detection across issues
3. Implement auto-merge for low-risk changes (Level 3 autonomy)
4. Add learning and evolution tracking
5. Deploy monitoring dashboards

**Success Criteria**: Amber proactively identifies breaking changes, auto-merges dependency patches

## Alternatives Considered

### Alternative 1: Extend Claude Code Runner

**Approach**: Add Amber persona to existing claude-code-runner, dispatch based on workflow

**Pros**:
- Reuses existing infrastructure
- No new runner component needed
- Simpler deployment

**Cons**:
- Claude Code SDK not designed for autonomous agents
- No built-in state management or checkpointing
- Mixes concerns (interactive sessions vs background automation)
- Hard to evolve independently

**Rejected**: Architectural mismatch, violates modularity principle

### Alternative 2: Custom Python Agent (No LangGraph)

**Approach**: Build Amber as traditional Python service with custom state management

**Pros**:
- Full control over implementation
- No framework dependencies
- Simpler for basic workflows

**Cons**:
- Reimplements state management, checkpointing, retry logic
- No standardized patterns for human-in-the-loop
- Higher maintenance burden
- Harder to reason about control flow

**Rejected**: Reinvents LangGraph features, increases complexity

### Alternative 3: LangChain Agents (Not LangGraph)

**Approach**: Use LangChain's agent framework without LangGraph

**Pros**:
- Simpler than LangGraph for basic agents
- Good tool integration

**Cons**:
- No stateful workflows (DAG-only)
- Limited support for human-in-the-loop
- No built-in checkpointing
- Less control over execution flow

**Rejected**: Insufficient for Amber's complex, multi-mode requirements

## Risks and Mitigations

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| LangGraph learning curve | Delayed delivery | Medium | Allocate 1 week for LangGraph experimentation in Phase 1 |
| PostgreSQL adds operational complexity | Ops burden | Medium | Start with StatefulSet, document backup/restore procedures |
| GitHub API rate limiting | Failed executions | High | Implement exponential backoff, cache responses, request rate limit increase |
| Amber makes incorrect changes | Loss of trust | High | Enforce human approval for all changes initially, track auto-merge success rate |
| State corruption in checkpointer | Session failures | Low | Regular PostgreSQL backups, implement checkpoint validation |
| Anthropic API costs | Budget overrun | Medium | Set per-session token limits, monitor costs via Langfuse, optimize prompts |

## Success Metrics

### Technical Metrics

- **Execution Success Rate**: >95% of sessions complete without errors
- **Checkpoint Restore Rate**: 100% of sessions resume correctly after approval
- **API Failure Recovery**: >90% of transient failures recovered via retry policies
- **Graph Execution Time**: <5 minutes for triage+analysis, <15 minutes for full PR creation

### Product Metrics

- **Maintainer Adoption**: >5 issues labeled `amber:auto-fix` per week
- **Auto-Merge Success**: >95% of auto-merged PRs pass CI and remain unreverted
- **Time-to-Resolution**: Amber PRs merged 50% faster than human-only PRs
- **Issue Triage Accuracy**: >90% of Amber severity/component labels match maintainer judgment

### Trust Metrics

- **Approval Rate**: >80% of Amber PRs approved without changes requested
- **Rollback Rate**: <5% of Amber PRs require revert
- **Feedback Score**: >4.0/5.0 average rating on Amber contributions (thumbs up/down)

## Open Questions

1. **PostgreSQL Sharing**: Should Amber share PostgreSQL instance with other ACP components, or use dedicated instance?
   - **Recommendation**: Dedicated instance initially for isolation, shared instance in Phase 4 if appropriate

2. **GitHub Token Management**: Should Amber use bot account token or per-user tokens?
   - **Recommendation**: Bot account token for autonomous mode, user tokens for on-demand mode

3. **Workflow Repository**: Should Amber persona be in separate repo or live in ACP repo?
   - **Recommendation**: Separate repo (`ambient-code/amber-workflow`) for independent versioning

4. **Multi-Tenant Isolation**: How should Amber handle multiple projects with different GitHub repos?
   - **Recommendation**: One Amber instance per project namespace, credentials from ProjectSettings CR

5. **LangGraph Cloud**: Should we use LangGraph Cloud for hosting, or self-host?
   - **Recommendation**: Self-host on Kubernetes to maintain control, evaluate LangGraph Cloud in Phase 4

## Next Steps

1. **Approval**: Obtain maintainer approval for this ADR
2. **RFC**: Create RFC with detailed implementation plan for Phase 1
3. **Prototype**: Implement Phase 1 POC in feature branch
4. **Demo**: Demo to team, gather feedback
5. **Iterate**: Refine based on feedback, proceed to Phase 2

## Resources

- [LangGraph Documentation](https://docs.langchain.com/oss/python/langgraph/overview)
- [Amber Persona Specification](../../agents/amber.md)
- [ACP Constitution](../../.specify/memory/constitution.md)
- [AgenticSession CRD](../../components/manifests/base/crds/agenticsessions-crd.yaml)
- [Claude Code Runner](../../components/runners/claude-code-runner/)
- [Building LangGraph GitHub Issue Butler](https://www.decodingai.com/p/the-github-issue-ai-butler-on-kubernetes)

## Consequences

### Positive

- **Production-Ready**: LangGraph provides battle-tested state management, checkpointing, error handling
- **Human-in-the-Loop**: Built-in support for approval workflows via checkpoints
- **Observability**: Integration with Langfuse provides tracing, cost tracking, debugging
- **Modularity**: Clear separation of concerns (nodes, tools, state) enables parallel development
- **Extensibility**: Easy to add new modes, nodes, tools as Amber evolves
- **ACP Integration**: Fits cleanly into existing Kubernetes-native architecture

### Negative

- **Framework Dependency**: Tied to LangGraph evolution, breaking changes require adaptation
- **Operational Complexity**: PostgreSQL checkpointer adds operational burden (backups, scaling, monitoring)
- **Learning Curve**: Team must learn LangGraph patterns, StateGraph design
- **Cost**: Anthropic API costs for autonomous execution (mitigated by prompt optimization, rate limiting)

### Neutral

- **Testing Burden**: Requires comprehensive test suite for nodes, graph, integrations (aligned with TDD principle)
- **Documentation**: Extensive docs needed for nodes, tools, mode behaviors (improves maintainability)
- **Migration Path**: Phased rollout enables learning and iteration before full production deployment
