---
title: "Sessions"
---

A **session** is an AI agent execution environment. When you create a session, the platform spins up an isolated container running Claude, connects it to your repositories and integrations, and gives you a real-time chat interface to collaborate with the agent.

## Creating a session

Click **New Session** inside a workspace. The creation dialog lets you configure:

<figure class="screenshot-pair">
  <img class="screenshot-light" src="/platform/images/screenshots/new-session-dialog-light.png" alt="New session creation dialog" />
  <img class="screenshot-dark" src="/platform/images/screenshots/new-session-dialog-dark.png" alt="New session creation dialog" />
</figure>

| Setting | Description | Default |
|---------|------------|---------|
| **Display name** | A label for the session. | Auto-generated |
| **Model** | Which AI model to use. Available models: Claude Sonnet 4.5, Claude Opus 4.5, Claude Haiku 4.5, Gemini 2.5 Flash (generally available); Claude Opus 4.6, Claude Sonnet 4.6, Gemini 2.5 Pro (feature-gated, visible only when enabled for your workspace). | Claude Sonnet 4.5 |
| **Temperature** | Controls response randomness (0 = deterministic, 2 = highly creative). | 0.7 |
| **Max tokens** | Maximum output length per response. The UI enforces a range of 100--8,000, but the platform API accepts other values. | 4,000 |
| **Timeout** | Hard limit on total session duration. The UI enforces a range of 60--1,800 seconds, but the platform API accepts other values. A separate `inactivityTimeout` field controls how long a session can remain idle before the platform auto-stops it; if omitted, it falls back to the project-level setting and then to a 24-hour default. | 300 seconds |

After the session is created, you can attach repositories and select a workflow from the session sidebar. See [Context & Artifacts](../context-and-artifacts/) and [Workflows](../workflows/) for details.

## Session lifecycle

<figure class="screenshot-pair">
  <img class="screenshot-light" src="/platform/images/screenshots/session-list-light.png" alt="Sessions list" />
  <img class="screenshot-dark" src="/platform/images/screenshots/session-list-dark.png" alt="Sessions list" />
</figure>

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

## Human-in-the-loop

Sometimes the agent needs your input before it can continue. When this happens, the agent pauses and presents a **question panel** in the chat. The agent's status changes to `waiting_input` until you respond. After you submit your answer, the agent resumes work automatically.

### Question types

The question panel supports three input styles depending on what the agent needs to know:

- **Free-text** -- an open text field for you to type any response.
- **Single-select** -- a list of radio buttons when the agent offers predefined choices. An **Other** option lets you type a custom answer if none of the choices fit.
- **Multi-select** -- a list of checkboxes when the agent wants you to pick one or more options.

### Multiple questions at once

When the agent has several questions, the panel displays them in a **tabbed interface**. Each tab shows one question, and a counter tracks how many you have answered. After you select an answer the panel auto-advances to the next tab. You can click any tab to revisit a previous answer before submitting.

Once all questions are answered, click **Submit** to send your responses and let the agent continue.

## Session operations

| Operation | What it does |
|-----------|-------------|
| **Stop** | Gracefully halts the agent. You can resume later. |
| **Resume** | Resumes a stopped session from where it left off. |
| **Clone** | Creates a new session with the same configuration and repos -- useful for trying a different approach. Chat history is not copied. |
| **Export** | Downloads session data and offers Markdown or PDF export. If the session is running and Google Drive is connected, you can also save directly to your Drive. |
| **Delete** | Permanently removes the session and its data. |

## Feedback

You can rate any agent response with a thumbs-up or thumbs-down button that appears alongside messages in the chat. Clicking either button opens a feedback modal where you can optionally add a comment explaining what went well or what could be improved. The platform sends your rating to Langfuse for observability and quality tracking, automatically associating it with the session, message, user, active workflow, and trace so that teams can analyze agent performance over time.

## Tips for effective sessions

- **Be specific in your first message.** A clear prompt saves back-and-forth. Instead of "fix the bug," try "the login endpoint in `auth.go` returns 500 when the token is expired -- fix the error handling."
- **Attach the right repos.** The agent can only see code that has been added as context.
- **Pick the right model.** Sonnet 4.5 is fast and cost-effective for most tasks. Opus 4.6 excels at complex multi-step reasoning (if enabled for your workspace).
- **Use workflows for structured tasks.** If there is a workflow that matches your goal (bug fix, triage, spec writing), attach it from the session sidebar to give the agent a proven plan.
- **Review tool calls.** Expanding tool-use blocks lets you verify what the agent actually did before merging its changes.
