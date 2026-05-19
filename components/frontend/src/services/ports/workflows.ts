import type { OOTBWorkflow, WorkflowMetadataResponse } from './types'

export type WorkflowsPort = {
  listOOTBWorkflows: (projectName?: string) => Promise<OOTBWorkflow[]>
  getWorkflowMetadata: (projectName: string, sessionName: string) => Promise<WorkflowMetadataResponse>
}
