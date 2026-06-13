---
title: GitHub Action
---

import { Badge } from '@astrojs/starlight/components';

<Badge text="Beta" variant="caution" />

The [`ambient-action`](https://github.com/ambient-code/ambient-action) GitHub Action creates Ambient Code Platform sessions directly from GitHub workflows. Use it to automate bug fixes on new issues, run code analysis on pull requests, or trigger any agent workflow from CI/CD.

## Modes

- **Fire-and-forget** -- Create a session and move on. The workflow does not wait for the session to finish.
- **Wait-for-completion** -- Create a session and poll until it completes (or times out). Useful when subsequent steps depend on the agent's output.
- **Send to existing session** -- Send a message to a running session instead of creating a new one (set `session-name`).

## Inputs

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `api-url` | Yes | -- | Ambient Code Platform backend API URL |
| `api-token` | Yes | -- | Bot user bearer token for authentication (store as a GitHub secret) |
| `project` | Yes | -- | Target workspace/project name |
| `prompt` | Yes | -- | Initial prompt for the session, or message to send to an existing session |
| `session-name` | No | -- | Existing session name to send a message to (skips session creation) |
| `display-name` | No | -- | Human-readable session display name |
| `repos` | No | -- | JSON array of repo objects (`[{"url":"...","branch":"...","autoPush":true}]`) |
| `labels` | No | -- | JSON object of labels for the session |
| `environment-variables` | No | -- | JSON object of environment variables to inject into the runner |
| `workflow` | No | -- | JSON workflow object (e.g., `{"gitUrl":"https://...","branch":"main","path":"workflows/my-wf"}`) |
| `model` | No | -- | Model override (e.g., `claude-sonnet-4-20250514`) |
| `timeout` | No | `0` | Session inactivity timeout in seconds -- auto-stops after this duration of inactivity (`0` means no timeout) |
| `stop-on-run-finished` | No | `false` | Stop the session automatically when the agent finishes its run |
| `wait` | No | `false` | Wait for session completion |
| `poll-interval` | No | `15` | Seconds between status checks (only when `wait: true`) |
| `poll-timeout` | No | `60` | Maximum minutes to poll before giving up (only when `wait: true`) |
| `no-verify-ssl` | No | `false` | Disable SSL certificate verification (for self-signed certs) |

## Outputs

| Output | Description |
|--------|-------------|
| `session-name` | Name of the created session |
| `session-uid` | UID of the created session |
| `session-url` | URL to the session in the Ambient UI |
| `session-phase` | Final session phase (only set when `wait: true`) |
| `session-result` | Session result text (only set when `wait: true`) |

## Quick start

Add the action to any GitHub Actions workflow:

```yaml
name: Run ACP Session
on:
  issues:
    types: [opened]

jobs:
  triage:
    runs-on: ubuntu-latest
    steps:
      - uses: ambient-code/ambient-action@v0.0.5
        with:
          api-url: ${{ secrets.ACP_URL }}
          api-token: ${{ secrets.ACP_TOKEN }}
          project: my-team
          prompt: |
            Triage this issue and suggest a severity label:
            ${{ github.event.issue.title }}
            ${{ github.event.issue.body }}
```

## Wait for completion

Set `wait: true` to block the workflow until the session finishes:

```yaml
- uses: ambient-code/ambient-action@v0.0.5
  with:
    api-url: ${{ secrets.ACP_URL }}
    api-token: ${{ secrets.ACP_TOKEN }}
    project: my-team
    prompt: "Analyze the codebase for security vulnerabilities and report findings."
    wait: true
    poll-timeout: 30
```

## Send a message to an existing session

Use `session-name` to send a follow-up prompt to a running session instead of creating a new one:

```yaml
- uses: ambient-code/ambient-action@v0.0.5
  with:
    api-url: ${{ secrets.ACP_URL }}
    api-token: ${{ secrets.ACP_TOKEN }}
    project: my-team
    session-name: my-existing-session
    prompt: "Run the test suite again and report the results."
```

## Multi-repo sessions

Pass a JSON array to clone multiple repositories into the session:

```yaml
- uses: ambient-code/ambient-action@v0.0.5
  id: session
  with:
    api-url: ${{ secrets.ACP_URL }}
    api-token: ${{ secrets.ACP_TOKEN }}
    project: platform-team
    prompt: "Refactor shared types to use the new schema"
    repos: |
      [
        {"url": "https://github.com/org/frontend.git", "branch": "main", "autoPush": true},
        {"url": "https://github.com/org/backend.git", "branch": "main", "autoPush": true}
      ]
    workflow: |
      {"gitUrl": "https://github.com/org/workflows.git", "branch": "main", "path": "workflows/cross-repo-refactor"}
    wait: true
    poll-timeout: 45

- run: echo "Session ${{ steps.session.outputs.session-name }} finished"
```
