import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { ContextTab } from '../context-tab';

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
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders empty state when no repos or files', () => {
    render(<ContextTab {...defaultProps} />);
    expect(screen.getByText('No repositories added')).toBeDefined();
    expect(screen.getByText('No files uploaded')).toBeDefined();
  });

  it('renders Add button in header', () => {
    render(<ContextTab {...defaultProps} />);
    expect(screen.getByText('Add')).toBeDefined();
  });

  it('renders repository items', () => {
    const repos = [
      { url: 'https://github.com/org/my-repo.git', name: 'my-repo', branch: 'main' },
      { url: 'https://github.com/org/other-repo.git', name: 'other-repo', branch: 'dev' },
    ];
    render(<ContextTab {...defaultProps} repositories={repos} />);
    expect(screen.getByText('my-repo')).toBeDefined();
    expect(screen.getByText('other-repo')).toBeDefined();
  });

  it('renders uploaded file items', () => {
    const files = [
      { name: 'readme.txt', path: '/uploads/readme.txt', size: 1024 },
      { name: 'data.csv', path: '/uploads/data.csv', size: 2048 },
    ];
    render(<ContextTab {...defaultProps} uploadedFiles={files} />);
    expect(screen.getByText('readme.txt')).toBeDefined();
    expect(screen.getByText('data.csv')).toBeDefined();
  });

  it('shows repo branch badge', () => {
    const repos = [
      { url: 'https://github.com/org/repo.git', name: 'repo', branch: 'feature-branch' },
    ];
    render(<ContextTab {...defaultProps} repositories={repos} />);
    expect(screen.getByText('feature-branch')).toBeDefined();
  });
});
