/**
 * AG-UI Feedback Endpoint Proxy
 * Forwards user feedback (thumbs up/down) to backend, which sends to runner for Langfuse logging.
 * 
 * See: https://docs.ag-ui.com/drafts/meta-events#user-feedback
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

    const backendUrl = `${BACKEND_URL}/projects/${encodeURIComponent(name)}/agentic-sessions/${encodeURIComponent(sessionName)}/agui/feedback`

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
    console.error('Error submitting feedback:', error)
    return Response.json(
      { error: 'Failed to submit feedback', details: error instanceof Error ? error.message : String(error) },
      { status: 500 }
    )
  }
}
