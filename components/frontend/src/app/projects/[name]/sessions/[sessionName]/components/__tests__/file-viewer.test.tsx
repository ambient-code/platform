import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { FileViewer } from '../file-viewer';

vi.mock('@/services/queries/use-workspace', () => ({
  useWorkspaceFile: vi.fn(),
}));

vi.mock('sonner', () => ({
  toast: { success: vi.fn(), error: vi.fn() },
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

  it('renders and allows download for zero-byte (empty string) content', () => {
    mockUseWorkspaceFile.mockReturnValue({
      data: '',
      isLoading: false,
      error: null,
    } as ReturnType<typeof useWorkspaceFile>);

    render(<FileViewer {...defaultProps} />);

    // Should NOT show "No content available"
    expect(screen.queryByText('No content available')).toBeNull();

    // Download button should be enabled
    const downloadButtons = screen.getAllByRole('button', { name: /download/i });
    expect(downloadButtons[0].hasAttribute('disabled')).toBe(false);
  });

  describe('isActive prop controls polling', () => {
    it('polls when isActive and session is Running', () => {
      mockUseWorkspaceFile.mockReturnValue({
        data: 'hello',
        isLoading: false,
        error: null,
      } as ReturnType<typeof useWorkspaceFile>);

      render(<FileViewer {...defaultProps} sessionPhase="Running" isActive={true} />);

      const opts = mockUseWorkspaceFile.mock.calls.at(-1)?.[3];
      expect(opts?.refetchInterval).toBe(5000);
    });

    it('does not poll when isActive is false even if session is Running', () => {
      mockUseWorkspaceFile.mockReturnValue({
        data: 'hello',
        isLoading: false,
        error: null,
      } as ReturnType<typeof useWorkspaceFile>);

      render(<FileViewer {...defaultProps} sessionPhase="Running" isActive={false} />);

      const opts = mockUseWorkspaceFile.mock.calls.at(-1)?.[3];
      expect(opts?.refetchInterval).toBe(false);
    });

    it('does not poll when session is not Running regardless of isActive', () => {
      mockUseWorkspaceFile.mockReturnValue({
        data: 'hello',
        isLoading: false,
        error: null,
      } as ReturnType<typeof useWorkspaceFile>);

      render(<FileViewer {...defaultProps} sessionPhase="Completed" isActive={true} />);

      const opts = mockUseWorkspaceFile.mock.calls.at(-1)?.[3];
      expect(opts?.refetchInterval).toBe(false);
    });
  });

  describe('download uses direct link instead of triggerDownload', () => {
    it('downloads via direct workspace API link, not triggerDownload', () => {
      mockUseWorkspaceFile.mockReturnValue({
        data: 'binary-looking-content',
        isLoading: false,
        error: null,
      } as ReturnType<typeof useWorkspaceFile>);

      render(<FileViewer {...defaultProps} filePath="repos/my-repo/archive.zip" />);

      // Set up spies AFTER render so React's createElement calls are not intercepted
      const capturedHrefs: string[] = [];
      const origCreateElement = document.createElement.bind(document);
      const createSpy = vi.spyOn(document, 'createElement').mockImplementation((tag: string, options?: ElementCreationOptions) => {
        const el = origCreateElement(tag, options);
        if (tag === 'a') {
          let hrefVal = '';
          Object.defineProperty(el, 'href', {
            get: () => hrefVal,
            set: (v: string) => { hrefVal = v; capturedHrefs.push(v); },
            configurable: true,
          });
          vi.spyOn(el as HTMLAnchorElement, 'click').mockImplementation(() => {});
        }
        return el;
      });
      vi.spyOn(document.body, 'appendChild').mockImplementation((node) => node);
      vi.spyOn(document.body, 'removeChild').mockImplementation((node) => node);

      // Click the download button in the header
      const downloadButtons = screen.getAllByRole('button', { name: /download/i });
      fireEvent.click(downloadButtons[0]);

      // Must create a direct link to workspace API
      expect(capturedHrefs).toHaveLength(1);
      expect(capturedHrefs[0]).toContain('/workspace/repos/my-repo/archive.zip');

      createSpy.mockRestore();
    });
  });
});
