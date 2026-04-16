/**
 * Parse a GitHub repo URL or owner/repo shorthand into owner and repo components.
 */
export function parseOwnerRepo(
  repoUrl: string
): { owner: string; repo: string } | null {
  const cleaned = repoUrl.replace(/\.git$/, "").replace(/\/$/, "");
  const httpsMatch = cleaned.match(/github\.com\/([^/]+)\/([^/]+)/);
  if (httpsMatch) {
    return { owner: httpsMatch[1], repo: httpsMatch[2] };
  }
  const parts = cleaned.split("/");
  if (parts.length === 2 && parts[0] && parts[1]) {
    return { owner: parts[0], repo: parts[1] };
  }
  return null;
}
