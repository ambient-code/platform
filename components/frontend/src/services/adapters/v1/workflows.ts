import * as workflowsApi from '../../api/workflows'
import type { WorkflowsPort } from '../../ports/workflows'

type WorkflowsApi = typeof workflowsApi

export function createWorkflowsAdapter(api: WorkflowsApi): WorkflowsPort {
  return {
    listOOTBWorkflows: api.listOOTBWorkflows,
    getWorkflowMetadata: api.getWorkflowMetadata,
    getWorkflowSources: api.getWorkflowSources,
    updateWorkflowSources: api.updateWorkflowSources,
  }
}

export const workflowsAdapter = createWorkflowsAdapter(workflowsApi)
