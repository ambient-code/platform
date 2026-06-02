import { env } from './env'
import { getSession } from './session'
import { storeToken, getToken, deleteToken } from './token-store'

export type RuntimeConfig = {
  apiServerUrl: string
  customToken: string | null
  defaultApiServerUrl: string
  isCustomContext: boolean
}

export async function getRuntimeConfig(): Promise<RuntimeConfig> {
  const defaultUrl = env.API_SERVER_URL
  try {
    const session = await getSession()
    const apiServerUrl = session.customApiServerUrl || defaultUrl
    const customToken = session.customTokenId ? (getToken(session.customTokenId) ?? null) : null
    return {
      apiServerUrl,
      customToken,
      defaultApiServerUrl: defaultUrl,
      isCustomContext: apiServerUrl !== defaultUrl || customToken !== null,
    }
  } catch {
    return {
      apiServerUrl: defaultUrl,
      customToken: null,
      defaultApiServerUrl: defaultUrl,
      isCustomContext: false,
    }
  }
}

export async function setCustomContext(url?: string, token?: string | null): Promise<void> {
  const session = await getSession()
  if (url) session.customApiServerUrl = url
  if (token === null || token === '') {
    if (session.customTokenId) {
      deleteToken(session.customTokenId)
      session.customTokenId = undefined
    }
  } else if (token) {
    if (session.customTokenId) {
      deleteToken(session.customTokenId)
    }
    session.customTokenId = storeToken(token)
  }
  await session.save()
}

export async function resetContext(): Promise<void> {
  const session = await getSession()
  if (session.customTokenId) {
    deleteToken(session.customTokenId)
  }
  session.customApiServerUrl = undefined
  session.customTokenId = undefined
  await session.save()
}
