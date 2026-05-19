import type { PermissionAssignment, ProjectAccess } from './types'

export type ProjectAccessPort = {
  getAccess: (projectName: string) => Promise<ProjectAccess>
  getPermissions: (projectName: string) => Promise<PermissionAssignment[]>
  addPermission: (projectName: string, permission: PermissionAssignment) => Promise<PermissionAssignment>
  removePermission: (projectName: string, subjectType: string, subjectName: string) => Promise<void>
}
