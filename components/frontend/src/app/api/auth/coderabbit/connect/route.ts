import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

export async function POST(request: Request) {
  try {
    const headers = await buildForwardHeadersAsync(request)
    const body = await request.text()

    const resp = await fetch(`${BACKEND_URL}/auth/coderabbit/connect`, {
      method: 'POST',
      headers,
      body,
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
