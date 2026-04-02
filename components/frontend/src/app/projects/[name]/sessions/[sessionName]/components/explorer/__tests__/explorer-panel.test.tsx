import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ExplorerPanel } from '../explorer-panel';

vi.mock('../files-tab', () => ({
  FilesTab: () => <div data-testid="files-tab">Files Content</div>,
}));

vi.mock('../context-tab', () => ({
  ContextTab: () => <div data-testid="context-tab">Context Content</div>,
}));

vi.mock('@/services/queries/use-project-access', () => ({
  useProjectAccess: vi.fn(() => ({
    data: { userRole: 'edit', allowed: true, project: 'test-project' },
    isLoading: false,
  })),
}));

describe('ExplorerPanel', () => {
  const defaultProps = {
    visible: true,
    activeTab: 'files' as const,
    onTabChange: vi.fn(),
    onClose: vi.fn(),
    projectName: 'test-project',
    sessionName: 'test-session',
    // Files tab props
    directoryOptions: [],
    selectedDirectory: { type: 'artifacts' as const, name: 'Shared Artifacts', path: 'artifacts' },
    onDirectoryChange: vi.fn(),
    files: [],
    currentSubPath: '',
    viewingFile: null,
    isLoadingFile: false,
    onFileOrFolderSelect: vi.fn(),
    onNavigateBack: vi.fn(),
    onRefresh: vi.fn(),
    onDownloadFile: vi.fn(),
    onUploadFile: vi.fn(),
    // Context tab props
    repositories: [],
    uploadedFiles: [],
    onAddRepository: vi.fn(),
    onRemoveRepository: vi.fn(),
    onRemoveFile: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders content when visible is false (parent controls visibility via CSS)', () => {
    const { container } = render(
      <ExplorerPanel {...defaultProps} visible={false} />
    );
    expect(container.innerHTML).not.toBe('');
  });

  it('renders Files and Context tab buttons', () => {
    render(<ExplorerPanel {...defaultProps} />);
    expect(screen.getByText('Files')).toBeDefined();
    expect(screen.getByText('Context')).toBeDefined();
  });

  it('shows FilesTab when activeTab is "files"', () => {
    render(<ExplorerPanel {...defaultProps} activeTab="files" />);
    expect(screen.getByTestId('files-tab')).toBeDefined();
    expect(screen.queryByTestId('context-tab')).toBeNull();
  });

  it('shows ContextTab when activeTab is "context"', () => {
    render(<ExplorerPanel {...defaultProps} activeTab="context" />);
    expect(screen.getByTestId('context-tab')).toBeDefined();
    expect(screen.queryByTestId('files-tab')).toBeNull();
  });

  it('calls onTabChange when tab clicked', () => {
    render(<ExplorerPanel {...defaultProps} activeTab="files" />);
    fireEvent.click(screen.getByText('Context'));
    expect(defaultProps.onTabChange).toHaveBeenCalledWith('context');
  });

  it('calls onClose when close button clicked', () => {
    render(<ExplorerPanel {...defaultProps} />);
    // The close button is the one with the X icon, last button in the header
    const buttons = screen.getAllByRole('button');
    // Close button is the last button in the tab header area
    const closeButton = buttons[buttons.length - 1];
    fireEvent.click(closeButton);
    expect(defaultProps.onClose).toHaveBeenCalledTimes(1);
  });
});
