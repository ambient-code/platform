import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { FileContentViewer } from '../file-content-viewer';

describe('FileContentViewer', () => {
  describe('file size display', () => {
    it('uses fileSize prop when provided instead of computing from content', () => {
      // Content is 5 bytes, but real file is 10240 bytes (10 KB)
      render(
        <FileContentViewer
          fileName="archive.zip"
          content="hello"
          fileSize={10240}
        />,
      );

      // Should display the real size (10.0 KB), not the content size (5 B)
      expect(screen.getByText(/10\.0 KB/)).toBeDefined();
    });

    it('falls back to computing size from content when fileSize not provided', () => {
      render(
        <FileContentViewer
          fileName="readme.txt"
          content="hello world"
        />,
      );

      // "hello world" = 11 bytes = "11.0 B"
      expect(screen.getByText(/11\.0 B/)).toBeDefined();
    });
  });
});
