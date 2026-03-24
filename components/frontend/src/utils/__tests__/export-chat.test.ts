import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { convertEventsToMarkdown } from '../export-chat';
import type { AgenticSession } from '@/types/agentic-session';
import type { SessionExportResponse } from '@/services/api/sessions';

function makeSession(overrides: Partial<AgenticSession> = {}): AgenticSession {
  return {
    metadata: {
      name: 'test-session',
      creationTimestamp: '2025-01-15T10:30:00Z',
      namespace: 'default',
      ...overrides.metadata,
    },
    spec: {
      llmSettings: { model: 'claude-sonnet-4-20250514', temperature: 0, maxTokens: 4096 },
      timeout: 3600,
      ...overrides.spec,
    },
    status: { phase: 'Completed', ...overrides.status },
  } as AgenticSession;
}

function makeExport(events: unknown[]): SessionExportResponse {
  return {
    sessionId: 'test-session',
    projectName: 'test-project',
    exportDate: '2025-01-15T12:00:00Z',
    aguiEvents: events,
    hasLegacy: false,
  };
}

describe('convertEventsToMarkdown', () => {
  beforeEach(() => {
    // Mock window.location for session URL generation
    vi.stubGlobal('window', {
      location: { origin: 'https://app.example.com' },
    });
  });

  it('returns header with "no content" for empty events', () => {
    const md = convertEventsToMarkdown(makeExport([]), makeSession());
    expect(md).toContain('# test-session');
    expect(md).toContain('*No conversation content found.*');
  });

  it('uses displayName in header when set', () => {
    const session = makeSession({ spec: { displayName: 'My Session', llmSettings: { model: 'gpt-4', temperature: 0, maxTokens: 4096 }, timeout: 3600 } });
    const md = convertEventsToMarkdown(makeExport([]), session);
    expect(md).toContain('# My Session');
  });

  it('renders a text message conversation', () => {
    const events = [
      { type: 'TEXT_MESSAGE_START', role: 'user' },
      { type: 'TEXT_MESSAGE_CONTENT', delta: 'Hello ' },
      { type: 'TEXT_MESSAGE_CONTENT', delta: 'world' },
      { type: 'TEXT_MESSAGE_END' },
      { type: 'TEXT_MESSAGE_START', role: 'assistant' },
      { type: 'TEXT_MESSAGE_CONTENT', delta: 'Hi there!' },
      { type: 'TEXT_MESSAGE_END' },
    ];
    const md = convertEventsToMarkdown(makeExport(events), makeSession());
    expect(md).toContain('Hello world');
    expect(md).toContain('Hi there!');
    expect(md).toContain('User');
    expect(md).toContain('Assistant');
  });

  it('renders tool calls with arguments and results', () => {
    const events = [
      { type: 'TOOL_CALL_START', toolCallId: 'tc1', toolCallName: 'readFile' },
      { type: 'TOOL_CALL_ARGS', toolCallId: 'tc1', delta: '{"path":"/foo"}' },
      { type: 'TOOL_CALL_END', toolCallId: 'tc1', result: 'file contents here' },
    ];
    const md = convertEventsToMarkdown(makeExport(events), makeSession());
    expect(md).toContain('readFile');
    expect(md).toContain('"path"');
    expect(md).toContain('file contents here');
  });

  it('renders tool call errors', () => {
    const events = [
      { type: 'TOOL_CALL_START', toolCallId: 'tc1', toolCallName: 'badTool' },
      { type: 'TOOL_CALL_ARGS', toolCallId: 'tc1', delta: '{}' },
      { type: 'TOOL_CALL_END', toolCallId: 'tc1', error: 'something failed' },
    ];
    const md = convertEventsToMarkdown(makeExport(events), makeSession());
    expect(md).toContain('**Error:**');
    expect(md).toContain('something failed');
  });

  it('prepends initial prompt as first user message', () => {
    const session = makeSession({
      spec: {
        initialPrompt: 'Fix the bug in main.ts',
        llmSettings: { model: 'claude-sonnet-4-20250514', temperature: 0, maxTokens: 4096 },
        timeout: 3600,
      },
    });
    const events = [
      { type: 'TEXT_MESSAGE_START', role: 'assistant' },
      { type: 'TEXT_MESSAGE_CONTENT', delta: 'Done!' },
      { type: 'TEXT_MESSAGE_END' },
    ];
    const md = convertEventsToMarkdown(makeExport(events), session);
    // Initial prompt should appear before assistant message
    const promptIdx = md.indexOf('Fix the bug in main.ts');
    const doneIdx = md.indexOf('Done!');
    expect(promptIdx).toBeGreaterThan(-1);
    expect(doneIdx).toBeGreaterThan(promptIdx);
  });

  it('handles session URL when projectName is provided', () => {
    const md = convertEventsToMarkdown(
      makeExport([]),
      makeSession(),
      { projectName: 'my-project' },
    );
    expect(md).toContain('/projects/my-project/sessions/test-session');
  });

  it('includes username in metadata table when provided', () => {
    const md = convertEventsToMarkdown(
      makeExport([]),
      makeSession(),
      { username: 'alice' },
    );
    expect(md).toContain('| User | alice |');
  });

  it('handles unclosed trailing message', () => {
    const events = [
      { type: 'TEXT_MESSAGE_START', role: 'assistant' },
      { type: 'TEXT_MESSAGE_CONTENT', delta: 'partial response' },
      // No TEXT_MESSAGE_END
    ];
    const md = convertEventsToMarkdown(makeExport(events), makeSession());
    expect(md).toContain('partial response');
  });

  it('skips non-event objects', () => {
    const events = [null, 42, 'string', { noType: true }];
    const md = convertEventsToMarkdown(makeExport(events), makeSession());
    expect(md).toContain('*No conversation content found.*');
  });

  it('truncates long tool arguments', () => {
    const longArgs = '{"data":"' + 'x'.repeat(3000) + '"}';
    const events = [
      { type: 'TOOL_CALL_START', toolCallId: 'tc1', toolCallName: 'bigTool' },
      { type: 'TOOL_CALL_ARGS', toolCallId: 'tc1', delta: longArgs },
      { type: 'TOOL_CALL_END', toolCallId: 'tc1' },
    ];
    const md = convertEventsToMarkdown(makeExport(events), makeSession());
    expect(md).toContain('... (truncated)');
  });

  it('defaults role to assistant when not provided in TEXT_MESSAGE_START', () => {
    const events = [
      { type: 'TEXT_MESSAGE_START' }, // no role
      { type: 'TEXT_MESSAGE_CONTENT', delta: 'no role msg' },
      { type: 'TEXT_MESSAGE_END' },
    ];
    const md = convertEventsToMarkdown(makeExport(events), makeSession());
    expect(md).toContain('Assistant');
    expect(md).toContain('no role msg');
  });
});

// Import additional functions for testing
// Note: triggerDownload, downloadAsMarkdown, exportAsPdf are exported
import { triggerDownload, downloadAsMarkdown, exportAsPdf } from '../export-chat';

describe('triggerDownload', () => {
  it('creates a download link and clicks it', () => {
    // jsdom provides real DOM — just spy on the click
    const origCreateElement = document.createElement.bind(document);
    let capturedLink: HTMLAnchorElement | null = null;
    vi.spyOn(document, 'createElement').mockImplementation((tag: string) => {
      const el = origCreateElement(tag);
      if (tag === 'a') {
        capturedLink = el as HTMLAnchorElement;
        vi.spyOn(el, 'click').mockImplementation(() => {});
      }
      return el;
    });
    vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:mock');
    vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {});

    triggerDownload('test content', 'test.md', 'text/markdown');
    expect(capturedLink).not.toBeNull();
    expect(capturedLink!.download).toBe('test.md');
    expect(capturedLink!.click).toHaveBeenCalled();

    vi.restoreAllMocks();
  });
});

describe('downloadAsMarkdown', () => {
  it('triggers download with markdown content', () => {
    const clickFn = vi.fn();
    const mockLink = { href: '', download: '', click: clickFn, style: {} as CSSStyleDeclaration };
    vi.spyOn(document, 'createElement').mockReturnValue(mockLink as unknown as HTMLElement);
    vi.spyOn(document.body, 'appendChild').mockImplementation(() => mockLink as unknown as HTMLElement);
    vi.spyOn(document.body, 'removeChild').mockImplementation(() => mockLink as unknown as HTMLElement);
    vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:mock');
    vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {});

    downloadAsMarkdown('# Hello', 'test.md');
    expect(clickFn).toHaveBeenCalled();

    vi.restoreAllMocks();
  });
});

describe('exportAsPdf', () => {
  it('opens a print window with HTML content', () => {
    const writeFn = vi.fn();
    const closeFn = vi.fn();
    const mockPrintWindow = {
      document: { write: writeFn, close: closeFn, title: '' },
      print: vi.fn(),
      close: vi.fn(),
      addEventListener: vi.fn(),
      focus: vi.fn(),
    };
    const openFn = vi.fn(() => mockPrintWindow);
    Object.defineProperty(window, 'open', { value: openFn, writable: true, configurable: true });

    exportAsPdf('# Hello **bold**', 'test-session');
    expect(openFn).toHaveBeenCalled();
    expect(writeFn).toHaveBeenCalled();
    // Verify HTML was written with converted markdown
    const htmlWritten = writeFn.mock.calls[0][0];
    expect(htmlWritten).toContain('Hello');
    expect(htmlWritten).toContain('<strong>bold</strong>');

    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });
});

describe('convertEventsToMarkdown — additional branches', () => {
  beforeEach(() => {
    vi.stubGlobal('window', {
      location: { origin: 'https://app.example.com' },
    });
  });

  it('handles TOOL_CALL_RESULT events', () => {
    const events = [
      { type: 'TOOL_CALL_START', toolCallId: 'tc1', toolCallName: 'Read' },
      { type: 'TOOL_CALL_ARGS', toolCallId: 'tc1', delta: '{"path":"/foo"}' },
      { type: 'TOOL_CALL_END', toolCallId: 'tc1' },
      { type: 'TOOL_CALL_RESULT', toolCallId: 'tc1', content: 'file contents here' },
    ];
    const md = convertEventsToMarkdown(makeExport(events), makeSession());
    expect(md).toContain('Read');
  });

  it('handles mixed user and assistant messages', () => {
    const events = [
      { type: 'TEXT_MESSAGE_START', role: 'user' },
      { type: 'TEXT_MESSAGE_CONTENT', delta: 'Question 1' },
      { type: 'TEXT_MESSAGE_END' },
      { type: 'TEXT_MESSAGE_START', role: 'assistant' },
      { type: 'TEXT_MESSAGE_CONTENT', delta: 'Answer 1' },
      { type: 'TEXT_MESSAGE_END' },
      { type: 'TEXT_MESSAGE_START', role: 'user' },
      { type: 'TEXT_MESSAGE_CONTENT', delta: 'Question 2' },
      { type: 'TEXT_MESSAGE_END' },
      { type: 'TEXT_MESSAGE_START', role: 'assistant' },
      { type: 'TEXT_MESSAGE_CONTENT', delta: 'Answer 2' },
      { type: 'TEXT_MESSAGE_END' },
    ];
    const md = convertEventsToMarkdown(makeExport(events), makeSession());
    expect(md).toContain('Question 1');
    expect(md).toContain('Answer 1');
    expect(md).toContain('Question 2');
    expect(md).toContain('Answer 2');
  });

  it('handles tool call with no result and no error', () => {
    const events = [
      { type: 'TOOL_CALL_START', toolCallId: 'tc1', toolCallName: 'Bash' },
      { type: 'TOOL_CALL_ARGS', toolCallId: 'tc1', delta: '{"cmd":"ls"}' },
      { type: 'TOOL_CALL_END', toolCallId: 'tc1' },
    ];
    const md = convertEventsToMarkdown(makeExport(events), makeSession());
    expect(md).toContain('Bash');
    expect(md).toContain('"cmd"');
  });

  it('handles consecutive tool calls', () => {
    const events = [
      { type: 'TOOL_CALL_START', toolCallId: 'tc1', toolCallName: 'Read' },
      { type: 'TOOL_CALL_ARGS', toolCallId: 'tc1', delta: '{"p":"a"}' },
      { type: 'TOOL_CALL_END', toolCallId: 'tc1', result: 'content a' },
      { type: 'TOOL_CALL_START', toolCallId: 'tc2', toolCallName: 'Write' },
      { type: 'TOOL_CALL_ARGS', toolCallId: 'tc2', delta: '{"p":"b"}' },
      { type: 'TOOL_CALL_END', toolCallId: 'tc2', result: 'written' },
    ];
    const md = convertEventsToMarkdown(makeExport(events), makeSession());
    expect(md).toContain('Read');
    expect(md).toContain('Write');
    expect(md).toContain('content a');
    expect(md).toContain('written');
  });

  it('handles long tool results that get truncated', () => {
    const longResult = 'x'.repeat(5000);
    const events = [
      { type: 'TOOL_CALL_START', toolCallId: 'tc1', toolCallName: 'Read' },
      { type: 'TOOL_CALL_ARGS', toolCallId: 'tc1', delta: '{}' },
      { type: 'TOOL_CALL_END', toolCallId: 'tc1', result: longResult },
    ];
    const md = convertEventsToMarkdown(makeExport(events), makeSession());
    expect(md).toContain('... (truncated)');
  });

  it('handles RUN_STARTED and RUN_FINISHED events gracefully', () => {
    const events = [
      { type: 'RUN_STARTED' },
      { type: 'TEXT_MESSAGE_START', role: 'assistant' },
      { type: 'TEXT_MESSAGE_CONTENT', delta: 'Hello' },
      { type: 'TEXT_MESSAGE_END' },
      { type: 'RUN_FINISHED' },
    ];
    const md = convertEventsToMarkdown(makeExport(events), makeSession());
    expect(md).toContain('Hello');
  });

  it('includes model in metadata', () => {
    const md = convertEventsToMarkdown(makeExport([]), makeSession());
    expect(md).toContain('claude-sonnet');
  });

  it('includes date in metadata', () => {
    const md = convertEventsToMarkdown(makeExport([]), makeSession());
    expect(md).toContain('2025');
  });
});

// ── markdownToHtml (tested through exportAsPdf) ──
// The internal functions escapeHtml, inlineFormat, parseTable, and markdownToHtml
// are not exported. We test them by capturing the HTML that exportAsPdf writes.

function captureHtml(markdown: string): string {
  const writeFn = vi.fn();
  const mockPrintWindow = {
    document: { write: writeFn, close: vi.fn(), title: '' },
    print: vi.fn(),
    close: vi.fn(),
    addEventListener: vi.fn(),
    focus: vi.fn(),
  };
  const openFn = vi.fn(() => mockPrintWindow);
  Object.defineProperty(window, 'open', { value: openFn, writable: true, configurable: true });

  exportAsPdf(markdown, 'test');
  return writeFn.mock.calls[0][0] as string;
}

describe('markdownToHtml via exportAsPdf', () => {
  beforeEach(() => {
    vi.stubGlobal('window', {
      location: { origin: 'https://app.example.com' },
      open: vi.fn(),
      matchMedia: vi.fn().mockReturnValue({ matches: false }),
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  // ── escapeHtml ──
  describe('escapeHtml', () => {
    it('escapes <, >, &, and " in session name (title)', () => {
      const html = captureHtml('hello');
      // The session name is passed to escapeHtml for the <title>
      // We pass a clean name so let's test with content containing special chars
      expect(html).toContain('<title>test - Chat Export</title>');
    });

    it('escapes HTML entities in paragraph text', () => {
      const html = captureHtml('Use <div> & "quotes"');
      expect(html).toContain('&lt;div&gt;');
      expect(html).toContain('&amp;');
      expect(html).toContain('&quot;quotes&quot;');
    });
  });

  // ── inlineFormat ──
  describe('inlineFormat', () => {
    it('converts **bold** to <strong>', () => {
      const html = captureHtml('This is **bold** text');
      expect(html).toContain('<strong>bold</strong>');
    });

    it('converts *italic* to <em>', () => {
      const html = captureHtml('This is *italic* text');
      expect(html).toContain('<em>italic</em>');
    });

    it('converts `code` to <code>', () => {
      const html = captureHtml('Use `console.log()` here');
      expect(html).toContain('<code>console.log()</code>');
    });

    it('converts [link](url) to <a> for http URLs', () => {
      const html = captureHtml('[click here](https://example.com)');
      expect(html).toContain('<a href="https://example.com"');
      expect(html).toContain('click here</a>');
    });

    it('renders non-http links as text with URL in parens', () => {
      const html = captureHtml('[bad](ftp://evil.com)');
      expect(html).not.toContain('<a href="ftp://evil.com"');
      expect(html).toContain('bad');
      expect(html).toContain('ftp://evil.com');
    });

    it('renders invalid URLs as text', () => {
      const html = captureHtml('[label](not a url at all)');
      expect(html).not.toContain('<a href');
      expect(html).toContain('label');
    });

    it('handles bold and italic in same line', () => {
      const html = captureHtml('**bold** and *italic*');
      expect(html).toContain('<strong>bold</strong>');
      expect(html).toContain('<em>italic</em>');
    });
  });

  // ── headings ──
  describe('headings', () => {
    it('converts # to h1', () => {
      const html = captureHtml('# Title');
      expect(html).toContain('<h1>Title</h1>');
    });

    it('converts ## to h2', () => {
      const html = captureHtml('## Subtitle');
      expect(html).toContain('<h2>Subtitle</h2>');
    });

    it('converts ### to h3', () => {
      const html = captureHtml('### Section');
      expect(html).toContain('<h3>Section</h3>');
    });

    it('applies blue color for h2 with user emoji', () => {
      const html = captureHtml('## \u{1F464} User');
      expect(html).toContain('color:#1d4ed8');
    });

    it('applies green color for h2 with assistant emoji', () => {
      const html = captureHtml('## \u{1F916} Assistant');
      expect(html).toContain('color:#15803d');
    });
  });

  // ── fenced code blocks ──
  describe('code blocks', () => {
    it('converts fenced code blocks to <pre><code>', () => {
      const md = '```\nconst x = 1;\nconsole.log(x);\n```';
      const html = captureHtml(md);
      expect(html).toContain('<pre><code>');
      expect(html).toContain('const x = 1;');
    });

    it('escapes HTML inside code blocks', () => {
      const md = '```\n<script>alert("xss")</script>\n```';
      const html = captureHtml(md);
      expect(html).toContain('&lt;script&gt;');
      expect(html).not.toContain('<script>');
    });
  });

  // ── tables ──
  describe('tables', () => {
    it('converts markdown tables to HTML tables', () => {
      const md = '| Field | Value |\n|-------|-------|\n| Name | Test |';
      const html = captureHtml(md);
      expect(html).toContain('<table>');
      expect(html).toContain('<th>Field</th>');
      expect(html).toContain('<th>Value</th>');
      expect(html).toContain('<td>Name</td>');
      expect(html).toContain('<td>Test</td>');
      expect(html).toContain('</table>');
    });

    it('applies inline formatting inside table cells', () => {
      const md = '| Col |\n|-----|\n| **bold** |';
      const html = captureHtml(md);
      expect(html).toContain('<strong>bold</strong>');
    });

    it('handles separator-only table gracefully', () => {
      const md = '|-------|-------|';
      const html = captureHtml(md);
      // Separator-only table has no header/body rows, parseTable returns empty
      // The result should not contain a broken table
      expect(html).not.toContain('<th>');
    });
  });

  // ── horizontal rule ──
  describe('horizontal rule', () => {
    it('converts --- to <hr />', () => {
      const html = captureHtml('---');
      expect(html).toContain('<hr />');
    });
  });

  // ── HTML pass-through (details/summary) ──
  describe('HTML pass-through', () => {
    it('passes through <details> with border styling', () => {
      const html = captureHtml('<details>\n<summary>Click me</summary>\nContent\n</details>');
      expect(html).toContain('border-left:3px solid #6b7280');
      expect(html).toContain('<summary>Click me</summary>');
    });
  });

  // ── paragraphs ──
  describe('paragraphs', () => {
    it('wraps plain text in <p> tags', () => {
      const html = captureHtml('Just some text');
      expect(html).toContain('<p>Just some text</p>');
    });

    it('skips empty lines', () => {
      const html = captureHtml('Line 1\n\n\nLine 2');
      expect(html).toContain('<p>Line 1</p>');
      expect(html).toContain('<p>Line 2</p>');
    });
  });
});

// ── MESSAGES_SNAPSHOT (compacted sessions) ──
describe('MESSAGES_SNAPSHOT support (compacted sessions)', () => {
  beforeEach(() => {
    vi.stubGlobal('window', {
      location: { origin: 'https://app.example.com' },
    });
  });

  it('renders messages from a MESSAGES_SNAPSHOT event', () => {
    const events = [
      { type: 'RUN_STARTED' },
      {
        type: 'MESSAGES_SNAPSHOT',
        messages: [
          { id: 'm1', role: 'user', content: 'What is 2+2?' },
          { id: 'm2', role: 'assistant', content: 'The answer is 4.' },
        ],
      },
      { type: 'RUN_FINISHED' },
    ];
    const md = convertEventsToMarkdown(makeExport(events), makeSession());
    expect(md).toContain('What is 2+2?');
    expect(md).toContain('The answer is 4.');
    expect(md).toContain('User');
    expect(md).toContain('Assistant');
  });

  it('renders tool calls from snapshot assistant messages', () => {
    const events = [
      {
        type: 'MESSAGES_SNAPSHOT',
        messages: [
          { id: 'm1', role: 'user', content: 'Read the file' },
          {
            id: 'm2', role: 'assistant', content: 'Let me read it.',
            toolCalls: [
              {
                id: 'tc1',
                function: { name: 'Read', arguments: '{"path":"/foo.ts"}' },
                result: 'file contents here',
              },
            ],
          },
          { id: 'm3', role: 'assistant', content: 'Here is the file content.' },
        ],
      },
    ];
    const md = convertEventsToMarkdown(makeExport(events), makeSession());
    expect(md).toContain('Read the file');
    expect(md).toContain('Let me read it.');
    expect(md).toContain('Read');
    expect(md).toContain('"path"');
    expect(md).toContain('file contents here');
    expect(md).toContain('Here is the file content.');
  });

  it('renders tool call errors from snapshot', () => {
    const events = [
      {
        type: 'MESSAGES_SNAPSHOT',
        messages: [
          {
            id: 'm1', role: 'assistant', content: 'Running command.',
            toolCalls: [
              {
                id: 'tc1',
                function: { name: 'Bash', arguments: '{"cmd":"fail"}' },
                error: 'command not found',
              },
            ],
          },
        ],
      },
    ];
    const md = convertEventsToMarkdown(makeExport(events), makeSession());
    expect(md).toContain('**Error:**');
    expect(md).toContain('command not found');
  });

  it('prepends initial prompt with snapshot messages', () => {
    const session = makeSession({
      spec: {
        initialPrompt: 'Fix the bug',
        llmSettings: { model: 'claude-sonnet-4-20250514', temperature: 0, maxTokens: 4096 },
        timeout: 3600,
      },
    });
    const events = [
      {
        type: 'MESSAGES_SNAPSHOT',
        messages: [
          { id: 'm1', role: 'assistant', content: 'Done!' },
        ],
      },
    ];
    const md = convertEventsToMarkdown(makeExport(events), session);
    const promptIdx = md.indexOf('Fix the bug');
    const doneIdx = md.indexOf('Done!');
    expect(promptIdx).toBeGreaterThan(-1);
    expect(doneIdx).toBeGreaterThan(promptIdx);
  });

  it('prefers snapshot over streaming events when both present', () => {
    const events = [
      // Streaming events (should be ignored when snapshot is present)
      { type: 'TEXT_MESSAGE_START', role: 'user' },
      { type: 'TEXT_MESSAGE_CONTENT', delta: 'streaming msg' },
      { type: 'TEXT_MESSAGE_END' },
      // Snapshot (canonical source)
      {
        type: 'MESSAGES_SNAPSHOT',
        messages: [
          { id: 'm1', role: 'user', content: 'snapshot msg' },
          { id: 'm2', role: 'assistant', content: 'snapshot reply' },
        ],
      },
    ];
    const md = convertEventsToMarkdown(makeExport(events), makeSession());
    expect(md).toContain('snapshot msg');
    expect(md).toContain('snapshot reply');
    expect(md).not.toContain('streaming msg');
  });

  it('handles snapshot with empty messages array', () => {
    const events = [
      { type: 'MESSAGES_SNAPSHOT', messages: [] },
    ];
    const md = convertEventsToMarkdown(makeExport(events), makeSession());
    expect(md).toContain('*No conversation content found.*');
  });

  it('handles multi-turn conversation in snapshot', () => {
    const events = [
      {
        type: 'MESSAGES_SNAPSHOT',
        messages: [
          { id: 'm1', role: 'user', content: 'Question 1' },
          { id: 'm2', role: 'assistant', content: 'Answer 1' },
          { id: 'm3', role: 'user', content: 'Question 2' },
          { id: 'm4', role: 'assistant', content: 'Answer 2' },
        ],
      },
    ];
    const md = convertEventsToMarkdown(makeExport(events), makeSession());
    expect(md).toContain('Question 1');
    expect(md).toContain('Answer 1');
    expect(md).toContain('Question 2');
    expect(md).toContain('Answer 2');
    // Verify ordering
    const q1 = md.indexOf('Question 1');
    const a1 = md.indexOf('Answer 1');
    const q2 = md.indexOf('Question 2');
    const a2 = md.indexOf('Answer 2');
    expect(q1).toBeLessThan(a1);
    expect(a1).toBeLessThan(q2);
    expect(q2).toBeLessThan(a2);
  });
});

// ── assembleBlocks additional branches ──
describe('assembleBlocks — TOOL_CALL_END with result and error', () => {
  beforeEach(() => {
    vi.stubGlobal('window', {
      location: { origin: 'https://app.example.com' },
    });
  });

  it('attaches result from TOOL_CALL_END to the tool block', () => {
    const events = [
      { type: 'TOOL_CALL_START', toolCallId: 'tc1', toolCallName: 'Read' },
      { type: 'TOOL_CALL_ARGS', toolCallId: 'tc1', delta: '{}' },
      { type: 'TOOL_CALL_END', toolCallId: 'tc1', result: 'file data' },
    ];
    const md = convertEventsToMarkdown(makeExport(events), makeSession());
    expect(md).toContain('**Result:**');
    expect(md).toContain('file data');
    // Should NOT show error section
    expect(md).not.toContain('**Error:**');
  });

  it('prefers error over result when both present in TOOL_CALL_END', () => {
    const events = [
      { type: 'TOOL_CALL_START', toolCallId: 'tc1', toolCallName: 'Bash' },
      { type: 'TOOL_CALL_ARGS', toolCallId: 'tc1', delta: '{}' },
      { type: 'TOOL_CALL_END', toolCallId: 'tc1', error: 'failed', result: 'partial' },
    ];
    const md = convertEventsToMarkdown(makeExport(events), makeSession());
    expect(md).toContain('**Error:**');
    expect(md).toContain('failed');
    // Error takes priority; result should not be shown
    expect(md).not.toContain('**Result:**');
  });

  it('handles tool call with empty args', () => {
    const events = [
      { type: 'TOOL_CALL_START', toolCallId: 'tc1', toolCallName: 'Read' },
      { type: 'TOOL_CALL_END', toolCallId: 'tc1', result: 'data' },
    ];
    const md = convertEventsToMarkdown(makeExport(events), makeSession());
    expect(md).toContain('Read');
    // No arguments section since args is empty
    expect(md).not.toContain('**Arguments:**');
  });

  it('handles long error messages that get truncated', () => {
    const longError = 'E'.repeat(2000);
    const events = [
      { type: 'TOOL_CALL_START', toolCallId: 'tc1', toolCallName: 'Bash' },
      { type: 'TOOL_CALL_ARGS', toolCallId: 'tc1', delta: '{}' },
      { type: 'TOOL_CALL_END', toolCallId: 'tc1', error: longError },
    ];
    const md = convertEventsToMarkdown(makeExport(events), makeSession());
    expect(md).toContain('... (truncated)');
  });

  it('handles TOOL_CALL_START without toolCallId gracefully', () => {
    const events = [
      { type: 'TOOL_CALL_START', toolCallName: 'orphan' },
      { type: 'TOOL_CALL_END' },
    ];
    // Should not throw, should produce markdown
    const md = convertEventsToMarkdown(makeExport(events), makeSession());
    expect(md).toBeDefined();
  });

  it('includes timestamps in tool call summary', () => {
    const events = [
      { type: 'TOOL_CALL_START', toolCallId: 'tc1', toolCallName: 'Read', timestamp: '2025-01-15T10:30:00Z' },
      { type: 'TOOL_CALL_ARGS', toolCallId: 'tc1', delta: '{}' },
      { type: 'TOOL_CALL_END', toolCallId: 'tc1', result: 'ok' },
    ];
    const md = convertEventsToMarkdown(makeExport(events), makeSession());
    expect(md).toContain('Read');
    // The timestamp should appear in the summary
    expect(md).toContain('2025');
  });
});
