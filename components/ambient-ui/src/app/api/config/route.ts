import { getRuntimeConfig, setCustomContext, resetContext } from '@/lib/runtime-config'

export const runtime = 'nodejs'
export const dynamic = 'force-dynamic'

export async function GET() {
  const config = await getRuntimeConfig()
  return Response.json({
    apiServerUrl: config.apiServerUrl,
    customToken: config.customToken !== null,
    isCustomContext: config.isCustomContext,
    defaultApiServerUrl: config.defaultApiServerUrl,
  })
}

export async function PUT(request: Request) {
  let body: unknown
  try {
    body = await request.json()
  } catch {
    return Response.json({ error: 'Invalid JSON' }, { status: 400 })
  }

  if (typeof body !== 'object' || body === null) {
    return Response.json(
      { error: 'Request body must be a JSON object' },
      { status: 400 },
    )
  }

  const parsed = body as Record<string, unknown>
  const url = parsed.apiServerUrl
  const token = parsed.customToken

  if (url !== undefined && typeof url !== 'string') {
    return Response.json(
      { error: 'apiServerUrl must be a string' },
      { status: 400 },
    )
  }

  if (token !== undefined && token !== null && typeof token !== 'string') {
    return Response.json(
      { error: 'customToken must be a string or null' },
      { status: 400 },
    )
  }

  if (typeof url === 'string' && !url.startsWith('http://') && !url.startsWith('https://')) {
    return Response.json(
      { error: 'apiServerUrl must start with http:// or https://' },
      { status: 400 },
    )
  }

  if (url === undefined && token === undefined) {
    return Response.json(
      { error: 'Request body must include apiServerUrl or customToken' },
      { status: 400 },
    )
  }

  await setCustomContext(
    typeof url === 'string' ? url : undefined,
    token === null ? null : typeof token === 'string' ? (token || null) : undefined,
  )

  const config = await getRuntimeConfig()
  return Response.json({
    apiServerUrl: config.apiServerUrl,
    customToken: config.customToken !== null,
    isCustomContext: config.isCustomContext,
    defaultApiServerUrl: config.defaultApiServerUrl,
  })
}

export async function DELETE() {
  await resetContext()
  const config = await getRuntimeConfig()
  return Response.json({
    apiServerUrl: config.apiServerUrl,
    customToken: config.customToken !== null,
    isCustomContext: config.isCustomContext,
    defaultApiServerUrl: config.defaultApiServerUrl,
  })
}
