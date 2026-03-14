import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { FileSelectionSummary } from '../file-selection-summary';

function makeFile(overrides: Record<string, unknown> = {}) {
  return {
    id: 'f1',
    name: 'document.pdf',
    mimeType: 'application/pdf',
    sizeBytes: null as number | null,
    isFolder: false,
    status: 'active',
    ...overrides,
  };
}

describe('FileSelectionSummary', () => {
  it('renders "No files selected." when files array is empty', () => {
    render(<FileSelectionSummary files={[]} />);
    expect(screen.getByText('No files selected.')).toBeDefined();
  });

  it('renders file count badge', () => {
    const files = [
      makeFile({ id: 'f1', name: 'a.pdf' }),
      makeFile({ id: 'f2', name: 'b.pdf' }),
      makeFile({ id: 'f3', name: 'c.pdf' }),
    ];
    render(<FileSelectionSummary files={files} />);
    expect(screen.getByText('3 files')).toBeDefined();
  });

  it('renders singular "file" for single file', () => {
    render(<FileSelectionSummary files={[makeFile()]} />);
    expect(screen.getByText('1 file')).toBeDefined();
  });

  it('renders file names', () => {
    const files = [
      makeFile({ id: 'f1', name: 'report.docx' }),
      makeFile({ id: 'f2', name: 'data.csv' }),
    ];
    render(<FileSelectionSummary files={files} />);
    expect(screen.getByText('report.docx')).toBeDefined();
    expect(screen.getByText('data.csv')).toBeDefined();
  });

  it('shows "Unavailable" badge for files with status "unavailable"', () => {
    const files = [makeFile({ id: 'f1', name: 'gone.pdf', status: 'unavailable' })];
    render(<FileSelectionSummary files={files} />);
    expect(screen.getByText('Unavailable')).toBeDefined();
  });

  it('does not show "Unavailable" badge for active files', () => {
    const files = [makeFile({ id: 'f1', name: 'ok.pdf', status: 'active' })];
    render(<FileSelectionSummary files={files} />);
    expect(screen.queryByText('Unavailable')).toBeNull();
  });

  it('renders folder icon for folders', () => {
    const files = [makeFile({ id: 'f1', name: 'My Folder', isFolder: true, mimeType: 'application/vnd.google-apps.folder' })];
    const { container } = render(<FileSelectionSummary files={files} />);
    // Folder icon gets blue-500 class
    const icon = container.querySelector('.text-blue-500');
    expect(icon).toBeDefined();
    expect(icon).not.toBeNull();
  });

  it('renders spreadsheet icon for spreadsheet mime type', () => {
    const files = [makeFile({ id: 'f1', name: 'sheet.xlsx', mimeType: 'application/vnd.google-apps.spreadsheet' })];
    const { container } = render(<FileSelectionSummary files={files} />);
    const icon = container.querySelector('.text-green-600');
    expect(icon).toBeDefined();
    expect(icon).not.toBeNull();
  });

  it('renders image icon for image mime type', () => {
    const files = [makeFile({ id: 'f1', name: 'photo.png', mimeType: 'image/png' })];
    const { container } = render(<FileSelectionSummary files={files} />);
    const icon = container.querySelector('.text-purple-500');
    expect(icon).toBeDefined();
    expect(icon).not.toBeNull();
  });

  it('renders document icon for document mime type', () => {
    const files = [makeFile({ id: 'f1', name: 'doc.docx', mimeType: 'application/vnd.google-apps.document' })];
    const { container } = render(<FileSelectionSummary files={files} />);
    const icon = container.querySelector('.text-blue-600');
    expect(icon).toBeDefined();
    expect(icon).not.toBeNull();
  });

  it('shows file size formatted correctly', () => {
    const files = [makeFile({ id: 'f1', name: 'big.zip', sizeBytes: 1536 })];
    render(<FileSelectionSummary files={files} />);
    expect(screen.getByText('1.5 KB')).toBeDefined();
  });

  it('shows MB for larger files', () => {
    const files = [makeFile({ id: 'f1', name: 'huge.zip', sizeBytes: 2621440 })];
    render(<FileSelectionSummary files={files} />);
    expect(screen.getByText('2.5 MB')).toBeDefined();
  });

  it('does not show size when sizeBytes is null', () => {
    const files = [makeFile({ id: 'f1', name: 'nosize.pdf', sizeBytes: null })];
    render(<FileSelectionSummary files={files} />);
    // Should not render any size text
    expect(screen.queryByText(/KB|MB|GB|B$/)).toBeNull();
  });

  it('renders custom title and description', () => {
    const files = [makeFile()];
    render(
      <FileSelectionSummary
        files={files}
        title="Granted Files"
        description="Files shared with this project"
      />,
    );
    expect(screen.getByText('Granted Files')).toBeDefined();
    expect(screen.getByText('Files shared with this project')).toBeDefined();
  });

  it('renders default title when no title prop is provided', () => {
    const files = [makeFile()];
    render(<FileSelectionSummary files={files} />);
    expect(screen.getByText('Selected Files')).toBeDefined();
  });
});
