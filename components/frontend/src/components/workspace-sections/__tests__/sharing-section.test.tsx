import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { SharingSection } from '../sharing-section';

const mockAddMutate = vi.fn();
const mockRemoveMutate = vi.fn();
const mockRefetch = vi.fn();

vi.mock('@/services/queries', () => ({
  useProjectPermissions: vi.fn(() => ({
    data: [
      { subjectType: 'group', subjectName: 'developers', role: 'edit' },
      { subjectType: 'user', subjectName: 'alice', role: 'admin' },
    ],
    isLoading: false,
    refetch: mockRefetch,
  })),
  useAddProjectPermission: vi.fn(() => ({
    mutate: mockAddMutate,
    isPending: false,
  })),
  useRemoveProjectPermission: vi.fn(() => ({
    mutate: mockRemoveMutate,
    isPending: false,
    variables: null,
  })),
}));

vi.mock('@/hooks/use-toast', () => ({
  successToast: vi.fn(),
  errorToast: vi.fn(),
}));

vi.mock('@/components/confirmation-dialog', () => ({
  DestructiveConfirmationDialog: ({
    open,
    onConfirm,
    title,
  }: {
    open: boolean;
    onConfirm: () => void;
    title: string;
  }) =>
    open ? (
      <div data-testid="revoke-dialog">
        <span>{title}</span>
        <button onClick={onConfirm}>Confirm Revoke</button>
      </div>
    ) : null,
}));

describe('SharingSection', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders permissions table with data', () => {
    render(<SharingSection projectName="test-project" />);
    expect(screen.getByText('Sharing')).toBeDefined();
    expect(screen.getByText('developers')).toBeDefined();
    expect(screen.getByText('alice')).toBeDefined();
  });

  it('shows role badges for permissions', () => {
    render(<SharingSection projectName="test-project" />);
    expect(screen.getByText('Edit')).toBeDefined();
    expect(screen.getByText('Admin')).toBeDefined();
  });

  it('shows subject type labels', () => {
    render(<SharingSection projectName="test-project" />);
    expect(screen.getByText('Group')).toBeDefined();
    expect(screen.getByText('User')).toBeDefined();
  });

  it('opens Grant Permission dialog', () => {
    render(<SharingSection projectName="test-project" />);
    fireEvent.click(screen.getByText('Grant Permission'));
    expect(screen.getByText('Add a user or group to this workspace with a role')).toBeDefined();
  });

  it('submits grant permission form', () => {
    render(<SharingSection projectName="test-project" />);
    fireEvent.click(screen.getByText('Grant Permission'));

    const nameInput = screen.getByPlaceholderText('Enter group name');
    fireEvent.change(nameInput, { target: { value: 'new-team' } });

    // Click the Grant Permission button in the dialog footer
    const grantButtons = screen.getAllByText('Grant Permission');
    const dialogBtn = grantButtons[grantButtons.length - 1];
    fireEvent.click(dialogBtn);

    expect(mockAddMutate).toHaveBeenCalledWith(
      expect.objectContaining({
        projectName: 'test-project',
        permission: expect.objectContaining({
          subjectType: 'group',
          subjectName: 'new-team',
          role: 'view',
        }),
      }),
      expect.any(Object)
    );
  });

  it('opens revoke confirmation dialog', () => {
    render(<SharingSection projectName="test-project" />);

    // Click the trash button for the first permission
    const rows = screen.getAllByRole('row').slice(1);
    const firstRowBtn = rows[0]?.querySelector('button');
    if (firstRowBtn) fireEvent.click(firstRowBtn);

    expect(screen.getByTestId('revoke-dialog')).toBeDefined();
    expect(screen.getByText('Revoke Permission')).toBeDefined();
  });

  it('calls revoke mutation on confirm', () => {
    render(<SharingSection projectName="test-project" />);

    // Open revoke dialog
    const rows = screen.getAllByRole('row').slice(1);
    const firstRowBtn = rows[0]?.querySelector('button');
    if (firstRowBtn) fireEvent.click(firstRowBtn);

    fireEvent.click(screen.getByText('Confirm Revoke'));
    expect(mockRemoveMutate).toHaveBeenCalledWith(
      expect.objectContaining({
        projectName: 'test-project',
        subjectType: 'group',
        subjectName: 'developers',
      }),
      expect.any(Object)
    );
  });

  it('shows Grant First Permission in empty state', async () => {
    const queries = await import('@/services/queries');
    const mockedPermissions = vi.mocked(queries.useProjectPermissions);
    mockedPermissions.mockReturnValue({
      data: [],
      isLoading: false,
      refetch: mockRefetch,
    } as ReturnType<typeof queries.useProjectPermissions>);

    render(<SharingSection projectName="test-project" />);
    expect(screen.getByText('No users or groups have access yet')).toBeDefined();
  });
});
