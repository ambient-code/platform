/**
 * Chat export utilities
 * Converts AG-UI events to human-readable Markdown and supports download/PDF export.
 */

import { AGUIEventType } from '@/types/agui';
import type { AgenticSession } from '@/types/agentic-session';
import type { SessionExportResponse } from '@/services/api/sessions';

type ExportEvent = {
  type: string;
  role?: string;
  delta?: string;
  toolCallId?: string;
  toolCallName?: string;
  result?: string;
  error?: string;
  timestamp?: string;
};

function isExportEvent(raw: unknown): raw is ExportEvent {
  return typeof raw === 'object' && raw !== null && 'type' in raw;
}

const MAX_TOOL_ARGS_LENGTH = 2000;
const MAX_ERROR_LENGTH = 1000;
const MAX_RESULT_LENGTH = 2000;

type ConversationBlock =
  | { kind: 'message'; role: string; content: string; timestamp?: string }
  | { kind: 'tool'; name: string; args: string; result?: string; error?: string; timestamp?: string };

/**
 * Walk the raw AG-UI event array and assemble conversation blocks.
 */
function assembleBlocks(events: unknown[]): ConversationBlock[] {
  const blocks: ConversationBlock[] = [];
  let currentRole: string | null = null;
  let currentContent = '';
  let currentTimestamp: string | undefined;
  const toolCalls = new Map<string, { name: string; args: string; timestamp?: string }>();

  for (const raw of events) {
    if (!isExportEvent(raw)) continue;
    const ev = raw;

    switch (ev.type) {
      case AGUIEventType.TEXT_MESSAGE_START:
        currentRole = ev.role ?? 'assistant';
        currentContent = '';
        currentTimestamp = ev.timestamp;
        break;

      case AGUIEventType.TEXT_MESSAGE_CONTENT:
        if (ev.delta) currentContent += ev.delta;
        break;

      case AGUIEventType.TEXT_MESSAGE_END:
        if (currentRole && currentContent.trim()) {
          blocks.push({ kind: 'message', role: currentRole, content: currentContent.trim(), timestamp: currentTimestamp });
        }
        currentRole = null;
        currentContent = '';
        currentTimestamp = undefined;
        break;

      case AGUIEventType.TOOL_CALL_START:
        if (ev.toolCallId) {
          toolCalls.set(ev.toolCallId, { name: ev.toolCallName ?? 'unknown', args: '', timestamp: ev.timestamp });
        }
        break;

      case AGUIEventType.TOOL_CALL_ARGS:
        if (ev.toolCallId) {
          const tc = toolCalls.get(ev.toolCallId);
          if (tc && ev.delta) tc.args += ev.delta;
        }
        break;

      case AGUIEventType.TOOL_CALL_END:
        if (ev.toolCallId) {
          const tc = toolCalls.get(ev.toolCallId);
          if (tc) {
            blocks.push({
              kind: 'tool',
              name: tc.name,
              args: tc.args,
              result: ev.result,
              error: ev.error,
              timestamp: tc.timestamp,
            });
            toolCalls.delete(ev.toolCallId);
          }
        }
        break;

      default:
        break;
    }
  }

  // Flush any trailing message that wasn't closed
  if (currentRole && currentContent.trim()) {
    blocks.push({ kind: 'message', role: currentRole, content: currentContent.trim(), timestamp: currentTimestamp });
  }

  return blocks;
}

function formatTimestamp(ts?: string): string {
  if (!ts) return '';
  try {
    return new Date(ts).toLocaleString();
  } catch {
    return ts;
  }
}

function prettyJson(raw: string): string {
  try {
    return JSON.stringify(JSON.parse(raw), null, 2);
  } catch {
    return raw;
  }
}

function truncate(s: string, max: number): string {
  if (s.length <= max) return s;
  return s.slice(0, max) + '\n... (truncated)';
}

/**
 * Convert AG-UI events into a Markdown string.
 */
export function convertEventsToMarkdown(
  exportData: SessionExportResponse,
  session: AgenticSession,
): string {
  const displayName = session.spec.displayName || session.metadata.name;
  const model = session.spec.llmSettings.model;
  const created = formatTimestamp(session.metadata.creationTimestamp);
  const phase = session.status?.phase ?? 'Unknown';

  const lines: string[] = [
    `# ${displayName}`,
    '',
    `| Field | Value |`,
    `|-------|-------|`,
    `| Session | ${session.metadata.name} |`,
    `| Model | ${model} |`,
    `| Status | ${phase} |`,
    `| Created | ${created} |`,
    `| Exported | ${new Date().toLocaleString()} |`,
    '',
    '---',
    '',
  ];

  const blocks = assembleBlocks(exportData.aguiEvents);

  if (blocks.length === 0) {
    lines.push('*No conversation content found.*');
    return lines.join('\n');
  }

  for (const block of blocks) {
    if (block.kind === 'message') {
      const roleLabel = block.role === 'user' ? 'User' : block.role === 'assistant' ? 'Assistant' : block.role;
      const ts = formatTimestamp(block.timestamp);
      lines.push(`## ${roleLabel}`);
      if (ts) lines.push(`*${ts}*`);
      lines.push('');
      lines.push(block.content);
      lines.push('');
    } else {
      const ts = formatTimestamp(block.timestamp);
      lines.push(`<details>`);
      lines.push(`<summary>Tool: ${block.name}${ts ? ` (${ts})` : ''}</summary>`);
      lines.push('');
      if (block.args.trim()) {
        lines.push('**Arguments:**');
        lines.push('```json');
        lines.push(truncate(prettyJson(block.args), MAX_TOOL_ARGS_LENGTH));
        lines.push('```');
      }
      if (block.error) {
        lines.push('**Error:**');
        lines.push('```');
        lines.push(truncate(block.error, MAX_ERROR_LENGTH));
        lines.push('```');
      } else if (block.result) {
        lines.push('**Result:**');
        lines.push('```');
        lines.push(truncate(block.result, MAX_RESULT_LENGTH));
        lines.push('```');
      }
      lines.push('</details>');
      lines.push('');
    }
  }

  return lines.join('\n');
}

/**
 * Trigger a browser file download with the given content.
 */
export function triggerDownload(content: string, filename: string, mimeType: string): void {
  const blob = new Blob([content], { type: mimeType });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  link.click();
  URL.revokeObjectURL(url);
}

/**
 * Download a Markdown string as a `.md` file.
 */
export function downloadAsMarkdown(markdown: string, filename: string): void {
  triggerDownload(markdown, filename, 'text/markdown;charset=utf-8');
}

/**
 * Render markdown as styled HTML in a new window and trigger the browser print dialog
 * (which offers "Save as PDF").
 */
export function exportAsPdf(markdown: string, sessionName: string): void {
  const html = markdownToHtml(markdown);

  const printWindow = window.open('', '_blank');
  if (!printWindow) {
    throw new Error('Failed to open print window. Please allow popups for this site.');
  }

  printWindow.document.write(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8" />
  <title>${escapeHtml(sessionName)} - Chat Export</title>
  <style>
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
      max-width: 800px;
      margin: 2rem auto;
      padding: 0 1rem;
      color: #1a1a1a;
      line-height: 1.6;
    }
    h1 { font-size: 1.5rem; border-bottom: 2px solid #e5e7eb; padding-bottom: 0.5rem; }
    h2 { font-size: 1.15rem; margin-top: 1.5rem; color: #374151; }
    table { border-collapse: collapse; margin-bottom: 1rem; font-size: 0.9rem; }
    th, td { border: 1px solid #d1d5db; padding: 0.35rem 0.75rem; text-align: left; }
    th { background: #f3f4f6; }
    hr { border: none; border-top: 1px solid #e5e7eb; margin: 1.5rem 0; }
    pre { background: #f9fafb; border: 1px solid #e5e7eb; border-radius: 4px; padding: 0.75rem; overflow-x: auto; font-size: 0.8rem; }
    code { font-family: "SF Mono", "Fira Code", monospace; font-size: 0.85em; }
    details { background: #f9fafb; border: 1px solid #e5e7eb; border-radius: 4px; padding: 0.5rem 0.75rem; margin: 0.5rem 0; }
    summary { cursor: pointer; font-weight: 500; }
    em { color: #6b7280; font-size: 0.85rem; }
    @media print {
      body { margin: 0; max-width: 100%; }
      details { break-inside: avoid; }
      pre { white-space: pre-wrap; word-break: break-all; }
    }
  </style>
</head>
<body>
${html}
</body>
</html>`);

  printWindow.document.close();

  // Trigger print once: load listener with timeout fallback
  const timeoutId = setTimeout(() => printWindow.print(), 500);
  printWindow.addEventListener('load', () => {
    clearTimeout(timeoutId);
    printWindow.print();
  });
}

// --------------- Helpers ---------------

function escapeHtml(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

/**
 * Minimal markdown-to-HTML converter.
 * Handles headings, tables, code blocks, paragraphs, <details>, hr, bold, italic, and inline code.
 * Only processes the structured output we generate — not a general-purpose parser.
 */
function markdownToHtml(md: string): string {
  const lines = md.split('\n');
  const out: string[] = [];
  let i = 0;

  while (i < lines.length) {
    const line = lines[i];

    if (line.trim() === '') {
      i++;
      continue;
    }

    // Pass through HTML tags directly (<details>, <summary>, </details>)
    if (/^\s*<\/?(?:details|summary)/.test(line)) {
      out.push(line);
      i++;
      continue;
    }

    // Horizontal rule
    if (/^-{3,}$/.test(line.trim())) {
      out.push('<hr />');
      i++;
      continue;
    }

    // Headings
    const headingMatch = line.match(/^(#{1,6})\s+(.*)$/);
    if (headingMatch) {
      const level = headingMatch[1].length;
      out.push(`<h${level}>${inlineFormat(headingMatch[2])}</h${level}>`);
      i++;
      continue;
    }

    // Fenced code block
    if (line.trim().startsWith('```')) {
      i++;
      const codeLines: string[] = [];
      while (i < lines.length && !lines[i].trim().startsWith('```')) {
        codeLines.push(lines[i]);
        i++;
      }
      i++; // skip closing ```
      out.push(`<pre><code>${escapeHtml(codeLines.join('\n'))}</code></pre>`);
      continue;
    }

    // Table (starts with |)
    if (line.trim().startsWith('|')) {
      const tableLines: string[] = [];
      while (i < lines.length && lines[i].trim().startsWith('|')) {
        tableLines.push(lines[i]);
        i++;
      }
      out.push(parseTable(tableLines));
      continue;
    }

    // Paragraph
    out.push(`<p>${inlineFormat(line)}</p>`);
    i++;
  }

  return out.join('\n');
}

function inlineFormat(s: string): string {
  // Escape HTML first to prevent XSS from message content
  let result = escapeHtml(s);
  result = result.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');
  result = result.replace(/(?<!\*)\*([^*]+?)\*(?!\*)/g, '<em>$1</em>');
  result = result.replace(/`([^`]+?)`/g, '<code>$1</code>');
  return result;
}

function parseTable(rows: string[]): string {
  const parsed = rows
    .filter((r) => !/^\|\s*[-:]+/.test(r))
    .map((r) =>
      r
        .split('|')
        .slice(1, -1)
        .map((c) => c.trim()),
    );

  if (parsed.length === 0) return '';

  const [header, ...body] = parsed;
  let html = '<table><thead><tr>';
  for (const cell of header) html += `<th>${inlineFormat(cell)}</th>`;
  html += '</tr></thead><tbody>';
  for (const row of body) {
    html += '<tr>';
    for (const cell of row) html += `<td>${inlineFormat(cell)}</td>`;
    html += '</tr>';
  }
  html += '</tbody></table>';
  return html;
}
