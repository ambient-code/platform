import type { Project, CreateProjectRequest, UpdateProjectRequest, PaginationParams, PaginatedResult, IntegrationStatus, MCPServersConfig } from './types'

export type ProjectsPort = {
  listProjects: (params?: PaginationParams) => Promise<PaginatedResult<Project>>
  getProject: (name: string) => Promise<Project>
  createProject: (data: CreateProjectRequest) => Promise<Project>
  updateProject: (name: string, data: UpdateProjectRequest) => Promise<Project>
  deleteProject: (name: string) => Promise<string>
  getProjectIntegrationStatus: (projectName: string) => Promise<IntegrationStatus>
  getProjectMcpServers: (projectName: string) => Promise<MCPServersConfig>
  updateProjectMcpServers: (projectName: string, config: MCPServersConfig) => Promise<MCPServersConfig>
}
