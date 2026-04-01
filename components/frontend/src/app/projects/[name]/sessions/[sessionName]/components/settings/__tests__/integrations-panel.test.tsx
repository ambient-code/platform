import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { IntegrationsPanel } from '../integrations-panel';

type IntegrationsData = {
  github: { active: string | null };
  gitlab: { connected: boolean };
  jira: { connected: boolean };
  google: { connected: boolean };
  coderabbit: { connected: boolean; updatedAt: string; valid: boolean };
} | null;

const mockUseIntegrationsStatus = vi.fn((): { data: IntegrationsData; isPending: boolean } => ({
  data: null,
  isPending: false,
}));

vi.mock('@/services/queries/use-integrations', () => ({
  useIntegrationsStatus: () => mockUseIntegrationsStatus(),
}));

describe('IntegrationsPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseIntegrationsStatus.mockReturnValue({ data: null, isPending: false });
  });

  it('renders heading', () => {
    render(<IntegrationsPanel />);
    expect(screen.getByText('Integrations')).toBeDefined();
  });

  it('renders skeleton cards when loading', () => {
    mockUseIntegrationsStatus.mockReturnValue({ data: null, isPending: true });
    const { container } = render(<IntegrationsPanel />);
    const skeletons = container.querySelectorAll('[aria-hidden="true"]');
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it('renders integration cards (GitHub, GitLab, Google Workspace, Jira, CodeRabbit)', () => {
    mockUseIntegrationsStatus.mockReturnValue({
      data: {
        github: { active: null },
        gitlab: { connected: false },
        jira: { connected: false },
        google: { connected: false },
        coderabbit: { connected: true, updatedAt: '2026-04-01T00:00:00Z', valid: true },
      },
      isPending: false,
    });
    render(<IntegrationsPanel />);
    expect(screen.getByText('GitHub')).toBeDefined();
    expect(screen.getByText('GitLab')).toBeDefined();
    expect(screen.getByText('Google Workspace')).toBeDefined();
    expect(screen.getByText('Jira')).toBeDefined();
    expect(screen.getByText('CodeRabbit')).toBeDefined();
  });

  it('shows connected status for configured integrations', () => {
    mockUseIntegrationsStatus.mockReturnValue({
      data: {
        github: { active: 'some-user' },
        gitlab: { connected: true },
        jira: { connected: true },
        google: { connected: false },
        coderabbit: { connected: true, updatedAt: '2026-04-01T00:00:00Z', valid: true },
      },
      isPending: false,
    });
    render(<IntegrationsPanel />);
    // 4 out of 5 configured: badge should show 4/5
    expect(screen.getByText('4/5')).toBeDefined();
  });
});
