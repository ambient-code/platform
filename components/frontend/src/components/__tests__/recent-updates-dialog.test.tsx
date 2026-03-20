import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { RecentUpdatesDialog } from '../recent-updates-dialog';
import type { GitHubRelease } from '@/services/api/github-releases';

const mockReleases: GitHubRelease[] = [
  {
    id: 1,
    tag_name: 'v1.2.0',
    name: 'Release 1.2.0',
    body: '## What\'s New\n\n- Feature A\n- Feature B',
    html_url: 'https://github.com/ambient-code/platform/releases/tag/v1.2.0',
    published_at: '2026-03-18T12:00:00Z',
    prerelease: false,
    draft: false,
  },
  {
    id: 2,
    tag_name: 'v1.1.0',
    name: 'Release 1.1.0',
    body: 'Bug fixes and improvements',
    html_url: 'https://github.com/ambient-code/platform/releases/tag/v1.1.0',
    published_at: '2026-03-10T12:00:00Z',
    prerelease: false,
    draft: false,
  },
];

const mockRefetch = vi.fn();
const mockUseGitHubReleases = vi.fn();
const mockUseLocalStorage = vi.fn();

vi.mock('@/services/queries/use-github-releases', () => ({
  useGitHubReleases: () => mockUseGitHubReleases(),
}));

vi.mock('@/hooks/use-local-storage', () => ({
  useLocalStorage: (...args: unknown[]) => mockUseLocalStorage(...args),
}));

describe('RecentUpdatesDialog', () => {
  const mockSetLastSeen = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    mockUseGitHubReleases.mockReturnValue({
      data: mockReleases,
      isLoading: false,
      isError: false,
      refetch: mockRefetch,
    });
    mockUseLocalStorage.mockReturnValue([null, mockSetLastSeen, vi.fn()]);
  });

  it('renders gift icon button', () => {
    render(<RecentUpdatesDialog />);
    const button = screen.getByRole('button', { name: /recent updates/i });
    expect(button).toBeDefined();
  });

  it('shows red dot when unseen releases exist', () => {
    render(<RecentUpdatesDialog />);
    const button = screen.getByRole('button', { name: /recent updates/i });
    const dot = button.querySelector('.bg-red-500');
    expect(dot).not.toBeNull();
  });

  it('hides red dot when all releases are seen', () => {
    mockUseLocalStorage.mockReturnValue([
      '2026-03-19T00:00:00Z',
      mockSetLastSeen,
      vi.fn(),
    ]);
    render(<RecentUpdatesDialog />);
    const button = screen.getByRole('button', { name: /recent updates/i });
    const dot = button.querySelector('.bg-red-500');
    expect(dot).toBeNull();
  });

  it('opens dialog on click with release content', async () => {
    render(<RecentUpdatesDialog />);
    fireEvent.click(screen.getByRole('button', { name: /recent updates/i }));
    await waitFor(() => {
      expect(screen.getByText('Recent Updates')).toBeDefined();
      expect(screen.getByText('Release 1.2.0')).toBeDefined();
      expect(screen.getByText('v1.2.0')).toBeDefined();
      expect(screen.getByText('Release 1.1.0')).toBeDefined();
    });
  });

  it('marks updates as seen when dialog opens', () => {
    render(<RecentUpdatesDialog />);
    fireEvent.click(screen.getByRole('button', { name: /recent updates/i }));
    expect(mockSetLastSeen).toHaveBeenCalledWith(expect.any(String));
  });

  it('shows loading state', () => {
    mockUseGitHubReleases.mockReturnValue({
      data: undefined,
      isLoading: true,
      isError: false,
      refetch: mockRefetch,
    });
    render(<RecentUpdatesDialog />);
    fireEvent.click(screen.getByRole('button', { name: /recent updates/i }));
    // Loader2 renders as an SVG with animate-spin
    const dialog = screen.getByRole('dialog');
    const spinner = dialog.querySelector('.animate-spin');
    expect(spinner).not.toBeNull();
  });

  it('shows error state with retry button', () => {
    mockUseGitHubReleases.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
      refetch: mockRefetch,
    });
    render(<RecentUpdatesDialog />);
    fireEvent.click(screen.getByRole('button', { name: /recent updates/i }));
    expect(screen.getByText('Failed to load updates.')).toBeDefined();
    fireEvent.click(screen.getByText('Retry'));
    expect(mockRefetch).toHaveBeenCalled();
  });

  it('shows empty state', () => {
    mockUseGitHubReleases.mockReturnValue({
      data: [],
      isLoading: false,
      isError: false,
      refetch: mockRefetch,
    });
    render(<RecentUpdatesDialog />);
    fireEvent.click(screen.getByRole('button', { name: /recent updates/i }));
    expect(screen.getByText('No updates available.')).toBeDefined();
  });

  it('renders markdown in release body', async () => {
    render(<RecentUpdatesDialog />);
    fireEvent.click(screen.getByRole('button', { name: /recent updates/i }));
    await waitFor(() => {
      // The markdown heading "What's New" should be rendered
      expect(screen.getByText("What's New")).toBeDefined();
      expect(screen.getByText('Feature A')).toBeDefined();
    });
  });
});
