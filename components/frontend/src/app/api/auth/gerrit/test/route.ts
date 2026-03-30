import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

export async function POST(request: Request) {
  const headers = await buildForwardHeadersAsync(request)
  const body = await request.text()

  try {
    const resp = await fetch(`${BACKEND_URL}/auth/gerrit/test`, {
      method: 'POST',
      headers,
      body,
      signal: AbortSignal.timeout(15_000),
    })

    const data = await resp.text()
    return new Response(data, { status: resp.status, headers: { 'Content-Type': 'application/json' } })
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Backend request failed'
    return new Response(JSON.stringify({ valid: false, error: message }), {
      status: 502,
      headers: { 'Content-Type': 'application/json' },
    })
  }
}
