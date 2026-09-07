import type { OOTBWorkflow, WorkflowMetadataResponse, WorkflowSourcesConfig } from './types'

export type WorkflowsPort = {
  listOOTBWorkflows: (projectName?: string) => Promise<OOTBWorkflow[]>
  getWorkflowMetadata: (projectName: string, sessionName: string) => Promise<WorkflowMetadataResponse>
  getWorkflowSources: (projectName: string) => Promise<WorkflowSourcesConfig>
  updateWorkflowSources: (projectName: string, config: WorkflowSourcesConfig) => Promise<WorkflowSourcesConfig>
}
