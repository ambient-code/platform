/**
 * GitHub Releases API client
 * Uses raw fetch since this is a third-party API, not our backend
 */

export type GitHubRelease = {
  id: number;
  tag_name: string;
  name: string | null;
  body: string | null;
  html_url: string;
  published_at: string;
  prerelease: boolean;
  draft: boolean;
};

const GITHUB_RELEASES_URL =
  "https://api.github.com/repos/ambient-code/platform/releases";

export async function getGitHubReleases(): Promise<GitHubRelease[]> {
  const response = await fetch(GITHUB_RELEASES_URL, {
    headers: { Accept: "application/vnd.github+json" },
  });

  if (!response.ok) {
    throw new Error(`Failed to fetch releases: ${response.status}`);
  }

  const releases: GitHubRelease[] = await response.json();
  return releases.filter((r) => !r.draft && !r.prerelease);
}
