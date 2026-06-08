import { env } from './env'
import { getSession, getContextSession } from './session'

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
    const ctxSession = await getContextSession()
    const apiServerUrl = session.customApiServerUrl || defaultUrl
    const customToken = ctxSession.customToken ?? null
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
  const ctxSession = await getContextSession()
  if (url) session.customApiServerUrl = url
  if (token === null || token === '') {
    ctxSession.customToken = undefined
  } else if (token) {
    ctxSession.customToken = token
  }
  await session.save()
  await ctxSession.save()
}

export async function resetContext(): Promise<void> {
  const session = await getSession()
  const ctxSession = await getContextSession()
  session.customApiServerUrl = undefined
  ctxSession.customToken = undefined
  await session.save()
  await ctxSession.save()
}
