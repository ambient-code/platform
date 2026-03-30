import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { IntegrationsPanel } from '../integrations-panel';
import type { IntegrationsStatus } from '@/services/api/integrations';

type IntegrationsData = IntegrationsStatus | null;

function makeStatus(overrides: Partial<IntegrationsStatus> = {}): IntegrationsStatus {
  return {
    github: { installed: false, pat: { configured: false } },
    gitlab: { connected: false },
    jira: { connected: false },
    google: { connected: false },
    gerrit: { instances: [] },
    ...overrides,
  }
}

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

  it('renders integration cards (GitHub, GitLab, Google Workspace, Jira, Gerrit)', () => {
    mockUseIntegrationsStatus.mockReturnValue({
      data: makeStatus(),
      isPending: false,
    });
    render(<IntegrationsPanel />);
    expect(screen.getByText('GitHub')).toBeDefined();
    expect(screen.getByText('GitLab')).toBeDefined();
    expect(screen.getByText('Google Workspace')).toBeDefined();
    expect(screen.getByText('Jira')).toBeDefined();
    expect(screen.getByText('Gerrit')).toBeDefined();
  });

  it('shows connected status for configured integrations', () => {
    mockUseIntegrationsStatus.mockReturnValue({
      data: makeStatus({
        github: { installed: false, pat: { configured: true, valid: true }, active: 'pat' },
        gitlab: { connected: true },
        jira: { connected: true },
        google: { connected: false },
        gerrit: { instances: [] },
      }),
      isPending: false,
    });
    render(<IntegrationsPanel />);
    // 3 out of 5 configured: badge should show 3/5
    expect(screen.getByText('3/5')).toBeDefined();
  });
});
