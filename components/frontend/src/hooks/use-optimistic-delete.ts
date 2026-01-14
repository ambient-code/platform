/**
 * Reusable hook for non-blocking deletion operations with visual feedback.
 *
 * Provides consistent UX across all delete operations:
 * - Immediate dialog close (non-blocking)
 * - "Deleting..." toast notification
 * - Deleting state tracking (for badges/spinners)
 * - Background mutation execution
 * - Success/error toast notifications
 * - Automatic state cleanup
 *
 * @example
 * const { confirmDelete, isDeleting } = useOptimisticDelete({
 *   getId: (project) => project.name,
 *   getDisplayName: (project) => project.displayName || project.name,
 *   mutation: deleteProjectMutation,
 * });
 *
 * // In table row
 * {isDeleting(project) && <Badge>Deleting...</Badge>}
 * <Button onClick={() => confirmDelete(project)} disabled={isDeleting(project)}>
 *   Delete
 * </Button>
 */

import { useState } from 'react';
import { UseMutationResult } from '@tanstack/react-query';
import { toast, successToast, errorToast } from '@/hooks/use-toast';

type UseOptimisticDeleteOptions<T, TMutationVars = string> = {
  /**
   * Extract unique ID from item (used for tracking deletion state)
   * @example (project) => project.name
   */
  getId: (item: T) => string;

  /**
   * Extract display name from item (used in toast messages)
   * @example (project) => project.displayName || project.name
   */
  getDisplayName: (item: T) => string;

  /**
   * React Query mutation hook for delete operation
   * @example useDeleteProject()
   */
  mutation: UseMutationResult<unknown, Error, TMutationVars>;

  /**
   * Function to convert item to mutation variables
   * @example (project) => project.name
   * @example (key, projectName) => ({ projectName, keyId: key.id })
   */
  getMutationVariables: (item: T, ...args: any[]) => TMutationVars;

  /**
   * Optional: Custom deleting message
   * @default "Deleting..."
   */
  deletingMessage?: string;

  /**
   * Optional: Custom success message template
   * @default (displayName) => `"${displayName}" deleted successfully`
   */
  successMessage?: (displayName: string) => string;

  /**
   * Optional: Custom error message template
   * @default (displayName, error) => error?.message || `Failed to delete "${displayName}"`
   */
  errorMessage?: (displayName: string, error?: Error) => string;

  /**
   * Optional: Additional callback to run on successful deletion
   * @example (item, id) => { if (pathname.includes(id)) router.push('/') }
   */
  onSuccess?: (item: T, id: string) => void;

  /**
   * Optional: Additional callback to run on deletion error
   */
  onError?: (item: T, id: string, error: Error) => void;
};

export function useOptimisticDelete<T, TMutationVars = string>({
  getId,
  getDisplayName,
  mutation,
  getMutationVariables,
  deletingMessage = 'Deleting...',
  successMessage = (displayName) => `"${displayName}" deleted successfully`,
  errorMessage = (displayName, error) =>
    error?.message || `Failed to delete "${displayName}"`,
  onSuccess,
  onError,
}: UseOptimisticDeleteOptions<T, TMutationVars>) {
  const [deletingId, setDeletingId] = useState<string | null>(null);

  /**
   * Execute deletion with non-blocking UX pattern
   */
  const confirmDelete = (item: T, ...args: any[]) => {
    const id = getId(item);
    const displayName = getDisplayName(item);
    const mutationVars = getMutationVariables(item, ...args);

    // Set deleting state (for visual feedback)
    setDeletingId(id);

    // Show immediate "deleting" toast
    toast({
      title: deletingMessage,
      description: `Removing "${displayName}"`,
    });

    // Fire mutation in background (non-blocking)
    mutation.mutate(mutationVars, {
      onSuccess: () => {
        successToast(successMessage(displayName));
        setDeletingId(null); // Clear deleting state
        onSuccess?.(item, id); // Call custom success callback if provided
      },
      onError: (error) => {
        errorToast(errorMessage(displayName, error as Error));
        setDeletingId(null); // Clear deleting state on error
        onError?.(item, id, error as Error); // Call custom error callback if provided
      },
    });
  };

  /**
   * Check if specific item is currently being deleted
   */
  const isDeleting = (item: T) => deletingId === getId(item);

  return {
    confirmDelete,
    isDeleting,
    deletingId, // Exposed for advanced use cases
  };
}
