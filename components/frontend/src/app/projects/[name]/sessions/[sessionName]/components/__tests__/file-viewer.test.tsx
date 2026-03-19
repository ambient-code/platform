import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { FileViewer } from '../file-viewer';

vi.mock('@/services/queries/use-workspace', () => ({
  useWorkspaceFile: vi.fn(),
}));

import { useWorkspaceFile } from '@/services/queries/use-workspace';

const mockUseWorkspaceFile = vi.mocked(useWorkspaceFile);

const defaultProps = {
  projectName: 'my-project',
  sessionName: 'my-session',
  filePath: 'src/index.ts',
};

describe('FileViewer', () => {
  it('renders loading skeleton when isLoading', () => {
    mockUseWorkspaceFile.mockReturnValue({
      data: undefined,
      isLoading: true,
      error: null,
    } as ReturnType<typeof useWorkspaceFile>);

    const { container } = render(<FileViewer {...defaultProps} />);

    // Skeleton elements should be present (the component renders multiple Skeleton divs)
    const skeletons = container.querySelectorAll('[class*="animate-pulse"], [data-slot="skeleton"]');
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it('renders error state when error', () => {
    mockUseWorkspaceFile.mockReturnValue({
      data: undefined,
      isLoading: false,
      error: new Error('File not found'),
    } as ReturnType<typeof useWorkspaceFile>);

    render(<FileViewer {...defaultProps} />);

    expect(screen.getByText('Failed to load file')).toBeDefined();
    expect(screen.getByText('File not found')).toBeDefined();
  });

  it('renders file content using FileContentViewer', () => {
    const content = 'const a = 1;\nconst b = 2;\nconst c = 3;';
    mockUseWorkspaceFile.mockReturnValue({
      data: content,
      isLoading: false,
      error: null,
    } as ReturnType<typeof useWorkspaceFile>);

    const { container } = render(<FileViewer {...defaultProps} />);

    // Verify the file content is rendered inside <code>
    const codeElement = container.querySelector('code');
    expect(codeElement?.textContent).toBe(content);
  });

  it('renders file path in header', () => {
    mockUseWorkspaceFile.mockReturnValue({
      data: 'const x = 1;',
      isLoading: false,
      error: null,
    } as ReturnType<typeof useWorkspaceFile>);

    render(<FileViewer {...defaultProps} filePath="src/app.tsx" />);

    expect(screen.getByText('src/app.tsx')).toBeDefined();
  });

  it('renders download button', () => {
    mockUseWorkspaceFile.mockReturnValue({
      data: 'hello',
      isLoading: false,
      error: null,
    } as ReturnType<typeof useWorkspaceFile>);

    render(<FileViewer {...defaultProps} />);

    // Should have at least one download button
    const downloadButtons = screen.getAllByRole('button', { name: /download/i });
    expect(downloadButtons.length).toBeGreaterThan(0);
  });

  it('renders no content state when content is undefined', () => {
    mockUseWorkspaceFile.mockReturnValue({
      data: undefined,
      isLoading: false,
      error: null,
    } as ReturnType<typeof useWorkspaceFile>);

    render(<FileViewer {...defaultProps} />);

    expect(screen.getByText('No content available')).toBeDefined();
  });
});
