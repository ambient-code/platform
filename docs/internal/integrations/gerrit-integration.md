# Gerrit Integration for Ambient Code Platform

Ambient Code Platform supports Gerrit code review instances, enabling AgenticSessions to interact with Gerrit changes through an MCP (Model Context Protocol) server. This guide covers connecting Gerrit instances, authentication methods, and using the 21 available code review tools.

## Overview

**What's Supported:**
- Multiple Gerrit instances connected simultaneously (e.g., "openstack", "android")
- HTTP Basic Auth (username + HTTP password)
- Gitcookies-based authentication
- 21 MCP tools for code review operations (browse changes, submit reviews, vote on changes, etc.)
- Credential validation on connect via Gerrit's `/a/accounts/self` endpoint

**Requirements:**
- A Gerrit account with HTTP credentials or gitcookies
- Network connectivity from runner pods to the Gerrit instance
- Ambient Code Platform backend v1.2.0 or higher

---

## Quick Start

### 1. Obtain Gerrit Credentials

Choose one of the two supported authentication methods.

**Option A: HTTP Basic Auth**

1. Log in to your Gerrit instance (e.g., `https://review.opendev.org`)
2. Navigate to **Settings** (gear icon or profile menu)
3. Select **HTTP Credentials** (sometimes labeled "HTTP Password")
4. Click **Generate Password** if you do not already have one
5. Copy and save both your **username** and the generated **HTTP password**

**Option B: Gitcookies**

1. Follow your Gerrit instance's instructions for generating gitcookies (often found at `/new-password` on the instance)
2. Open your local `~/.gitcookies` file
3. Copy the line(s) corresponding to your Gerrit instance
4. You will paste this content when connecting

---

### 2. Connect a Gerrit Instance

Each Gerrit instance you connect requires a unique **instance name**. Instance names must be lowercase alphanumeric with hyphens, between 2 and 63 characters (regex: `^[a-z0-9][a-z0-9-]{0,61}[a-z0-9]$`).

**Via API (HTTP Basic Auth):**
```bash
curl -X POST http://vteam-backend:8080/api/auth/gerrit/connect \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-acp-auth-token>" \
  -d '{
    "instanceName": "openstack",
    "url": "https://review.opendev.org",
    "authMethod": "http_basic",
    "username": "john",
    "httpToken": "abc123"
  }'
```

**Via API (Gitcookies):**
```bash
curl -X POST http://vteam-backend:8080/api/auth/gerrit/connect \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-acp-auth-token>" \
  -d '{
    "instanceName": "android",
    "url": "https://android-review.googlesource.com",
    "authMethod": "git_cookies",
    "gitcookiesContent": ".googlesource.com\tTRUE\t/\tTRUE\t...\to\tgit-user.cookies=..."
  }'
```

**Success Response (200 OK):**
```json
{
  "message": "Gerrit instance 'openstack' connected successfully",
  "instanceName": "openstack",
  "url": "https://review.opendev.org",
  "authMethod": "http_basic"
}
```

**What Happens on Connect:**
1. Credentials are validated by calling `GET /a/accounts/self` on the Gerrit instance
2. If validation passes, credentials are stored in a Kubernetes Secret
3. The instance becomes available for use in AgenticSessions

---

### 3. Test Credentials (Optional)

You can test credentials before storing them. This calls the same validation endpoint but does not persist anything.

```bash
curl -X POST http://vteam-backend:8080/api/auth/gerrit/test \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-acp-auth-token>" \
  -d '{
    "instanceName": "openstack",
    "url": "https://review.opendev.org",
    "authMethod": "http_basic",
    "username": "john",
    "httpToken": "abc123"
  }'
```

Use this to verify your credentials are correct before committing to a connection.

---

### 4. Use Gerrit in an AgenticSession

Once a Gerrit instance is connected, AgenticSessions automatically have access to the Gerrit MCP server and its 21 code review tools.

**What Happens at Runtime:**
1. The runner generates a `gerrit_config.json` file at `/tmp/gerrit-mcp/`
2. The Gerrit MCP server launches in STDIO mode using a Python 3.12 virtual environment
3. The agent can browse changes, submit reviews, vote, and perform other code review operations through the MCP tools

No additional session configuration is needed -- connected Gerrit instances are available automatically.

---

## Managing Connections

### List Connected Instances

```bash
curl -X GET http://vteam-backend:8080/api/auth/gerrit/instances \
  -H "Authorization: Bearer <your-acp-auth-token>"
```

This returns all Gerrit instances currently connected for your user.

### Check Instance Status

```bash
curl -X GET http://vteam-backend:8080/api/auth/gerrit/openstack/status \
  -H "Authorization: Bearer <your-acp-auth-token>"
```

Replace `openstack` with your instance name.

### Disconnect an Instance

```bash
curl -X DELETE http://vteam-backend:8080/api/auth/gerrit/openstack/disconnect \
  -H "Authorization: Bearer <your-acp-auth-token>"
```

This removes:
- Stored credentials for the instance from the Kubernetes Secret
- The instance from your list of connected instances
- Access to that Gerrit instance in future AgenticSessions

Your Gerrit account and credentials on the Gerrit server itself are not affected.

### Update Credentials

To update credentials for an existing instance (e.g., after an HTTP password rotation):

1. Disconnect the instance
2. Reconnect with the new credentials using the same instance name

---

## Multi-Instance Support

You can connect multiple Gerrit instances simultaneously. Each instance is identified by its unique instance name.

**Example: Connecting Two Instances**

```bash
# Connect OpenStack Gerrit
curl -X POST http://vteam-backend:8080/api/auth/gerrit/connect \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-acp-auth-token>" \
  -d '{
    "instanceName": "openstack",
    "url": "https://review.opendev.org",
    "authMethod": "http_basic",
    "username": "john",
    "httpToken": "abc123"
  }'

# Connect Android Gerrit
curl -X POST http://vteam-backend:8080/api/auth/gerrit/connect \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-acp-auth-token>" \
  -d '{
    "instanceName": "android",
    "url": "https://android-review.googlesource.com",
    "authMethod": "git_cookies",
    "gitcookiesContent": ".googlesource.com\tTRUE\t/\tTRUE\t..."
  }'
```

Both instances will be available to the MCP server during AgenticSessions.

---

## Security

### Credential Storage

- Credentials are stored in per-user Kubernetes Secrets named `gerrit-credentials-{userID}`
- Each credential entry is keyed by `instanceName` within the per-user Secret
- Stored in Kubernetes Secrets (encrypted at rest when cluster-level encryption is configured)
- Credentials are never logged in plaintext
- Credentials are not exposed in API responses after storage

### Authentication Methods Compared

| Method | Best For | Credential |
|--------|----------|------------|
| HTTP Basic Auth | Most Gerrit instances | Username + HTTP password from Gerrit Settings |
| Gitcookies | Google-hosted instances (e.g., Googlesource) | Content from `~/.gitcookies` file |

### Credential Rotation

**Recommendation:** Rotate credentials whenever your Gerrit HTTP password changes or your gitcookies expire.

**Process:**
1. Generate new credentials in your Gerrit instance
2. Test the new credentials using the `/api/auth/gerrit/test` endpoint
3. Disconnect the existing instance
4. Reconnect with the new credentials

### Instance Name Rules

Instance names serve as identifiers and must follow these rules:
- Lowercase letters, digits, and hyphens only
- Between 2 and 63 characters
- Must start and end with a letter or digit (not a hyphen)
- Regex: `^[a-z0-9][a-z0-9-]{0,61}[a-z0-9]$`

**Valid examples:** `openstack`, `android`, `my-gerrit-01`

**Invalid examples:** `-bad`, `a` (too short), `UPPER` (uppercase not allowed)

---

## Troubleshooting

### Connection Issues

**Problem:** "Authentication failed" when connecting

**Solutions:**
1. Verify your credentials are correct by testing them manually:
   ```bash
   # For HTTP Basic Auth
   curl -u "john:abc123" https://review.opendev.org/a/accounts/self
   ```
2. Confirm you are using the correct auth method for your instance
3. For HTTP Basic Auth: regenerate the HTTP password in Gerrit Settings > HTTP Credentials
4. For gitcookies: confirm the cookie lines correspond to the correct Gerrit host

**Problem:** "Invalid instance name" error

**Solutions:**
1. Check that the name is 2-63 characters long
2. Use only lowercase letters, digits, and hyphens
3. Do not start or end the name with a hyphen

**Problem:** "Instance already connected" error

**Solutions:**
1. Choose a different instance name, or
2. Disconnect the existing instance first, then reconnect

---

### Runtime Issues

**Problem:** Gerrit MCP server fails to start in AgenticSession

**Solutions:**
1. Check session pod logs for Python version errors (Python 3.12 or higher is required):
   ```bash
   kubectl logs <session-pod> -n <project-namespace>
   ```
2. Verify the instance is still connected:
   ```bash
   curl -X GET http://vteam-backend:8080/api/auth/gerrit/instances \
     -H "Authorization: Bearer <your-acp-auth-token>"
   ```
3. Check that the Gerrit instance is reachable from the runner pod:
   ```bash
   kubectl exec -it <session-pod> -n <project-namespace> -- \
     curl -s -o /dev/null -w "%{http_code}" https://review.opendev.org
   ```

**Problem:** MCP tools return authorization errors during a session

**Solutions:**
1. Your credentials may have expired or been revoked on the Gerrit server
2. Disconnect and reconnect with fresh credentials
3. Verify your Gerrit account still has the necessary permissions

---

### Network Issues

**Problem:** Connection times out

**Solutions:**
1. Verify the Gerrit URL is correct and includes `https://`
2. Check network connectivity from backend pods:
   ```bash
   kubectl exec -it <backend-pod> -n vteam-backend -- \
     curl -s -o /dev/null -w "%{http_code}" https://review.opendev.org
   ```
3. Check firewall rules allow traffic from the Kubernetes cluster to the Gerrit host
4. Verify SSL/TLS certificates are valid

---

## API Reference

### Connect Gerrit Instance

```http
POST /api/auth/gerrit/connect
Content-Type: application/json
Authorization: Bearer <acp-token>
```

**Request Body (HTTP Basic Auth):**
```json
{
  "instanceName": "openstack",
  "url": "https://review.opendev.org",
  "authMethod": "http_basic",
  "username": "john",
  "httpToken": "abc123"
}
```

**Request Body (Gitcookies):**
```json
{
  "instanceName": "android",
  "url": "https://android-review.googlesource.com",
  "authMethod": "git_cookies",
  "gitcookiesContent": ".googlesource.com\tTRUE\t/\tTRUE\t...\to\tgit-user.cookies=..."
}
```

**Response (200 OK):**
```json
{
  "message": "Gerrit instance 'openstack' connected successfully",
  "instanceName": "openstack",
  "url": "https://review.opendev.org",
  "authMethod": "http_basic"
}
```

---

### Test Gerrit Credentials

```http
POST /api/auth/gerrit/test
Content-Type: application/json
Authorization: Bearer <acp-token>
```

Request body is identical to the connect endpoint. Validates credentials without storing them.

---

### List Connected Instances

```http
GET /api/auth/gerrit/instances
Authorization: Bearer <acp-token>
```

**Response (200 OK):**
Returns a list of all connected Gerrit instances for the authenticated user.

---

### Get Instance Status

```http
GET /api/auth/gerrit/:instanceName/status
Authorization: Bearer <acp-token>
```

**Response (200 OK):**
Returns the current status of the specified Gerrit instance.

---

### Disconnect Gerrit Instance

```http
DELETE /api/auth/gerrit/:instanceName/disconnect
Authorization: Bearer <acp-token>
```

**Response (200 OK):**
Removes stored credentials and disconnects the instance.

---

## FAQ

**Q: Can I connect multiple Gerrit instances at the same time?**
A: Yes. Each instance requires a unique instance name. You can connect as many instances as you need (e.g., "openstack", "android", "internal").

**Q: Which authentication method should I use?**
A: Use HTTP Basic Auth for most Gerrit instances. Use gitcookies for Google-hosted instances (such as those on `googlesource.com`) where gitcookies are the standard authentication mechanism.

**Q: What happens if my HTTP password changes on the Gerrit server?**
A: AgenticSessions will fail to authenticate with that instance. Disconnect the instance and reconnect with the new credentials.

**Q: Do I need to configure anything in the AgenticSession to use Gerrit?**
A: No. Connected Gerrit instances are automatically available to the MCP server during sessions. The runner handles configuration and MCP server startup.

**Q: What tools does the Gerrit MCP server provide?**
A: The MCP server provides 21 tools for code review operations, including browsing changes, submitting reviews, voting on changes, and other Gerrit workflow actions.

**Q: Can I use Gerrit alongside GitHub and GitLab integrations?**
A: Yes. The Gerrit integration is independent of GitHub and GitLab integrations. All three can be used simultaneously within the same Ambient Code Platform deployment.

**Q: What Python version does the Gerrit MCP server require?**
A: The Gerrit MCP server requires Python 3.12 or higher. The runner creates a dedicated virtual environment for it automatically.

**Q: How do I know if my credentials are valid before connecting?**
A: Use the `/api/auth/gerrit/test` endpoint. It validates credentials against the Gerrit instance without storing them.

**Q: What is the instance name used for?**
A: The instance name is a user-chosen identifier that distinguishes between multiple Gerrit instances. It is used in API paths (e.g., `/api/auth/gerrit/openstack/status`) and as part of the credential storage key.

**Q: Can two users connect the same Gerrit instance with the same instance name?**
A: Yes. Instance names are scoped per user. Two different users can both have an instance named "openstack" without conflict, as credentials are stored with a compound key of `instanceName.userID`.

---

## Support and Resources

**Troubleshooting:**
- Check backend logs: `kubectl logs -l app=vteam-backend -n vteam-backend`
- Check session logs: `kubectl logs <session-pod> -n <project-namespace>`
- Verify Gerrit instance availability by accessing it directly in a browser

**Gerrit Resources:**
- [Gerrit REST API Documentation](https://gerrit-review.googlesource.com/Documentation/rest-api.html)
- [Gerrit HTTP Credentials](https://gerrit-review.googlesource.com/Documentation/user-upload.html#http)
- [Gitcookies Authentication](https://gerrit-review.googlesource.com/Documentation/user-upload.html#cookies)
