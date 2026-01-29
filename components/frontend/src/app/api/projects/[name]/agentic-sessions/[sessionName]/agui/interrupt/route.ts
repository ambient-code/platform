/**
 * AG-UI Interrupt Endpoint Proxy
 * Forwards interrupt signal to backend to stop Claude SDK execution.
 * 
 * See: https://platform.claude.com/docs/en/agent-sdk/python#methods
 */

import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

export const runtime = 'nodejs'
export const dynamic = 'force-dynamic'

export async function POST(
  request: Request,
  { params }: { params: Promise<{ name: string; sessionName: string }> },
) {
  const { name, sessionName } = await params
  const headers = await buildForwardHeadersAsync(request)
  const body = await request.text()

  const backendUrl = `${BACKEND_URL}/projects/${encodeURIComponent(name)}/agentic-sessions/${encodeURIComponent(sessionName)}/agui/interrupt`

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
}

