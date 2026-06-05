import type {
  DomainRoleBinding,
  DomainRoleBindingCreateRequest,
  ListParams,
  PaginatedResult,
} from '@/domain/types'

export type RoleBindingsPort = {
  list: (params?: ListParams) => Promise<PaginatedResult<DomainRoleBinding>>
  create: (request: DomainRoleBindingCreateRequest) => Promise<DomainRoleBinding>
  delete: (id: string) => Promise<void>
}
