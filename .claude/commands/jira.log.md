---
description: Log a new Jira issue to RHOAIENG with Team (Ambient team) and Component (Agentic) pre-filled.
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Goal

Create a new Jira Story in the RHOAIENG project with the correct Team and Component pre-filled for the Ambient team.

## Execution Steps

### 1. Parse User Input

Extract the following from `$ARGUMENTS`:

- **Summary** (required): The title/summary of the issue
- **Description** (optional): Detailed description of the work
- **Issue Type** (optional): Defaults to `Story`, but can be `Bug` or `Task`. Tasks are tech debt related and not user facing
- **Priority** (optional): Defaults to `Normal`

If the user provides a simple sentence, use it as the summary. If they provide multiple lines, use the first line as summary and the rest as description.

### 2. Gather Cold-Start Context

**IMPORTANT**: To make this Jira actionable by an agent, gather the following context. Ask the user for any missing critical info:

**Required for Stories:**
- What is the user-facing goal? (As a [user], I want [X], so that [Y])
- What are the acceptance criteria? (How do we know it's done)
- Which repo/codebase? (e.g., `vTeam`, `ambient-cli`)

**Required for Bugs:**
- Steps to reproduce
- Expected vs actual behaviour
- Environment/browser info if relevant

**Helpful for all types:**
- Relevant file paths or components (e.g., `components/frontend/src/...`)
- Related issues/PRs/design docs
- Screenshots or mockups (as links)
- Constraints or out-of-scope items
- Testing requirements

### 3. Build Structured Description

Format the description using this **agent-friendly template**:

```markdown
## Overview
[One paragraph summary of what needs to be done and why]

## User Story (for Stories)
As a [type of user], I want [goal], so that [benefit].

## Acceptance Criteria
- [ ] [Criterion 1]
- [ ] [Criterion 2]
- [ ] [Criterion 3]

## Technical Context
**Repo**: [repo name or URL]
**Relevant Paths**:
- `path/to/relevant/file.ts`
- `path/to/another/area/`

## Related Links
- Design: [link if any]
- Related Issues: [RHOAIENG-XXXX]
- PR: [link if any]

## Constraints
- [What NOT to do]
- [Boundaries to respect]

## Testing Requirements
- [ ] Unit tests for [X]
- [ ] E2E test for [Y]

## Bug Details (for Bugs only)
**Steps to Reproduce**:
1. Step 1
2. Step 2

**Expected**: [what should happen]
**Actual**: [what actually happens]
**Environment**: [browser/OS if relevant]
```

### 4. Confirm Details

Before creating the issue, confirm with the user:

```
📋 About to create RHOAIENG Jira:

**Summary**: [extracted summary]
**Type**: Story
**Component**: Agentic
**Team**: Ambient team

**Description Preview**:
[Show first 500 chars of formatted description]

This description is structured for agent cold-start. Shall I create this issue? (yes/no/edit)
```

### 5. Create the Jira Issue

Use the `mcp_mcp-atlassian_jira_create_issue` tool with:

```json
{
  "project_key": "RHOAIENG",
  "summary": "[user provided summary]",
  "issue_type": "Story",
  "description": "[structured description from template]",
  "components": "Agentic"
}
```

Then **update the issue** to set the Team field (must be done as a separate update call):

```json
{
  "issue_key": "[CREATED_ISSUE_KEY]",
  "fields": {
    "customfield_12313240": "6290"
  }
}
```

### 6. Report Success

After creation, report:

```
✅ Created: [ISSUE_KEY]
🔗 Link: https://issues.redhat.com/browse/[ISSUE_KEY]

Summary: [summary]
Component: Agentic
Team: Ambient team

📋 Agent Cold-Start Ready: Yes
```

## Examples

### Quick Story (will prompt for more context)

```
/jira.log Add dark mode toggle to session viewer
```

The command will then ask you for acceptance criteria, relevant files, etc.

### Detailed Story (agent-ready)

```
/jira.log Add dark mode toggle to session viewer

As a user, I want to toggle dark mode in the session viewer, so that I can reduce eye strain during long sessions.

Acceptance:
- Toggle persists across sessions (localStorage)
- Respects system preference by default
- Smooth transition animation

Repo: vTeam
Files: components/frontend/src/components/session-viewer/
Related: RHOAIENG-38000 (design system tokens)

Constraints:
- Use existing Shadcn theme tokens, don't create new colours
- Must work with existing syntax highlighting

Tests:
- Unit test for toggle logic
- E2E test for persistence
```

### Bug Report

```
/jira.log [Bug] Session list doesn't refresh after deletion

Steps:
1. Create a session
2. Delete the session via UI
3. Observe the list

Expected: Session disappears from list
Actual: Session remains until page refresh

Repo: vTeam  
Files: components/frontend/src/components/session-list/
Browser: Chrome 120, Firefox 121

Fix should invalidate the React Query cache after mutation.
```

### Tech Debt Task

```
/jira.log [Task] Migrate session queries to use React Query v5 patterns

Current queries use deprecated `onSuccess` callbacks.
Need to migrate to the new `select` and mutation patterns.

Repo: vTeam
Files: 
- components/frontend/src/services/queries/sessions.ts
- components/frontend/src/hooks/

Constraints:
- Don't change API contracts
- Maintain backwards compatibility with existing components

Tests:
- Existing tests should pass
- Add test for cache invalidation edge case
```

## Field Reference

| Field | Value | Notes |
|-------|-------|-------|
| Project | RHOAIENG | Red Hat OpenShift AI Engineering |
| Component | Agentic | Pre-filled |
| Team | Ambient team | Custom field `customfield_12313240` = `6290` |
| Issue Type | Story | Default, can override with [Bug], [Task] |
| Priority | Normal | Default |

## Agent Cold-Start Checklist

For a Jira to be immediately actionable by an agent, ensure:

| Element | Why It Matters |
|---------|----------------|
| **User Story** | Agent understands the "who" and "why" |
| **Acceptance Criteria** | Clear definition of done, testable outcomes |
| **Repo + File Paths** | Agent knows where to look/edit |
| **Related Links** | Context from design docs, related PRs |
| **Constraints** | Prevents agent from over-engineering or going off-piste |
| **Testing Requirements** | Agent knows what coverage is expected |
| **Bug Repro Steps** | Agent can verify the fix works |

### What Makes a Good vs Bad Jira for Agents

**❌ Bad (vague, agent will struggle):**
> "Fix the login bug"

**✅ Good (agent can start immediately):**
> "Fix login redirect loop on Safari"
> 
> **Steps**: 1. Open Safari 2. Click Login 3. Observe infinite redirect
> **Expected**: Redirect to dashboard
> **Actual**: Loops back to login
> **Repo**: vTeam
> **Files**: `components/frontend/src/app/auth/callback/`
> **Constraint**: Don't break Chrome/Firefox flows
> **Test**: Add E2E test for Safari user-agent

## Context

$ARGUMENTS
