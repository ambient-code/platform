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
    retry: false, // Don't retry if runner isn't available
  });
}

