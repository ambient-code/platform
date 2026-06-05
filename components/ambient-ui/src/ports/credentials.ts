import type {
  DomainCredential,
  DomainCredentialCreateRequest,
  DomainCredentialUpdateRequest,
  ListParams,
  PaginatedResult,
} from '@/domain/types'

export type CredentialsPort = {
  list: (params?: ListParams) => Promise<PaginatedResult<DomainCredential>>
  get: (id: string) => Promise<DomainCredential>
  create: (request: DomainCredentialCreateRequest) => Promise<DomainCredential>
  update: (id: string, request: DomainCredentialUpdateRequest) => Promise<DomainCredential>
  delete: (id: string) => Promise<void>
}
