---
title: "Sessions"
---

A **session** is an AI agent execution environment. When you create a session, the platform spins up an isolated container running Claude, connects it to your repositories and integrations, and gives you a real-time chat interface to collaborate with the agent.

## Creating a session

Click **New Session** inside a workspace. The creation dialog lets you configure:

| Setting | Description | Default |
|---------|------------|---------|
| **Display name** | A label for the session. | Auto-generated |
| **Model** | Which Claude model to use: Claude Sonnet 4.5, Claude Opus 4.6, Claude Opus 4.5, or Claude Haiku 4.5. | Claude Sonnet 4.5 |
| **Temperature** | Controls response randomness (0 = deterministic, 2 = highly creative). | 0.7 |
| **Max tokens** | Maximum output length per response (100 -- 8,000). | 4,000 |
| **Timeout** | Hard limit on total session duration (60 -- 1,800 seconds). | 300 seconds |

After the session is created, you can attach repositories and select a workflow from the session sidebar. See [Context & Artifacts](./context-and-artifacts) and [Workflows](./workflows) for details.

## Session lifecycle

Every session moves through a series of phases:

```
Pending --> Creating --> Running --> Completed
                          |
                          +--> Stopping --> Stopped
                          |
                          +--> Failed
```

| Phase | What is happening |
|-------|------------------|
| **Pending** | The session request has been accepted and is waiting to be scheduled. |
| **Creating** | The platform is provisioning the container, cloning repositories, and injecting secrets. |
| **Running** | The agent is active and ready to accept messages. |
| **Stopping** | A stop was requested; the agent is finishing its current turn and saving state. |
| **Stopped** | The session was stopped manually. It can be continued later. |
| **Completed** | The agent finished its work and exited on its own. |
| **Failed** | Something went wrong -- check the session events for details. |

## The chat interface

Once a session is **Running**, the chat panel is your primary way to interact with the agent.

### Agent status indicators

At any moment the agent is in one of three states:

- `working` -- actively processing your request, calling tools, or writing code.
- `idle` -- finished its current turn and waiting for your next message.
- `waiting_input` -- the agent has asked a clarifying question and is blocked until you reply.

### What you see in the chat

- **Messages** -- your prompts and the agent's responses.
- **Tool use blocks** -- expandable panels showing each tool the agent called (file reads, edits, shell commands, searches) along with their results.
- **Thinking blocks** -- the agent's internal reasoning, visible for transparency.

### Interrupting the agent

If the agent is heading in the wrong direction while it is still **Working**, you can send a new message at any time. The agent will read your message after its current tool call finishes and adjust course.

## Session operations

| Operation | What it does |
|-----------|-------------|
| **Stop** | Gracefully halts the agent. You can resume later. |
| **Resume** | Resumes a stopped session from where it left off. |
| **Clone** | Creates a new session with the same configuration and repos -- useful for trying a different approach. Chat history is not copied. |
| **Export** | Exports the chat history in one of three formats: Markdown, PDF, or Google Drive. |
| **Delete** | Permanently removes the session and its data. |

## Tips for effective sessions

- **Be specific in your first message.** A clear prompt saves back-and-forth. Instead of "fix the bug," try "the login endpoint in `auth.go` returns 500 when the token is expired -- fix the error handling."
- **Attach the right repos.** The agent can only see code that has been added as context.
- **Pick the right model.** Sonnet 4.5 is fast and cost-effective for most tasks. Opus 4.6 excels at complex multi-step reasoning.
- **Use workflows for structured tasks.** If there is a workflow that matches your goal (bug fix, triage, spec writing), attach it from the session sidebar to give the agent a proven plan.
- **Review tool calls.** Expanding tool-use blocks lets you verify what the agent actually did before merging its changes.
