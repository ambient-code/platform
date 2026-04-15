import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ContextTab } from '../context-tab';

function renderWithProviders(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>
  );
}

describe('ContextTab', () => {
  const defaultProps = {
    repositories: [] as {
      url: string;
      name?: string;
      branch?: string;
      branches?: string[];
      currentActiveBranch?: string;
      defaultBranch?: string;
      status?: 'Cloning' | 'Ready' | 'Failed' | 'Removing';
    }[],
    uploadedFiles: [] as { name: string; path: string; size?: number }[],
    onAddRepository: vi.fn(),
    onUploadFile: vi.fn(),
    onRemoveRepository: vi.fn(),
    onRemoveFile: vi.fn(),
    canModify: true,
    projectName: 'test-project',
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders empty state when no repos or files', () => {
    renderWithProviders(<ContextTab {...defaultProps} />);
    expect(screen.getByText('No repositories added')).toBeDefined();
    expect(screen.getByText('No files uploaded')).toBeDefined();
  });

  it('renders Add button in header', () => {
    renderWithProviders(<ContextTab {...defaultProps} />);
    expect(screen.getByText('Add')).toBeDefined();
  });

  it('renders repository items', () => {
    const repos = [
      { url: 'https://github.com/org/my-repo.git', name: 'my-repo', branch: 'main' },
      { url: 'https://github.com/org/other-repo.git', name: 'other-repo', branch: 'dev' },
    ];
    renderWithProviders(<ContextTab {...defaultProps} repositories={repos} />);
    expect(screen.getByText('my-repo')).toBeDefined();
    expect(screen.getByText('other-repo')).toBeDefined();
  });

  it('renders uploaded file items', () => {
    const files = [
      { name: 'readme.txt', path: '/uploads/readme.txt', size: 1024 },
      { name: 'data.csv', path: '/uploads/data.csv', size: 2048 },
    ];
    renderWithProviders(<ContextTab {...defaultProps} uploadedFiles={files} />);
    expect(screen.getByText('readme.txt')).toBeDefined();
    expect(screen.getByText('data.csv')).toBeDefined();
  });

  it('shows repo branch badge', () => {
    const repos = [
      { url: 'https://github.com/org/repo.git', name: 'repo', branch: 'feature-branch' },
    ];
    renderWithProviders(<ContextTab {...defaultProps} repositories={repos} />);
    expect(screen.getByText('feature-branch')).toBeDefined();
  });

  it('hides Add Repository button when canModify is false', () => {
    renderWithProviders(<ContextTab {...defaultProps} canModify={false} />);
    expect(screen.queryByText('Add')).toBeNull();
    expect(screen.queryByText('Add Repository')).toBeNull();
  });

  it('hides Upload File button when canModify is false', () => {
    renderWithProviders(<ContextTab {...defaultProps} canModify={false} />);
    expect(screen.queryByText('Upload')).toBeNull();
    expect(screen.queryByText('Upload File')).toBeNull();
  });

  it('shows Add and Upload buttons when canModify is true', () => {
    renderWithProviders(<ContextTab {...defaultProps} canModify={true} />);
    expect(screen.getByText('Add')).toBeDefined();
    expect(screen.getByText('Upload')).toBeDefined();
  });
});
