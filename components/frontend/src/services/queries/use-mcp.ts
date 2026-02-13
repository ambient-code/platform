import { useQuery } from "@tanstack/react-query";
import * as sessionsApi from "@/services/api/sessions";

export const mcpKeys = {
  all: ["mcp"] as const,
  status: (projectName: string, sessionName: string) =>
    [...mcpKeys.all, "status", projectName, sessionName] as const,
};

export function useMcpStatus(
  projectName: string,
  sessionName: string,
  enabled: boolean = true
) {
  return useQuery({
    queryKey: mcpKeys.status(projectName, sessionName),
    queryFn: () => sessionsApi.getMcpStatus(projectName, sessionName),
    enabled: enabled && !!projectName && !!sessionName,
    staleTime: 30 * 1000, // 30 seconds
    retry: false, // Don't retry on failure (e.g. runner not ready)
    // Backend returns 200 with empty servers when runner isn't ready, so keep polling until we have servers
    refetchInterval: (query) => {
      const servers = query.state.data?.servers
      if (servers && servers.length > 0) return false
      // Cap at ~2 min (12 × 10s) so we don't poll forever if session has no MCP servers
      const updatedCount = (query.state as { dataUpdatedCount?: number }).dataUpdatedCount ?? 0
      if (updatedCount >= 12) return false
      return 10 * 1000
    },
  });
}

