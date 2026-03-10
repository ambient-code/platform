/**
 * Default loading tips shown during AI response generation.
 * These are used as fallback when LOADING_TIPS env var is not configured.
 * Tips support markdown-style links: [text](url)
 */
export const DEFAULT_LOADING_TIPS = [
  "Tip: Clone sessions to quickly duplicate your setup for similar tasks",
  "Tip: Export chat transcripts as Markdown or PDF for documentation",
  "Tip: Add multiple repositories as context for cross-repo analysis",
  "Tip: Stopped sessions can be resumed without losing your progress",
  "Tip: Check MCP Servers to see which tools are available in your session",
  "Tip: Repository URLs are remembered for quick re-use across sessions",
  "Tip: Adjust temperature and max tokens in session settings for different tasks",
  "Tip: Connect Google Drive to export chats directly to your Drive",
  "Tip: Load custom workflows from your own Git repositories",
  "Tip: Use the artifacts panel to browse and download files created by AI",
];
