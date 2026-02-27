---
title: "Triage Workflow"
---

The Triage workflow helps teams efficiently process their issue backlogs. Given a repository URL, the agent analyzes every open issue, categorizes it, assigns a recommendation, and produces actionable reports with one-click bulk operations.

## When to use

- Your repository has accumulated a backlog of open issues and you need to prioritize them.
- You want a structured, consistent analysis of each issue rather than ad-hoc triage.
- You want to generate bulk operation scripts to close, label, or reassign issues in batch.

## How it works

The Triage workflow is conversational rather than command-driven. Provide a repository URL and the agent handles the rest:

1. **Fetch** -- Reads all open issues from the repository.
2. **Analyze** -- Evaluates each issue for validity, priority, duplicates, and actionability.
3. **Categorize** -- Assigns a recommendation type to every issue.
4. **Report** -- Generates an interactive HTML report with accept/reject checkboxes and a markdown summary table.
5. **Automate** -- Creates a bulk operations script that executes the approved recommendations.

### Getting started

Provide the repository URL in your session prompt:

```
Triage the backlog for https://github.com/owner/repo
```

The agent fetches all open issues, analyzes them, and generates the reports.

## Recommendation types

Each issue receives one of the following recommendations:

| Recommendation | Meaning |
|----------------|---------|
| **CLOSE** | Issue is invalid, obsolete, or a duplicate. |
| **FIX_NOW** | Critical bug or high-value quick win that should be addressed immediately. |
| **BACKLOG** | Valid but not urgent -- move to the backlog for future planning. |
| **NEEDS_INFO** | Blocked waiting for more information from the reporter or stakeholders. |
| **DUPLICATE** | Merge with another issue (the report specifies which one). |
| **CBA_AUTO** | Can be resolved automatically by a codebase agent session. |
| **ASSIGN** | Ready to work and needs an owner assigned. |
| **WONT_FIX** | Out of scope or a deliberate decision not to address. |

## Generated artifacts

The workflow produces three artifacts:

### Markdown report -- `artifacts/triage/triage-report.md`

A table-format summary of every issue with its recommendation. Useful for quick review or pasting into team channels.

### Interactive HTML report -- `artifacts/triage/report.html`

An interactive web page with a checkbox next to each recommendation. Open it in a browser, review each suggestion, check the boxes you agree with, and click "Generate Script" at the bottom. The page produces a ready-to-run script based on your selections.

### Bulk operations script -- `artifacts/triage/bulk-operations.sh`

A shell script that executes the approved recommendations using the GitHub CLI or API. After reviewing the HTML report, download this script and run it locally with your own credentials.

## Post-triage process

1. Open `report.html` in your browser.
2. Review each recommendation and check the box to accept it.
3. Click **Generate Script** at the bottom of the page.
4. Download `bulk-operations.sh` from the file explorer.
5. Review the script, then run it locally with your GitHub token to execute the approved actions.

The script runs on your machine with your credentials, so sensitive tokens never leave your environment.

## Tips

- **Be concise.** The workflow favors one-line recommendations over lengthy analysis. If you need deeper investigation on a specific issue, ask the agent to elaborate.
- **Watch for duplicates.** The agent clusters related issues and identifies duplicates, but you should verify the groupings before merging.
- **Use CBA_AUTO.** Issues tagged as automatable can be handed off to another agent session for resolution, closing the loop without manual work.
- **Iterate.** After executing the bulk script, re-run the triage on the remaining open issues to catch anything that was deferred or needs re-evaluation.
