/**
 * Corrections Endpoint Proxy
 * Forwards user corrections to backend for Langfuse persistence and optional runner forwarding.
 */

import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

export const runtime = 'nodejs'
export const dynamic = 'force-dynamic'

export async function POST(
  request: Request,
  { params }: { params: Promise<{ name: string }> },
) {
  try {
    const { name } = await params
    const headers = await buildForwardHeadersAsync(request)
    const body = await request.text()

    const backendUrl = `${BACKEND_URL}/projects/${encodeURIComponent(name)}/corrections`

    const resp = await fetch(backendUrl, {
      method: 'POST',
      headers: {
        ...headers,
        'Content-Type': 'application/json',
      },
      body,
    })

    const data = await resp.text()
    return new Response(data, {
      status: resp.status,
      headers: { 'Content-Type': 'application/json' },
    })
  } catch (error) {
    console.error('Error submitting correction:', error)
    return Response.json(
      { error: 'Failed to submit correction', details: error instanceof Error ? error.message : String(error) },
      { status: 500 }
    )
  }
}
