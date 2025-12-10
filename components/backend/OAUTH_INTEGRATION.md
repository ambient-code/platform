# OAuth Integration Guide

This document describes the generic OAuth2 integration in the vTeam backend.

## Overview

The backend now supports generic OAuth2 authentication flows through a standardized `/oauth2callback` endpoint. This enables integration with various OAuth providers including Google, GitHub, and others.

## Supported Providers

### Google OAuth
- **Provider ID**: `google`
- **Environment Variables**:
  - `GOOGLE_OAUTH_CLIENT_ID`: Your Google OAuth client ID
  - `GOOGLE_OAUTH_CLIENT_SECRET`: Your Google OAuth client secret
- **Scopes**: Drive, Drive File, Drive Readonly, User Info Email

### GitHub OAuth
- **Provider ID**: `github`
- **Environment Variables**:
  - `GITHUB_CLIENT_ID`: Your GitHub OAuth client ID
  - `GITHUB_CLIENT_SECRET`: Your GitHub OAuth client secret
- **Scopes**: repo, user

## Endpoints

### 1. OAuth Callback Endpoint
**Route**: `GET /oauth2callback`

This is the redirect URI you should configure in your OAuth provider settings.

**Query Parameters**:
- `code` (required): Authorization code from OAuth provider
- `state` (optional): State parameter for security/tracking
- `provider` (optional): Provider identifier (defaults to "google")
- `error` (optional): Error code if authorization failed
- `error_description` (optional): Human-readable error description

**Response**:
- Success: HTML page with success message and auto-close script
- Error: HTML page with error details

**Example Redirect URI**:
```
http://localhost:8000/oauth2callback
```

### 2. OAuth Status Endpoint
**Route**: `GET /oauth2callback/status?state={state}`

Check the status of an OAuth flow by state parameter.

**Query Parameters**:
- `state` (required): The state parameter from the OAuth flow

**Response**:
```json
{
  "provider": "google",
  "userId": "user@example.com",
  "receivedAt": "2025-12-10T12:00:00Z",
  "consumed": false,
  "hasToken": true
}
```

Or if there was an error:
```json
{
  "provider": "google",
  "receivedAt": "2025-12-10T12:00:00Z",
  "consumed": false,
  "hasToken": false,
  "error": "access_denied",
  "errorDescription": "User denied access"
}
```

## Configuration

### For MCP Google Workspace Integration

Add to your `.mcp.json`:

```json
{
  "mcpServers": {
    "google_workspaceX": {
      "type": "stdio",
      "command": "uvx",
      "args": ["workspace-mcp", "--tools", "drive"],
      "env": {
        "GOOGLE_OAUTH_CLIENT_ID": "your-client-id",
        "GOOGLE_OAUTH_CLIENT_SECRET": "your-client-secret",
        "OAUTHLIB_INSECURE_TRANSPORT": "1"
      }
    }
  }
}
```

### Environment Variables for Backend

Add to your backend environment (`.env.local` or deployment config):

```bash
# Google OAuth
GOOGLE_OAUTH_CLIENT_ID=your-google-client-id
GOOGLE_OAUTH_CLIENT_SECRET=your-google-client-secret

# GitHub OAuth (if not already configured)
GITHUB_CLIENT_ID=your-github-client-id
GITHUB_CLIENT_SECRET=your-github-client-secret
GITHUB_STATE_SECRET=your-state-secret
```

## OAuth Flow

1. **User initiates OAuth**: User clicks "Connect" in your application
2. **Redirect to provider**: User is redirected to OAuth provider (Google, GitHub, etc.)
3. **User authorizes**: User grants permissions
4. **Provider redirects back**: Provider redirects to `/oauth2callback?code=xxx&state=yyy`
5. **Backend exchanges code**: Backend exchanges authorization code for access token
6. **Backend stores data**: OAuth callback data is stored in Secret `oauth-callbacks`
7. **User sees success**: User sees success page and can close the window
8. **Application retrieves token**: MCP or application retrieves token data using state parameter

## Data Storage

OAuth callback data is stored in a Kubernetes Secret named `oauth-callbacks` in the backend namespace for enhanced security. Each entry is keyed by the `state` parameter and contains:

- `provider`: OAuth provider name
- `userId`: User ID (if available from session)
- `code`: Authorization code
- `state`: State parameter
- `accessToken`: Access token (after exchange)
- `refreshToken`: Refresh token (if provided)
- `expiresIn`: Token expiration time in seconds
- `receivedAt`: Timestamp when callback was received
- `consumed`: Whether the token has been consumed
- `error`: Error code (if authorization failed)

## Security Considerations

1. **HTTPS Required**: In production, always use HTTPS for OAuth callbacks
2. **State Validation**: Always validate the state parameter to prevent CSRF attacks
3. **Token Storage**: Tokens are stored in Kubernetes Secrets for secure storage
4. **Token Exposure**: The status endpoint does not return actual tokens, only metadata
5. **RBAC**: Ensure appropriate RBAC policies are in place to restrict access to the `oauth-callbacks` Secret
6. **Cleanup**: Consider implementing token cleanup/expiration logic to prevent Secret bloat

## Adding New Providers

To add support for a new OAuth provider:

1. Add a new case in `getOAuthProvider()` function in `handlers/oauth.go`
2. Configure the provider's token URL and scopes
3. Add environment variables for client ID and secret
4. Update this documentation

Example:

```go
case "gitlab":
    clientID := os.Getenv("GITLAB_OAUTH_CLIENT_ID")
    clientSecret := os.Getenv("GITLAB_OAUTH_CLIENT_SECRET")
    if clientID == "" || clientSecret == "" {
        return nil, fmt.Errorf("GitLab OAuth not configured")
    }
    return &OAuthProvider{
        Name:         "gitlab",
        ClientID:     clientID,
        ClientSecret: clientSecret,
        TokenURL:     "https://gitlab.com/oauth/token",
        Scopes:       []string{"api", "read_user"},
    }, nil
```

## Testing

### Manual Testing

1. Start the backend server with OAuth credentials configured
2. Navigate to your OAuth provider's authorization URL with redirect_uri=http://localhost:8000/oauth2callback
3. Authorize the application
4. You should be redirected back to the callback endpoint
5. Check the status endpoint to verify token storage

### Using with MCP

The MCP server will automatically handle the OAuth flow when you use MCP-enabled tools that require authentication.

## Troubleshooting

### "OAuth not configured" error
- Ensure environment variables are set correctly
- Check that the provider name matches (case-insensitive)

### "Failed to exchange authorization code"
- Verify client ID and secret are correct
- Check that redirect URI matches exactly (including protocol, port, path)
- Ensure the authorization code hasn't expired (usually valid for ~10 minutes)

### "Callback not found" when checking status
- Verify the state parameter matches exactly
- Check that the callback was successfully stored (check backend logs)
- Secret may have been cleared or backend restarted
- Verify you have RBAC permissions to read the `oauth-callbacks` Secret

## Related Files

- `components/backend/handlers/oauth.go` - OAuth handler implementation
- `components/backend/routes.go` - Route registration
- `.mcp.json` - MCP server configuration (local development)
