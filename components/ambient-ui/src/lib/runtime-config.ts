import { env } from './env'
import { getSession } from './session'

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
    const customToken = session.customToken || null
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
    session.customToken = undefined
  } else if (token) {
    session.customToken = token
  }
  await session.save()
}

export async function resetContext(): Promise<void> {
  const session = await getSession()
  session.customApiServerUrl = undefined
  session.customToken = undefined
  await session.save()
}
