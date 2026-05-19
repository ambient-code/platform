import * as keysApi from '../../api/keys'
import type { KeysPort } from '../../ports/keys'

type KeysApi = typeof keysApi

export function createKeysAdapter(api: KeysApi): KeysPort {
  return {
    listKeys: api.listKeys,
    createKey: api.createKey,
    deleteKey: api.deleteKey,
  }
}

export const keysAdapter = createKeysAdapter(keysApi)
