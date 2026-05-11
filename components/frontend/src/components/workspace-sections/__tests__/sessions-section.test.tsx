import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { SessionsSection } from '../sessions-section';

vi.mock('next/link', () => ({
  default: ({ children, href }: { children: React.ReactNode; href: string }) => (
    <a href={href}>{children}</a>
  ),
}));

const mockUseProject = vi.fn(() => ({ data: undefined }));
vi.mock('@/services/queries', () => ({
  useSessionsPaginated: () => ({ data: { items: [], totalCount: 0, hasMore: false }, isFetching: false, refetch: vi.fn() }),
  useStopSession: () => ({ mutate: vi.fn(), isPending: false }),
  useDeleteSession: () => ({ mutate: vi.fn(), isPending: false }),
  useContinueSession: () => ({ mutate: vi.fn(), isPending: false }),
  useUpdateSessionDisplayName: () => ({ mutate: vi.fn(), isPending: false }),
  useRunnerTypes: () => ({ data: [] }),
  useProject: () => mockUseProject(),
}));

vi.mock('@/services/queries/use-project-access', () => ({
  useProjectAccess: () => ({ data: { userRole: 'admin' } }),
}));

vi.mock('@/services/queries/use-workspace', () => ({
  useWorkspaceList: () => ({ data: [], isLoading: false }),
}));

vi.mock('@/hooks/use-debounce', () => ({
  useDebounce: (value: string) => value,
}));

vi.mock('@/components/empty-state', () => ({
  EmptyState: ({ title }: { title: string }) => <div>{title}</div>,
}));

vi.mock('@/components/session-status-dot', () => ({
  SessionStatusDot: ({ phase }: { phase: string }) => <span>{phase}</span>,
}));

vi.mock('@/components/agent-status-indicator', () => ({
  AgentStatusIndicator: () => <span>status</span>,
}));

vi.mock('@/hooks/use-agent-status', () => ({
  deriveAgentStatusFromPhase: () => 'idle',
}));

vi.mock('@/components/edit-session-name-dialog', () => ({
  EditSessionNameDialog: () => null,
}));

vi.mock('@/lib/pagination', () => ({
  getPageNumbers: () => [],
}));

vi.mock('date-fns', () => ({
  formatDistanceToNow: () => '2 hours',
}));

describe('SessionsSection — workspace name', () => {
  const defaultProps = { projectName: 'my-project' };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows displayName in header when available', () => {
    mockUseProject.mockReturnValue({ data: { displayName: 'My Project Workspace', name: 'my-project' } });
    render(<SessionsSection {...defaultProps} />);
    expect(screen.getByText(/My Project Workspace/)).toBeDefined();
  });

  it('falls back to raw project name when displayName is empty', () => {
    mockUseProject.mockReturnValue({ data: { displayName: '', name: 'my-project' } });
    render(<SessionsSection {...defaultProps} />);
    expect(screen.getByText(/Workspace:.*my-project/)).toBeDefined();
  });

  it('falls back to raw project name when project data is not loaded', () => {
    mockUseProject.mockReturnValue({ data: undefined });
    render(<SessionsSection {...defaultProps} />);
    expect(screen.getByText(/Workspace:.*my-project/)).toBeDefined();
  });
});
