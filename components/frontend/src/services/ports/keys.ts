import type { ProjectKey, CreateKeyRequest, CreateKeyResponse } from './types'

export type KeysPort = {
  listKeys: (projectName: string) => Promise<ProjectKey[]>
  createKey: (projectName: string, data: CreateKeyRequest) => Promise<CreateKeyResponse>
  deleteKey: (projectName: string, keyId: string) => Promise<void>
}
