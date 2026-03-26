---
title: "Scheduled Sessions"
---

A **scheduled session** is a recurring AI agent execution that runs automatically on a cron schedule. Each scheduled session is backed by a Kubernetes CronJob that creates a new agentic session at every scheduled interval, using a preconfigured session template. You can use scheduled sessions to automate repetitive tasks such as nightly code reviews, dependency scans, or periodic issue triage -- without manually creating a session each time.

## Create a scheduled session

Open a workspace and navigate to the **Scheduled Sessions** tab. Click **New Scheduled Session** to open the creation dialog.

| Setting | Description | Default |
|---------|------------|---------|
| **Name** | A display name for the schedule (up to 50 characters). | Auto-generated |
| **Schedule** | How often the session runs. Choose a preset or enter a custom cron expression. | Every hour (`0 * * * *`) |
| **Initial prompt** | The prompt sent to the agent when each session starts. This defines the task the agent performs on every run. | *(required)* |
| **Runner type** | The runtime environment for the agent. | Claude Agent SDK |
| **Model** | Which AI model the agent uses. | Claude Sonnet 4.5 |

### Schedule presets

The creation dialog offers common presets so you do not need to write cron expressions manually:

| Preset | Cron expression |
|--------|----------------|
| Every hour | `0 * * * *` |
| Daily at 9:00 AM | `0 9 * * *` |
| Every weekday at 9:00 AM | `0 9 * * 1-5` |
| Weekly on Monday | `0 9 * * 1` |
| Custom | Enter any valid 5-field cron expression |

When you select or enter a schedule, the dialog shows a human-readable description and previews the next three run times.

### Session template

Each scheduled session stores a **session template** that defines the configuration for every session it creates. The template includes the initial prompt, model, runner type, temperature (0.7), max tokens (4,000), and timeout (300 seconds). To change these values after creation, update the scheduled session.

## Manage schedules

You can manage scheduled sessions from the **Scheduled Sessions** tab or from the actions menu on each row.

| Operation | What it does |
|-----------|-------------|
| **Suspend** | Pauses the schedule. No new sessions are created until you resume. Existing running sessions are not affected. |
| **Resume** | Reactivates a suspended schedule. The next session is created at the next scheduled time. |
| **Trigger now** | Immediately creates a one-off session from the schedule's template, regardless of the cron timing. The triggered session is linked to the schedule as a child job. |
| **Update** | Changes the schedule, display name, or session template. You can update any combination of fields. |
| **Delete** | Permanently removes the schedule and its associated Kubernetes CronJob. Child jobs are cleaned up in the background. |

### Schedule status

Each scheduled session displays one of two statuses:

- **Active** -- the schedule is running and creates sessions at the configured interval.
- **Suspended** -- the schedule is paused and does not create new sessions.

## View run history

Click a scheduled session name to view its detail page, which lists all sessions created by that schedule. Each run is a standard agentic session that follows the same lifecycle as manually created sessions (Pending, Creating, Running, Completed, Failed, or Stopped). The scheduled sessions list also shows the **Last Run** column, indicating when the most recent session was created.

The platform retains history for the last 5 successful runs and 3 failed runs per schedule.

## Concurrency and reliability

Scheduled sessions use a **forbid-concurrent** policy: if a session from a previous run is still active when the next scheduled time arrives, the new run is skipped. This prevents overlapping agent executions on the same task.

If the platform is unavailable at the scheduled time, the CronJob has a 5-minute starting deadline. Runs that miss this window are skipped entirely.

## Use cases

Scheduled sessions work well for tasks that benefit from regular, unattended execution:

- **Nightly code health reports** -- run a daily scan that reviews code quality, flags potential issues, and generates a summary.
- **Recurring dependency scans** -- check for outdated or vulnerable dependencies on a weekly cadence.
- **Periodic issue triage** -- have the agent review and categorize new issues every morning before the team starts work.
- **Automated documentation updates** -- regenerate or validate documentation against code changes on a schedule.
- **Regression checks** -- run a suite of verification prompts after each nightly build to catch regressions early.
