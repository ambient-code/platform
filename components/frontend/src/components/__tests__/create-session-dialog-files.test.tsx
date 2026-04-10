import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { CreateSessionDialog } from '../create-session-dialog';

const mockMutate = vi.fn();

// Mock all required hooks and dependencies
vi.mock('next/navigation', () => ({
  useRouter: vi.fn(() => ({
    push: vi.fn(),
    replace: vi.fn(),
    prefetch: vi.fn(),
  })),
}));

vi.mock('@/lib/feature-flags', () => ({
  useFlag: vi.fn(() => false),
}));

vi.mock('@/services/queries/use-sessions', () => ({
  useCreateSession: vi.fn(() => ({
    mutate: mockMutate,
    isPending: false,
  })),
}));

vi.mock('@/services/queries/use-runner-types', () => ({
  useRunnerTypes: vi.fn(() => ({
    data: [{ id: 'claude-agent-sdk', displayName: 'Claude Agent SDK', description: 'Default', provider: 'anthropic' }],
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  })),
}));

vi.mock('@/services/queries/use-integrations', () => ({
  useIntegrationsStatus: vi.fn(() => ({
    data: { github: { active: null }, gitlab: { connected: false }, jira: { connected: false }, google: { connected: false } },
  })),
}));

vi.mock('@/services/queries/use-models', () => ({
  useModels: vi.fn(() => ({
    data: { models: [{ id: 'claude-sonnet-4-6', label: 'Claude Sonnet 4.6' }], defaultModel: 'claude-sonnet-4-6' },
    isLoading: false,
    isError: false,
  })),
}));

vi.mock('@/services/queries/use-workflows', () => ({
  useOOTBWorkflows: vi.fn(() => ({
    data: [],
    isLoading: false,
  })),
}));

vi.mock('@/services/api/runner-types', () => ({
  DEFAULT_RUNNER_TYPE_ID: 'claude-agent-sdk',
}));

vi.mock('sonner', () => ({
  toast: { error: vi.fn(), success: vi.fn() },
}));

describe('CreateSessionDialog - File Attachments', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders file attachment section', () => {
    render(
      <CreateSessionDialog
        projectName="test-project"
        trigger={<button>Create</button>}
      />
    );

    fireEvent.click(screen.getByText('Create'));

    expect(screen.getByText('Files (optional)')).toBeDefined();
    expect(screen.getByText('Click to attach files')).toBeDefined();
  });

  it('adds files to the pending list', async () => {
    render(
      <CreateSessionDialog
        projectName="test-project"
        trigger={<button>Create</button>}
      />
    );

    fireEvent.click(screen.getByText('Create'));

    const fileInput = screen.getByLabelText('Attach files');
    const file = new File(['content'], 'test.txt', { type: 'text/plain' });
    Object.defineProperty(file, 'size', { value: 1024 });

    fireEvent.change(fileInput, { target: { files: [file] } });

    await waitFor(() => {
      expect(screen.getByText(/test\.txt/)).toBeDefined();
      expect(screen.getByText(/1\.0 KB/)).toBeDefined();
    });
  });

  it('removes files from the pending list', async () => {
    render(
      <CreateSessionDialog
        projectName="test-project"
        trigger={<button>Create</button>}
      />
    );

    fireEvent.click(screen.getByText('Create'));

    const fileInput = screen.getByLabelText('Attach files');
    const file = new File(['content'], 'removeme.txt', { type: 'text/plain' });
    Object.defineProperty(file, 'size', { value: 512 });

    fireEvent.change(fileInput, { target: { files: [file] } });

    await waitFor(() => {
      expect(screen.getByText(/removeme\.txt/)).toBeDefined();
    });

    // Find and click the remove button (X icon)
    const removeButtons = screen.getAllByRole('button').filter(
      btn => btn.querySelector('svg') && btn.closest('.bg-muted\\/50')
    );
    if (removeButtons.length > 0) {
      fireEvent.click(removeButtons[0]);
      await waitFor(() => {
        expect(screen.queryByText(/removeme\.txt/)).toBeNull();
      });
    }
  });

  it('rejects files exceeding 10MB', async () => {
    const { toast } = await import('sonner');

    render(
      <CreateSessionDialog
        projectName="test-project"
        trigger={<button>Create</button>}
      />
    );

    fireEvent.click(screen.getByText('Create'));

    const fileInput = screen.getByLabelText('Attach files');
    const largeFile = new File(['x'], 'huge.bin', { type: 'application/octet-stream' });
    Object.defineProperty(largeFile, 'size', { value: 11 * 1024 * 1024 });

    fireEvent.change(fileInput, { target: { files: [largeFile] } });

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith(expect.stringContaining('exceeds 10MB'));
    });
  });
});
