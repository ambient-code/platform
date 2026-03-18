import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { SessionSettingsModal } from '../session-settings-modal';
import type { AgenticSession } from '@/types/agentic-session';

vi.mock('@/services/queries/use-mcp', () => ({
  useMcpStatus: vi.fn(() => ({ data: { servers: [] } })),
}));

vi.mock('@/services/queries/use-integrations', () => ({
  useIntegrationsStatus: vi.fn(() => ({ data: null, isPending: false })),
}));

vi.mock('../settings/session-details', () => ({
  SessionDetails: () => <div data-testid="session-details">Session Details</div>,
}));

vi.mock('../settings/mcp-servers-panel', () => ({
  McpServersPanel: () => <div data-testid="mcp-panel">MCP Panel</div>,
}));

vi.mock('../settings/integrations-panel', () => ({
  IntegrationsPanel: () => <div data-testid="integrations-panel">Integrations Panel</div>,
}));

function makeSession(): AgenticSession {
  return {
    metadata: {
      name: 'test-session',
      namespace: 'default',
      uid: '123',
      creationTimestamp: '2026-01-01T00:00:00Z',
    },
    spec: {
      displayName: 'Test Session',
      initialPrompt: 'test',
      llmSettings: { model: 'claude-sonnet-4-20250514', temperature: 0, maxTokens: 100 },
      timeout: 3600,
    },
    status: { phase: 'Running' },
  };
}

describe('SessionSettingsModal', () => {
  const defaultProps = {
    open: true,
    onOpenChange: vi.fn(),
    session: makeSession(),
    projectName: 'test-project',
    onEditName: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders dialog when open is true', () => {
    render(<SessionSettingsModal {...defaultProps} />);
    expect(screen.getByRole('dialog')).toBeDefined();
  });

  it('renders Settings title', () => {
    render(<SessionSettingsModal {...defaultProps} />);
    expect(screen.getByText('Settings')).toBeDefined();
  });

  it('renders sidebar nav tabs (Session, MCP Servers, Integrations)', () => {
    render(<SessionSettingsModal {...defaultProps} />);
    expect(screen.getByText('Session')).toBeDefined();
    expect(screen.getByText('MCP Servers')).toBeDefined();
    expect(screen.getByText('Integrations')).toBeDefined();
  });

  it('shows Session details by default', () => {
    render(<SessionSettingsModal {...defaultProps} />);
    expect(screen.getByTestId('session-details')).toBeDefined();
  });

  it('clicking MCP Servers tab shows MCP panel', () => {
    render(<SessionSettingsModal {...defaultProps} />);
    fireEvent.click(screen.getByText('MCP Servers'));
    expect(screen.getByTestId('mcp-panel')).toBeDefined();
  });
});
