import type { SessionRepo } from '@/types/agentic-session';

/**
 * Default branch name used when not specified
 */
export const DEFAULT_BRANCH = 'main';

/**
 * Extract a display-friendly name from a SessionRepo.
 * Uses repo.name if available, otherwise derives from repo.input.url.
 *
 * @param repo - The repository configuration
 * @param fallbackIndex - Index to use if name cannot be derived
 * @returns Display name for the repository
 */
export function getRepoDisplayName(repo: SessionRepo, fallbackIndex: number): string {
  if (repo.name) {
    return repo.name;
  }

  try {
    // Extract repository name from URL (e.g., "https://github.com/org/repo.git" -> "repo")
    const match = repo.input.url.match(/\/([^/]+?)(\.git)?$/);
    return match ? match[1] : `repo-${fallbackIndex}`;
  } catch {
    return `repo-${fallbackIndex}`;
  }
}

/**
 * Check if a repository has a valid output configuration that differs from input.
 * Used to determine if push functionality should be available.
 *
 * @param repo - The repository configuration
 * @returns True if repo has a different output URL or branch
 */
export function hasValidOutputConfig(repo: SessionRepo): boolean {
  if (!repo.output?.url) {
    return false;
  }

  // Output is valid if URL is different OR branch is different
  return (
    repo.output.url !== repo.input.url ||
    (repo.output.branch || DEFAULT_BRANCH) !== (repo.input.branch || DEFAULT_BRANCH)
  );
}
