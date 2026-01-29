# Google Drive Integration

Connect your Google Drive to agentic sessions so Claude can read and write files in your Drive.

## What It Does

Once connected, Claude can:
- List files and folders in your Google Drive
- Read file contents
- Create new files
- Update existing files
- Search your Drive

## How to Use

### 1. Connect Google Drive to a Session

1. Open or create an agentic session
2. In the left sidebar, expand **"MCP Integrations"**
3. Click **"Connect"** on the Google Drive card
4. Authorize in the popup window
5. Close the popup when you see "Authorization Successful"

### 2. Use Drive in Your Prompts

Once connected, you can ask Claude things like:

- "List the files in my Google Drive"
- "Read the contents of 'meeting-notes.txt' from my Drive"
- "Create a new document in my Drive called 'project-summary.md' with this content..."
- "Find all PDFs in my Drive from last month"

### 3. Disconnect (Optional)

Credentials are automatically removed when the session ends. To disconnect earlier, click "Disconnect" in the MCP Integrations accordion.

## First-Time Setup (Admin)

Administrators need to configure Google OAuth credentials once:

1. Create a Google Cloud project and OAuth 2.0 credentials
2. Set authorized redirect URI to: `https://your-vteam-backend/oauth2callback`
3. Create a Kubernetes Secret in the operator namespace:

```bash
kubectl create secret generic google-workflow-app-secret \
  --from-literal=GOOGLE_OAUTH_CLIENT_ID='your-client-id' \
  --from-literal=GOOGLE_OAUTH_CLIENT_SECRET='your-client-secret'
```

## Security & Privacy

- **Session-scoped**: Credentials only exist for the current session
- **Automatic cleanup**: Credentials deleted when session ends
- **No sharing**: Your credentials never accessible to other users or sessions
- **You control access**: You must explicitly connect for each session

## Troubleshooting

**"Connect" button doesn't work**
- Check that popup blockers aren't blocking the OAuth window

**Claude says it can't access Drive**
- Verify you see "Connected" status in MCP Integrations accordion
- Try disconnecting and reconnecting
- Check the session logs for errors

**"Invalid scopes" error**
- You may need to re-authorize with updated permissions
- Click "Disconnect" then "Connect" again

## Questions?

See the [workspace-mcp documentation](https://github.com/taylorwilsdon/google_workspace_mcp) for details about what Drive operations are supported.
