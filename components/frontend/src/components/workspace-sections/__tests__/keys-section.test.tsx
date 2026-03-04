import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { KeysSection } from '../keys-section';

const mockCreateMutate = vi.fn();
const mockDeleteMutate = vi.fn();
const mockRefetch = vi.fn();

vi.mock('@/services/queries', () => ({
  useKeys: vi.fn(() => ({
    data: [
      {
        id: 'key-1',
        name: 'ci-key',
        description: 'CI pipeline key',
        createdAt: '2025-01-01T00:00:00Z',
        lastUsedAt: null,
        role: 'edit',
      },
      {
        id: 'key-2',
        name: 'admin-key',
        description: '',
        createdAt: '2025-02-01T00:00:00Z',
        lastUsedAt: '2025-03-01T00:00:00Z',
        role: 'admin',
      },
    ],
    isLoading: false,
    error: null,
    refetch: mockRefetch,
  })),
  useCreateKey: vi.fn(() => ({
    mutate: mockCreateMutate,
    isPending: false,
    isError: false,
    error: null,
  })),
  useDeleteKey: vi.fn(() => ({
    mutate: mockDeleteMutate,
    isPending: false,
    isError: false,
    error: null,
    variables: null,
  })),
}));

vi.mock('@/hooks/use-toast', () => ({
  successToast: vi.fn(),
  errorToast: vi.fn(),
}));

vi.mock('@/components/error-message', () => ({
  ErrorMessage: ({ error }: { error: Error }) => <div data-testid="error">{error.message}</div>,
}));

vi.mock('@/components/empty-state', () => ({
  EmptyState: ({ title }: { title: string }) => <div data-testid="empty-state">{title}</div>,
}));

vi.mock('@/components/confirmation-dialog', () => ({
  DestructiveConfirmationDialog: ({
    open,
    onConfirm,
    title,
    description,
  }: {
    open: boolean;
    onConfirm: () => void;
    title: string;
    description: string;
  }) =>
    open ? (
      <div data-testid="delete-dialog">
        <span>{title}</span>
        <span>{description}</span>
        <button onClick={onConfirm}>Confirm Delete</button>
      </div>
    ) : null,
}));

describe('KeysSection', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders key table with mock data', () => {
    render(<KeysSection projectName="test-project" />);
    expect(screen.getByText('ci-key')).toBeDefined();
    expect(screen.getByText('admin-key')).toBeDefined();
    expect(screen.getByText('CI pipeline key')).toBeDefined();
    expect(screen.getByText('Access Keys (2)')).toBeDefined();
  });

  it('shows role badges', () => {
    render(<KeysSection projectName="test-project" />);
    expect(screen.getByText('Edit')).toBeDefined();
    expect(screen.getByText('Admin')).toBeDefined();
  });

  it('opens Create Key dialog', () => {
    render(<KeysSection projectName="test-project" />);
    fireEvent.click(screen.getByText('Create Key'));
    expect(screen.getByText('Create Access Key')).toBeDefined();
    expect(screen.getByLabelText('Name *')).toBeDefined();
  });

  it('submits create key form', async () => {
    render(<KeysSection projectName="test-project" />);
    fireEvent.click(screen.getByText('Create Key'));

    const nameInput = screen.getByLabelText('Name *');
    fireEvent.change(nameInput, { target: { value: 'new-key' } });

    const descInput = screen.getByLabelText('Description');
    fireEvent.change(descInput, { target: { value: 'New key description' } });

    // The "Create Key" button inside the dialog
    const createButtons = screen.getAllByText('Create Key');
    const dialogCreateBtn = createButtons[createButtons.length - 1];
    fireEvent.click(dialogCreateBtn);

    expect(mockCreateMutate).toHaveBeenCalledWith(
      expect.objectContaining({
        projectName: 'test-project',
        data: expect.objectContaining({
          name: 'new-key',
          description: 'New key description',
          role: 'edit',
        }),
      }),
      expect.any(Object)
    );
  });

  it('opens delete confirmation for a key', () => {
    render(<KeysSection projectName="test-project" />);

//     // Click the first delete button (trash icon buttons in the table)
//     screen.getAllByRole('button').filter((btn) => {
//       return btn.querySelector('svg');
//     });
    // Find buttons within table rows (the trash buttons)
    const trashButtons = screen.getAllByRole('row').slice(1).map((row) => {
      return row.querySelector('button');
    }).filter(Boolean);

    if (trashButtons[0]) {
      fireEvent.click(trashButtons[0]);
    }

    expect(screen.getByTestId('delete-dialog')).toBeDefined();
    expect(screen.getByText('Delete Access Key')).toBeDefined();
  });

  it('calls delete mutation on confirm', async () => {
    render(<KeysSection projectName="test-project" />);

    // Open delete dialog for first key
    const rows = screen.getAllByRole('row').slice(1);
    const firstRowBtn = rows[0]?.querySelector('button');
    if (firstRowBtn) fireEvent.click(firstRowBtn);

    fireEvent.click(screen.getByText('Confirm Delete'));
    expect(mockDeleteMutate).toHaveBeenCalledWith(
      expect.objectContaining({
        projectName: 'test-project',
        keyId: 'key-1',
      }),
      expect.any(Object)
    );
  });
});
