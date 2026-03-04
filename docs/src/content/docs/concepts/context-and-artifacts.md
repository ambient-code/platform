---
title: "Context & Artifacts"
---

In the Ambient Code Platform, **context** is what you give the agent and **artifacts** are what it gives back.

## Context -- the input

Context is the information the agent has access to when it starts working. The primary form of context is source code from Git repositories, but you can also provide files and instructions through the chat.

### Adding repositories

After creating a session, you can attach one or more Git repositories from the session sidebar:

1. Open the session and click **Add Repository** in the sidebar.
2. Enter the Git URL (HTTPS or SSH).
3. Select the **branch** to clone. The agent will work on this branch.
4. Repeat for additional repositories if your task spans multiple codebases.

The agent clones each repository into its workspace and has full read/write access to the files.

### Branch management

- **Branch selection** -- choose any branch at the time you add a repository.
- **Branch switching** -- the agent can create and switch branches during a session as part of its normal Git workflow.
- **Multiple repos** -- each repository tracks its own branch independently.

### Auto-push

When a session completes or is stopped, the platform can **automatically push** the agent's commits back to the remote repository. This behavior is configurable per repository when you add it to a session.

Auto-push is useful for fully automated workflows where you want changes delivered without manual intervention. For review-first workflows, leave auto-push off and inspect the agent's changes in the artifact browser before pushing manually.

## Artifacts -- the output

Artifacts are files the agent creates, modifies, or generates during a session. They live in the session's workspace and persist after the session ends.

### File browser

Every session has a **file browser** accessible from the session sidebar. It lets you:

- Browse the full directory tree of the session workspace.
- View file contents, including diffs of what the agent changed.
- See which files were added, modified, or deleted.

### Downloading artifacts

You can download individual files or the entire workspace from the file browser. This is useful for:

- Reviewing generated code before merging.
- Saving reports, specs, or documentation the agent produced.
- Archiving session output for compliance or audit purposes.

## Putting it together

A typical workflow looks like this:

1. **Provide context** -- attach one or more repositories, select the right branches, and write a clear prompt.
2. **The agent works** -- it reads your code, makes changes, runs tools, and writes new files.
3. **Review artifacts** -- browse the file tree, inspect diffs, and download what you need.
4. **Push or merge** -- if auto-push is enabled the changes are already on the remote; otherwise, push manually after review.

### Tips

- **Attach only the repos the agent needs.** Extra repositories add clone time and noise.
- **Use the right branch.** Point the agent at a feature branch if you do not want changes landing directly on `main`.
- **Check diffs before pushing.** The file browser shows exactly what changed -- use it.
- **Combine with workflows.** Workflows like Bugfix or Spec-kit structure the agent's output so artifacts are consistent and easy to review.
