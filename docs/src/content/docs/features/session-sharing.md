---
title: Session Sharing & Credentials
description: How credentials work in shared sessions
---

## Overview

When multiple users collaborate in a shared session, **each message uses the sender's credentials**, not the session owner's. This ensures that all actions are correctly attributed and that each user's access permissions are respected.

## Credential Behavior

### Interactive Sessions

| Scenario | Session Owner | Message Sender | Credentials Used |
|----------|---------------|----------------|------------------|
| Single-user session | User A | User A | User A |
| Shared session | User A | User B | User B |

**Example:**

- User A creates a session and adds User B as an editor
- Both User A and User B have configured their GitHub integrations
- User A sends "Create a PR" -- the PR is created using User A's GitHub token
- User B sends "Create a PR" -- the PR is created using User B's GitHub token

### Automated Sessions

**API Keys:** Always use the **creator's credentials** (the user who created the API key).

- GitHub commits show the creator's username
- The creator is responsible for all actions performed via their API key

**Scheduled Sessions:** Always use the **creator's credentials** (the user who scheduled the session).

- The session runs as the creating user even when they are offline
- The creator is accountable for all scheduled session behavior

## Requirements

Each editor in a shared session must configure their own integrations before using features that require credentials:

1. Go to **Settings > Integrations**
2. Connect the required services (GitHub, Jira, Google, GitLab)

If an editor sends a message that requires credentials they have not configured, Claude will report an error explaining which integration is missing.

## Security

- All actions are attributed to the actual message sender
- Audit logs show the correct user for each operation
- Credentials are never shared between users
- Each user's tokens are scoped to their own permissions
