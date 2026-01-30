interface RepoInfo {
  type: "github" | "gitlab" | "unknown";
  org: string;
  repo: string;
  host?: string; // For GitLab: 'gitlab.redhat.com'
}

/**
 * Parse Git URL to extract repository information
 * Handles: HTTPS, SSH, with/without .git suffix
 */
export function parseGitUrl(url: string): RepoInfo | null {
  if (!url) return null;

  // Normalize SSH to HTTPS
  let normalizedUrl = url;
  if (url.startsWith("git@")) {
    normalizedUrl = url.replace("git@", "https://").replace(":", "/");
  }

  // Remove .git suffix
  normalizedUrl = normalizedUrl.replace(/\.git$/, "");

  try {
    const parsed = new URL(normalizedUrl);
    const pathParts = parsed.pathname.split("/").filter(Boolean);

    if (pathParts.length < 2) return null;

    // GitHub detection
    if (parsed.hostname === "github.com") {
      return {
        type: "github",
        org: pathParts[0],
        repo: pathParts[1],
      };
    }

    // GitLab detection (any host with 'gitlab' in name)
    if (parsed.hostname.includes("gitlab")) {
      return {
        type: "gitlab",
        org: pathParts[0], // namespace
        repo: pathParts[1], // project
        host: parsed.hostname,
      };
    }

    return { type: "unknown", org: "", repo: "" };
  } catch {
    return null;
  }
}

/**
 * Generate IDE URL for opening file/folder in web IDE
 */
export function getIdeUrl(
  repoUrl: string,
  branch: string,
  path: string,
  gitlabInstance?: string
): string | null {
  const repo = parseGitUrl(repoUrl);
  if (!repo || repo.type === "unknown") return null;

  // Clean path (remove leading slash if present)
  const cleanPath = path.startsWith("/") ? path.slice(1) : path;

  if (repo.type === "github") {
    // GitHub.dev pattern: https://github.dev/org/repo/blob/branch/path
    return `https://github.dev/${repo.org}/${repo.repo}/blob/${branch}/${cleanPath}`;
  }

  if (repo.type === "gitlab") {
    // GitLab Web IDE pattern: https://[instance]/-/ide/project/[namespace]/[project]/edit/[branch]/-/[path]
    const instance = repo.host || gitlabInstance || "gitlab.com";
    return `https://${instance}/-/ide/project/${repo.org}/${repo.repo}/edit/${branch}/-/${cleanPath}`;
  }

  return null;
}

/**
 * Get IDE display name for user-facing UI
 */
export function getIdeName(repoUrl: string): string {
  const repo = parseGitUrl(repoUrl);
  if (repo?.type === "github") return "GitHub.dev";
  if (repo?.type === "gitlab") return "GitLab Web IDE";
  return "Web IDE";
}
