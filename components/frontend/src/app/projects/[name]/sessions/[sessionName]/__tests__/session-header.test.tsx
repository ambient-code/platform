import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { SessionHeader } from '../session-header';
import type { AgenticSession } from '@/types/agentic-session';

const mockMutate = vi.fn();
const mockRefetch = vi.fn().mockResolvedValue({ data: null });

vi.mock('@/services/queries', () => ({
  useUpdateSessionDisplayName: vi.fn(() => ({
    mutate: mockMutate,
    isPending: false,
  })),
  useCurrentUser: vi.fn(() => ({
    data: { displayName: 'Test User', username: 'testuser', email: 'test@test.com' },
  })),
  useSessionExport: vi.fn(() => ({
    refetch: mockRefetch,
  })),
}));

vi.mock('@/services/queries/use-mcp', () => ({
  useMcpStatus: vi.fn(() => ({ data: null })),
}));

vi.mock('@/services/queries/use-google', () => ({
  useGoogleStatus: vi.fn(() => ({ data: null })),
}));

vi.mock('@/services/queries/use-project-access', () => ({
  useProjectAccess: vi.fn(() => ({
    data: { userRole: 'admin' },
  })),
}));

vi.mock('@/hooks/use-toast', () => ({
  successToast: vi.fn(),
  errorToast: vi.fn(),
}));

vi.mock('@/services/api/sessions', () => ({
  saveToGoogleDrive: vi.fn(),
}));

vi.mock('@/utils/export-chat', () => ({
  convertEventsToMarkdown: vi.fn(() => '# Test'),
  downloadAsMarkdown: vi.fn(),
  exportAsPdf: vi.fn(),
}));

vi.mock('@/components/clone-session-dialog', () => ({
  CloneSessionDialog: ({ trigger }: { trigger: React.ReactNode }) => <div>{trigger}</div>,
}));

vi.mock('@/components/session-details-modal', () => ({
  SessionDetailsModal: () => null,
}));

vi.mock('@/components/edit-session-name-dialog', () => ({
  EditSessionNameDialog: () => null,
}));

function makeSession(phase: string, stoppedReason?: string): AgenticSession {
  return {
    metadata: { name: 'test-session', namespace: 'default', uid: '123', creationTimestamp: '' },
    spec: {
      displayName: 'Test Session',
      initialPrompt: 'test',
      llmSettings: { model: 'test', temperature: 0, maxTokens: 100 },
      timeout: 3600,
    },
    status: {
      phase: phase as NonNullable<AgenticSession['status']>['phase'],
      stoppedReason,
    },
  } as AgenticSession;
}

describe('SessionHeader', () => {
  const defaultProps = {
    projectName: 'test-project',
    actionLoading: null as string | null,
    onRefresh: vi.fn(),
    onStop: vi.fn(),
    onContinue: vi.fn(),
    onDelete: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders Stop button when session is Running', () => {
    render(
      <SessionHeader
        {...defaultProps}
        session={makeSession('Running')}
        renderMode="actions-only"
      />
    );
    expect(screen.getByText('Stop')).toBeDefined();
    expect(screen.queryByText('Resume')).toBeNull();
  });

  it('renders Resume button when session is Stopped', () => {
    render(
      <SessionHeader
        {...defaultProps}
        session={makeSession('Stopped')}
        renderMode="actions-only"
      />
    );
    expect(screen.getByText('Resume')).toBeDefined();
    expect(screen.queryByText('Stop')).toBeNull();
  });

  it('renders kebab menu trigger when session is Completed', () => {
    render(
      <SessionHeader
        {...defaultProps}
        session={makeSession('Completed')}
        renderMode="kebab-only"
      />
    );
    // The kebab trigger button renders
    expect(screen.getByRole('button')).toBeDefined();
  });

  it('calls onRefresh in full mode', () => {
    render(
      <SessionHeader
        {...defaultProps}
        session={makeSession('Running')}
        renderMode="full"
      />
    );
    fireEvent.click(screen.getByText('Refresh'));
    expect(defaultProps.onRefresh).toHaveBeenCalledTimes(1);
  });

  it('calls onStop when Stop button is clicked', () => {
    render(
      <SessionHeader
        {...defaultProps}
        session={makeSession('Running')}
        renderMode="actions-only"
      />
    );
    fireEvent.click(screen.getByText('Stop'));
    expect(defaultProps.onStop).toHaveBeenCalledTimes(1);
  });

  it('calls onContinue when Resume button is clicked', () => {
    render(
      <SessionHeader
        {...defaultProps}
        session={makeSession('Stopped')}
        renderMode="actions-only"
      />
    );
    fireEvent.click(screen.getByText('Resume'));
    expect(defaultProps.onContinue).toHaveBeenCalledTimes(1);
  });

  it('shows inactivity alert when stopped due to inactivity', () => {
    render(
      <SessionHeader
        {...defaultProps}
        session={makeSession('Stopped', 'inactivity')}
        renderMode="actions-only"
      />
    );
    expect(screen.getByText(/automatically stopped after being idle/)).toBeDefined();
  });

  it('renders kebab dropdown trigger in full mode', () => {
    render(
      <SessionHeader
        {...defaultProps}
        session={makeSession('Running')}
        renderMode="full"
      />
    );
    // Full mode renders Refresh, Stop, and kebab buttons
    const buttons = screen.getAllByRole('button');
    expect(buttons.length).toBeGreaterThanOrEqual(3);
  });
});
