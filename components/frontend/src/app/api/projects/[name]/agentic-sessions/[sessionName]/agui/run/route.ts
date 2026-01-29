/**
 * AG-UI Run Endpoint Proxy
 * Creates a new agent run and returns metadata immediately.
 * Events are broadcast to GET /agui/events subscribers (middleware pattern).
 * 
 * See: https://docs.ag-ui.com/concepts/architecture
 */

import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

export const runtime = 'nodejs'
export const dynamic = 'force-dynamic'

export async function POST(
  request: Request,
  { params }: { params: Promise<{ name: string; sessionName: string }> },
) {
  try {
    const { name, sessionName } = await params
    const headers = await buildForwardHeadersAsync(request)
    const body = await request.text()

    const backendUrl = `${BACKEND_URL}/projects/${encodeURIComponent(name)}/agentic-sessions/${encodeURIComponent(sessionName)}/agui/run`

    const resp = await fetch(backendUrl, {
      method: 'POST',
      headers: { 
        ...headers, 
        'Content-Type': 'application/json',
      },
      body,
    })

    // Backend returns JSON metadata immediately (not SSE stream)
    // Events are broadcast to GET /agui/events subscribers
    const data = await resp.text()
    return new Response(data, {
      status: resp.status,
      headers: { 'Content-Type': 'application/json' },
    })
  } catch (error) {
    console.error('Error starting agent run:', error)
    return Response.json(
      { error: 'Failed to start agent run', details: error instanceof Error ? error.message : String(error) },
      { status: 500 }
    )
  }
}
