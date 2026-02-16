import { useQuery } from "@tanstack/react-query";
import * as workflowsApi from "@/services/api/workflows";

export const workflowKeys = {
  all: ["workflows"] as const,
  ootb: (projectName?: string) => [...workflowKeys.all, "ootb", projectName] as const,
  metadata: (projectName: string, sessionName: string) =>
    [...workflowKeys.all, "metadata", projectName, sessionName] as const,
};

export function useOOTBWorkflows(projectName?: string, options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: workflowKeys.ootb(projectName),
    queryFn: async () => {
      const workflows = await workflowsApi.listOOTBWorkflows(projectName);
      return workflows;
    },
    enabled: (options?.enabled ?? true) && !!projectName,
    staleTime: 5 * 60 * 1000, // 5 minutes - workflows don't change often
  });
}

export function useWorkflowMetadata(
  projectName: string,
  sessionName: string,
  enabled: boolean,
  options?: { enabled?: boolean },
) {
  return useQuery({
    queryKey: workflowKeys.metadata(projectName, sessionName),
    queryFn: () => workflowsApi.getWorkflowMetadata(projectName, sessionName),
    enabled: (options?.enabled ?? true) && enabled && !!projectName && !!sessionName,
    staleTime: 60 * 1000, // 1 minute
  });
}

