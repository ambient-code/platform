/**
 * MCP Status API Route
 * GET /api/projects/:name/agentic-sessions/:sessionName/mcp/status
 * Proxies to backend which proxies to runner
 */

import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

export const dynamic = 'force-dynamic'

export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string; sessionName: string }> },
) {
  const { name, sessionName } = await params

  // Build auth headers from the incoming request
  const headers = await buildForwardHeadersAsync(request)

  // Build backend URL
  const backendUrl = `${BACKEND_URL}/projects/${encodeURIComponent(name)}/agentic-sessions/${encodeURIComponent(sessionName)}/mcp/status`

  try {
    const response = await fetch(backendUrl, {
      method: 'GET',
      headers,
    })

    if (!response.ok) {
      const errorText = await response.text()
      return new Response(JSON.stringify({ error: errorText }), {
        status: response.status,
        headers: { 'Content-Type': 'application/json' },
      })
    }

    const data = await response.json()
    return new Response(JSON.stringify(data), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    })
  } catch (error) {
    console.error('MCP status proxy error:', error)
    return new Response(
      JSON.stringify({
        error: error instanceof Error ? error.message : 'Failed to fetch MCP status',
        servers: [],
        totalCount: 0,
      }),
      { status: 500, headers: { 'Content-Type': 'application/json' } }
    )
  }
}

