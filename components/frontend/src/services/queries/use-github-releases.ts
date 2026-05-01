/**
 * React Query hook for GitHub releases
 */

import { useQuery } from "@tanstack/react-query";
import { getGitHubReleases } from "../api/github-releases";

export const githubReleasesKeys = {
  all: ["github-releases"] as const,
  list: () => [...githubReleasesKeys.all, "list"] as const,
};

export function useGitHubReleases() {
  return useQuery({
    queryKey: githubReleasesKeys.list(),
    queryFn: getGitHubReleases,
    staleTime: 15 * 60 * 1000, // 15 minutes — conserve GitHub's 60 req/hr limit
    retry: 1,
    refetchOnWindowFocus: false,
  });
}
