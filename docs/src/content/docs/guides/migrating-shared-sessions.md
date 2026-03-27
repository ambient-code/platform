---
title: Migrating to Per-Message Credentials
description: How to update existing shared sessions for per-message credential behavior
---

## What Changed

**Before:** Shared sessions used the session owner's credentials for all messages, regardless of who sent them.

**After:** Each message uses the sender's own credentials. Actions are attributed to the user who initiated them.

## Impact

If you have shared sessions created before this update:
- Messages sent by editors will now use **their own credentials** instead of the session owner's
- If an editor has not configured the required integrations, operations that need those credentials will fail

## Action Required

For each shared session with multiple editors:

### 1. All editors configure integrations

Each user who participates in the session must connect their own accounts:

- Go to **Settings > Integrations**
- Connect: GitHub, Jira, Google, GitLab (as needed for the session's work)

### 2. Verify credentials

- Send a test message from each editor that triggers an integration (e.g., "list my open PRs")
- Confirm that the operation uses the correct identity
- For GitHub operations, verify the commit author matches the message sender

### 3. Update API keys (if applicable)

- API keys continue using the creator's credentials -- no changes needed
- If you want operations to run under a different identity, create a new API key with that user

## Troubleshooting

**"GitHub credentials not configured"**
The message sender has not connected their GitHub account. That user must go to **Settings > Integrations** and connect GitHub.

**"Jira credentials not configured"**
Same as above, but for Jira. The sender must configure their Jira integration.

**API automation stopped working**
Check that the API key creator's credentials are still valid. API keys inherit the creator's identity, so if their token expired or was revoked, operations will fail.

## Rollback

If you encounter issues with per-message credentials, you can preserve the previous session owner behavior by setting the `KEEP_CREDENTIALS_PERSISTENT` environment variable on the session:

```bash
KEEP_CREDENTIALS_PERSISTENT=true
```

This prevents credential cleanup between turns, so the session owner's credentials remain active for all messages.
