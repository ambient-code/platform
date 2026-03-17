import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { SessionsSidebar } from '../sessions-sidebar';
import type { AgenticSession } from '@/types/api';

const mockPush = vi.fn();
vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: mockPush }),
  usePathname: () => '/projects/test-project/sessions/session-0',
}));

vi.mock('next/link', () => ({
  default: ({ children, href }: { children: React.ReactNode; href: string }) => (
    <a href={href}>{children}</a>
  ),
}));

const mockUseSessionsPaginated = vi.fn((): { data: { items: Partial<AgenticSession>[] } | null; isLoading: boolean } => ({
  data: { items: [] },
  isLoading: false,
}));
vi.mock('@/services/queries/use-sessions', () => ({
  useSessionsPaginated: () => mockUseSessionsPaginated(),
}));

vi.mock('@/services/queries/use-version', () => ({
  useVersion: () => ({ data: '1.0.0' }),
}));

vi.mock('@/components/session-status-dot', () => ({
  SessionStatusDot: ({ phase }: { phase: string }) => (
    <span data-testid="status-dot">{phase}</span>
  ),
  sessionPhaseLabel: (phase: string) => phase || 'Unknown',
}));

vi.mock('date-fns', () => ({
  formatDistanceToNow: () => '2 hours',
}));

function makeSessions(count: number) {
  return Array.from({ length: count }, (_, i) => ({
    metadata: {
      name: `session-${i}`,
      namespace: 'default',
      uid: `uid-${i}`,
      creationTimestamp: '2026-01-01T00:00:00Z',
    },
    spec: {
      displayName: `Session ${i}`,
      initialPrompt: 'test',
      llmSettings: { model: 'test', temperature: 0, maxTokens: 100 },
      timeout: 3600,
    },
    status: { phase: 'Running' as const },
  })) as unknown as AgenticSession[];
}

describe('SessionsSidebar', () => {
  const defaultProps = {
    projectName: 'test-project',
    currentSessionName: 'session-0',
    collapsed: false,
  };

  beforeEach(() => {
    vi.clearAllMocks();
    mockUseSessionsPaginated.mockReturnValue({
      data: { items: [] },
      isLoading: false,
    });
  });

  it('returns null when collapsed is true', () => {
    const { container } = render(
      <SessionsSidebar {...defaultProps} collapsed={true} />
    );
    expect(container.innerHTML).toBe('');
  });

  it('renders New Session button', () => {
    render(<SessionsSidebar {...defaultProps} />);
    expect(screen.getByText('New Session')).toBeDefined();
  });

  it('renders Workspaces back link', () => {
    render(<SessionsSidebar {...defaultProps} />);
    expect(screen.getByText('Workspaces')).toBeDefined();
  });

  it('renders workspace navigation links', () => {
    render(<SessionsSidebar {...defaultProps} />);
    expect(screen.getByText('Sessions')).toBeDefined();
    expect(screen.getByText('Schedules')).toBeDefined();
    expect(screen.getByText('Sharing')).toBeDefined();
    expect(screen.getByText('Access Keys')).toBeDefined();
    expect(screen.getByText('Workspace Settings')).toBeDefined();
  });

  it('renders RECENTS section header', () => {
    render(<SessionsSidebar {...defaultProps} />);
    expect(screen.getByText('Recents')).toBeDefined();
  });

  it('renders loading skeletons when isLoading', () => {
    mockUseSessionsPaginated.mockReturnValue({
      data: { items: [] },
      isLoading: true,
    });
    const { container } = render(<SessionsSidebar {...defaultProps} />);
    const skeletons = container.querySelectorAll('[class*="animate-pulse"], [data-slot="skeleton"]');
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it('renders "No sessions yet" when no sessions', () => {
    render(<SessionsSidebar {...defaultProps} />);
    expect(screen.getByText('No sessions yet')).toBeDefined();
  });

  it('renders session items when data exists', () => {
    mockUseSessionsPaginated.mockReturnValue({
      data: { items: makeSessions(3) },
      isLoading: false,
    });
    render(<SessionsSidebar {...defaultProps} />);
    expect(screen.getByText('Session 0')).toBeDefined();
    expect(screen.getByText('Session 1')).toBeDefined();
    expect(screen.getByText('Session 2')).toBeDefined();
  });

  it('clicking session navigates to correct URL', () => {
    mockUseSessionsPaginated.mockReturnValue({
      data: { items: makeSessions(2) },
      isLoading: false,
    });
    render(<SessionsSidebar {...defaultProps} />);
    fireEvent.click(screen.getByText('Session 1'));
    expect(mockPush).toHaveBeenCalledWith(
      '/projects/test-project/sessions/session-1'
    );
  });

  it('calls onSessionSelect when clicking a session', () => {
    const onSessionSelect = vi.fn();
    mockUseSessionsPaginated.mockReturnValue({
      data: { items: makeSessions(1) },
      isLoading: false,
    });
    render(<SessionsSidebar {...defaultProps} onSessionSelect={onSessionSelect} />);
    fireEvent.click(screen.getByText('Session 0'));
    expect(onSessionSelect).toHaveBeenCalled();
  });

  it('clicking New Session calls onNewSession callback', () => {
    const onNewSession = vi.fn();
    render(<SessionsSidebar {...defaultProps} onNewSession={onNewSession} />);
    fireEvent.click(screen.getByText('New Session'));
    expect(onNewSession).toHaveBeenCalled();
  });

  it('clicking New Session navigates to new page when no callback', () => {
    render(<SessionsSidebar {...defaultProps} />);
    fireEvent.click(screen.getByText('New Session'));
    expect(mockPush).toHaveBeenCalledWith('/projects/test-project/new');
  });

  it('renders collapse button when onCollapse is provided', () => {
    const onCollapse = vi.fn();
    render(<SessionsSidebar {...defaultProps} onCollapse={onCollapse} />);
    const collapseBtn = screen.getByTitle('Hide sidebar');
    expect(collapseBtn).toBeDefined();
    fireEvent.click(collapseBtn);
    expect(onCollapse).toHaveBeenCalled();
  });
});
