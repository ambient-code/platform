import { useQuery } from '@tanstack/react-query';
import { workflowsAdapter } from '../adapters/workflows';
import type { WorkflowsPort } from '../ports/workflows';
import { BACKEND_VERSION } from './query-keys';

export const workflowKeys = {
  all: [BACKEND_VERSION, 'workflows'] as const,
  ootb: (projectName?: string) => [...workflowKeys.all, 'ootb', projectName] as const,
  metadata: (projectName: string, sessionName: string) =>
    [...workflowKeys.all, 'metadata', projectName, sessionName] as const,
};

export function useOOTBWorkflows(projectName?: string, port: WorkflowsPort = workflowsAdapter) {
  return useQuery({
    queryKey: workflowKeys.ootb(projectName),
    queryFn: async () => {
      const workflows = await port.listOOTBWorkflows(projectName);
      return workflows;
    },
    enabled: !!projectName,
    staleTime: 5 * 60 * 1000,
  });
}

export function useWorkflowMetadata(
  projectName: string,
  sessionName: string,
  enabled: boolean,
  port: WorkflowsPort = workflowsAdapter,
) {
  return useQuery({
    queryKey: workflowKeys.metadata(projectName, sessionName),
    queryFn: () => port.getWorkflowMetadata(projectName, sessionName),
    enabled: enabled && !!projectName && !!sessionName,
    staleTime: 60 * 1000,
  });
}
