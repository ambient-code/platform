import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { SessionDetailsModal } from '../session-details-modal';
import type { AgenticSession } from '@/types/agentic-session';

const mockUseProject = vi.fn(() => ({ data: undefined }));
vi.mock('@/services/queries', () => ({
  useProject: () => mockUseProject(),
}));

vi.mock('@/services/queries/use-sessions', () => ({
  useSessionExport: () => ({ data: null, isLoading: false }),
}));

vi.mock('@/utils/session-helpers', () => ({
  getPhaseColor: () => '',
}));

vi.mock('@/utils/export-chat', () => ({
  triggerDownload: vi.fn(),
}));

const mockSession: AgenticSession = {
  metadata: {
    name: 'session-abc',
    namespace: 'default',
    creationTimestamp: '2026-01-01T00:00:00Z',
    uid: 'uid-abc',
  },
  spec: {
    llmSettings: { model: 'claude-3', temperature: 0, maxTokens: 4096 },
    timeout: 3600,
  },
};

describe('SessionDetailsModal — workspace name', () => {
  const defaultProps = {
    session: mockSession,
    projectName: 'my-project',
    open: true,
    onOpenChange: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows displayName when available', () => {
    mockUseProject.mockReturnValue({ data: { displayName: 'My Workspace Display', name: 'my-project' } });
    render(<SessionDetailsModal {...defaultProps} />);
    expect(screen.getByText('Workspace:')).toBeDefined();
    expect(screen.getByText('My Workspace Display')).toBeDefined();
  });

  it('falls back to raw project name when displayName is empty', () => {
    mockUseProject.mockReturnValue({ data: { displayName: '', name: 'my-project' } });
    render(<SessionDetailsModal {...defaultProps} />);
    expect(screen.getByText('Workspace:')).toBeDefined();
    expect(screen.getByText('my-project')).toBeDefined();
  });

  it('falls back to raw project name when project data is not loaded', () => {
    mockUseProject.mockReturnValue({ data: undefined });
    render(<SessionDetailsModal {...defaultProps} />);
    expect(screen.getByText('Workspace:')).toBeDefined();
    expect(screen.getByText('my-project')).toBeDefined();
  });
});
