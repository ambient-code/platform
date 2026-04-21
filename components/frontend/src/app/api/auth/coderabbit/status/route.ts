import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

export async function GET(request: Request) {
  try {
    const headers = await buildForwardHeadersAsync(request)

    const resp = await fetch(`${BACKEND_URL}/auth/coderabbit/status`, {
      method: 'GET',
      headers,
    })

    const data = await resp.text()
    return new Response(data, {
      status: resp.status,
      headers: { 'Content-Type': resp.headers.get('content-type') ?? 'application/json' },
    })
  } catch {
    return Response.json({ error: 'CodeRabbit upstream unavailable' }, { status: 502 })
  }
}
