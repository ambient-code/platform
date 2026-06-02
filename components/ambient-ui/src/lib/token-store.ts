import { randomBytes } from 'crypto'

const tokens = new Map<string, string>()

export function storeToken(token: string): string {
  const id = randomBytes(16).toString('hex')
  tokens.set(id, token)
  return id
}

export function getToken(id: string): string | undefined {
  return tokens.get(id)
}

export function deleteToken(id: string): void {
  tokens.delete(id)
}
