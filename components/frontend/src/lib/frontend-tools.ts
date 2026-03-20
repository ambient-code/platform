/**
 * Frontend-defined tools for AG-UI protocol
 *
 * These tools are executed in the frontend and passed to the agent during execution.
 * Reference: https://docs.ag-ui.com/concepts/tools#frontend-defined-tools
 */

import type { Tool } from '@ag-ui/client';
import { marked } from 'marked';
import DOMPurify from 'dompurify';

/**
 * Escape HTML to prevent XSS attacks
 */
function escapeHtml(text: string): string {
  const map: Record<string, string> = {
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#039;',
  };
  return text.replace(/[&<>"']/g, (char) => map[char]);
}

/**
 * Tool: open_in_browser
 *
 * Allows the agent to open files or content in a new browser tab.
 * The UI handles opening the file locally without backend execution.
 */
export const openInBrowserTool: Tool = {
  name: 'open_in_browser',
  description:
    'Open a file or HTML content in a new browser tab. Use this when the user asks to view, visualize, or open a file (especially HTML, markdown, images, PDFs). The UI will handle opening the file in a new tab with appropriate rendering.',
  parameters: {
    type: 'object',
    properties: {
      filePath: {
        type: 'string',
        description:
          'The path to the file to open (e.g., "artifacts/report.html", "workspace/chart.png"). This should be a path within the session workspace or artifacts directory.',
      },
      contentType: {
        type: 'string',
        enum: ['html', 'markdown', 'image', 'pdf', 'text', 'auto'],
        description:
          'The type of content to open. Use "auto" to detect based on file extension. Defaults to "auto".',
      },
      content: {
        type: 'string',
        description:
          'Optional: Raw content to open instead of reading from file. Useful for dynamically generated content.',
      },
    },
    required: ['filePath'],
  },
};

/**
 * All frontend-defined tools available to the agent
 */
export const frontendTools: Tool[] = [openInBrowserTool];

/**
 * Execute a frontend tool call
 *
 * @param toolName - Name of the tool to execute
 * @param args - Tool arguments (already parsed from JSON)
 * @returns Result message to send back to the agent
 */
export async function executeFrontendTool(
  toolName: string,
  args: Record<string, unknown>,
  context: {
    projectName: string;
    sessionName: string;
  }
): Promise<string> {
  if (toolName === 'open_in_browser') {
    return executeOpenInBrowser(args, context);
  }

  throw new Error(`Unknown frontend tool: ${toolName}`);
}

/**
 * Execute the open_in_browser tool
 */
async function executeOpenInBrowser(
  args: Record<string, unknown>,
  context: { projectName: string; sessionName: string }
): Promise<string> {
  const { filePath, contentType = 'auto', content } = args;

  if (typeof filePath !== 'string') {
    throw new Error('filePath must be a string');
  }

  try {
    let fileContent: string;
    let detectedType = contentType as string;

    // If content is provided directly, use it
    if (typeof content === 'string') {
      fileContent = content;
    } else {
      // Otherwise, fetch the file from the workspace
      const response = await fetch(
        `/api/projects/${encodeURIComponent(context.projectName)}/agentic-sessions/${encodeURIComponent(context.sessionName)}/workspace/${encodeURIComponent(filePath)}`
      );

      if (!response.ok) {
        throw new Error(`Failed to fetch file: ${response.statusText}`);
      }

      // Check content type to handle binary files appropriately
      const contentType = response.headers.get('content-type') || '';
      if (
        contentType.startsWith('image/') ||
        contentType === 'application/pdf' ||
        contentType === 'application/octet-stream'
      ) {
        // For binary files, convert to base64
        const arrayBuffer = await response.arrayBuffer();
        const bytes = new Uint8Array(arrayBuffer);
        let binary = '';
        for (let i = 0; i < bytes.length; i++) {
          binary += String.fromCharCode(bytes[i]);
        }
        fileContent = btoa(binary);
      } else {
        // For text files, use text()
        fileContent = await response.text();
      }
    }

    // Auto-detect content type if needed
    if (detectedType === 'auto') {
      const ext = filePath.toLowerCase().split('.').pop() || '';
      if (['html', 'htm'].includes(ext)) {
        detectedType = 'html';
      } else if (['md', 'mdx', 'markdown'].includes(ext)) {
        detectedType = 'markdown';
      } else if (['png', 'jpg', 'jpeg', 'gif', 'svg', 'webp'].includes(ext)) {
        detectedType = 'image';
      } else if (ext === 'pdf') {
        detectedType = 'pdf';
      } else {
        detectedType = 'text';
      }
    }

    // Determine MIME type
    const mimeTypeMap: Record<string, string> = {
      html: 'text/html',
      markdown: 'text/html', // Will render markdown as HTML
      image: 'image/*',
      pdf: 'application/pdf',
      text: 'text/plain',
    };

    const mimeType = mimeTypeMap[detectedType] || 'text/plain';

    // If markdown, convert to HTML first with sanitization
    let finalContent = fileContent;
    if (detectedType === 'markdown') {
      // Parse markdown and sanitize the result to prevent XSS
      const rawHtml = marked.parse(fileContent) as string;
      const sanitizedHtml = DOMPurify.sanitize(rawHtml);

      // Wrapper for rendering
      finalContent = `
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>${escapeHtml(filePath)}</title>
  <style>
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
      max-width: 800px;
      margin: 40px auto;
      padding: 20px;
      line-height: 1.6;
    }
    pre {
      background: #f5f5f5;
      padding: 10px;
      border-radius: 4px;
      overflow-x: auto;
    }
    code {
      background: #f5f5f5;
      padding: 2px 6px;
      border-radius: 3px;
      font-family: "Monaco", "Courier New", monospace;
    }
  </style>
</head>
<body>
  <div id="content">${sanitizedHtml}</div>
</body>
</html>
      `;
    }

    // Open in new tab with security flags
    const blob = new Blob([finalContent], { type: mimeType });
    const url = URL.createObjectURL(blob);
    const newWindow = window.open(url, '_blank', 'noopener,noreferrer');

    // Clean up URL after opening or after a delay
    setTimeout(() => URL.revokeObjectURL(url), newWindow ? 1000 : 5000);

    return `✓ Opened ${filePath} in a new browser tab`;
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    throw new Error(`Failed to open file in browser: ${errorMessage}`);
  }
}
