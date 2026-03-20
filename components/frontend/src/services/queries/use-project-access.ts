/**
 * React Query hook for checking user's access level to a project
 */

import { useQuery } from '@tanstack/react-query';
import type { PermissionRole } from '@/types/project';

export type ProjectAccess = {
  project: string;
  allowed: boolean;
  reason?: string;
  userRole: PermissionRole; // "view" | "edit" | "admin"
};

/**
 * Fetch the current user's access level for a project.
 *
 * The backend performs a SelfSubjectAccessReview to determine the user's role:
 * - "admin": Can create RoleBindings (full workspace access)
 * - "edit": Can create AgenticSessions (can interact with sessions)
 * - "view": Can only list/get resources (read-only)
 *
 * @param projectName The project/namespace name
 * @returns Query result with userRole field
 */
export function useProjectAccess(projectName: string) {
  return useQuery<ProjectAccess>({
    queryKey: ['project-access', projectName],
    queryFn: async () => {
      const res = await fetch(`/api/projects/${projectName}/access`);
      if (!res.ok) {
        throw new Error('Failed to fetch project access');
      }
      return res.json();
    },
    enabled: !!projectName,
    staleTime: 60000, // Cache for 1 minute
    retry: 1, // Retry once on failure
  });
}
