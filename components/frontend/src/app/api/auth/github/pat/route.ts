import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

export async function POST(request: Request) {
  const headers = await buildForwardHeadersAsync(request)
  const body = await request.text()

  const resp = await fetch(`${BACKEND_URL}/auth/github/pat`, {
    method: 'POST',
    headers,
    body,
  })

  const data = await resp.text()
  return new Response(data, { status: resp.status, headers: { 'Content-Type': 'application/json' } })
}

export async function DELETE(request: Request) {
  const headers = await buildForwardHeadersAsync(request)

  const resp = await fetch(`${BACKEND_URL}/auth/github/pat`, {
    method: 'DELETE',
    headers,
  })

  const data = await resp.text()
  return new Response(data, { status: resp.status, headers: { 'Content-Type': 'application/json' } })
}
