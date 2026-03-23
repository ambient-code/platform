import { renderHook, act } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { useFileOperations } from '../use-file-operations';

vi.mock('sonner', () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

vi.mock('@/services/api/workspace', () => ({
  readWorkspaceFile: vi.fn().mockResolvedValue('file content'),
}));

describe('useFileOperations', () => {
  let capturedHrefs: string[] = [];
  let origCreateElement: typeof document.createElement;

  beforeEach(() => {
    capturedHrefs = [];
    origCreateElement = document.createElement.bind(document);

    vi.spyOn(document, 'createElement').mockImplementation((tag: string, options?: ElementCreationOptions) => {
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
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('handleDownloadFile URL encoding', () => {
    it('encodes path segments individually, preserving slashes', () => {
      const { result } = renderHook(() =>
        useFileOperations({
          projectName: 'my-project',
          sessionName: 'my-session',
          basePath: 'repos/my-repo',
        }),
      );

      act(() => {
        result.current.setViewingFile({ path: 'archive.zip', content: '' });
        result.current.setCurrentSubPath('sub dir');
      });

      act(() => {
        result.current.handleDownloadFile();
      });

      expect(capturedHrefs).toHaveLength(1);
      const href = capturedHrefs[0];
      // Path slashes must be preserved (not encoded as %2F)
      // "sub dir" should be encoded as "sub%20dir"
      expect(href).toContain('/workspace/repos/my-repo/sub%20dir/archive.zip');
      // Must NOT contain %2F (encoded slashes)
      const workspacePart = href.split('/workspace/')[1];
      expect(workspacePart).not.toContain('%2F');
    });

    it('encodes special characters in individual path segments', () => {
      const { result } = renderHook(() =>
        useFileOperations({
          projectName: 'my-project',
          sessionName: 'my-session',
          basePath: 'repos/my repo',
        }),
      );

      act(() => {
        result.current.setViewingFile({ path: 'my file.zip', content: '' });
      });

      act(() => {
        result.current.handleDownloadFile();
      });

      expect(capturedHrefs).toHaveLength(1);
      const href = capturedHrefs[0];
      // Spaces in segments should be encoded, but slashes preserved
      expect(href).toContain('/workspace/repos/my%20repo/my%20file.zip');
    });
  });
});
