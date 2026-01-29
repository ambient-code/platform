/**
 * AG-UI History Endpoint Proxy
 * Returns compacted message history for a session.
 * 
 * See: https://docs.ag-ui.com/concepts/serialization
 */

import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string; sessionName: string }> },
) {
  const { name, sessionName } = await params
  const url = new URL(request.url)
  const runId = url.searchParams.get('runId') || ''
  const headers = await buildForwardHeadersAsync(request)

  let backendUrl = `${BACKEND_URL}/projects/${encodeURIComponent(name)}/agentic-sessions/${encodeURIComponent(sessionName)}/agui/history`
  if (runId) {
    backendUrl += `?runId=${encodeURIComponent(runId)}`
  }

  const resp = await fetch(backendUrl, {
    method: 'GET',
    headers,
  })

  const data = await resp.text()
  return new Response(data, {
    status: resp.status,
    headers: { 'Content-Type': 'application/json' },
  })
}

