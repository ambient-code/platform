/**
 * MCP Invoke API Route
 * POST /api/projects/:name/agentic-sessions/:sessionName/mcp/invoke
 * Proxies to backend which proxies to runner to invoke an MCP tool
 */

import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

export const dynamic = 'force-dynamic'

export async function POST(
  request: Request,
  { params }: { params: Promise<{ name: string; sessionName: string }> },
) {
  const { name, sessionName } = await params

  const headers = await buildForwardHeadersAsync(request, {
    'Content-Type': 'application/json',
  })

  const body = await request.text()

  const backendUrl = `${BACKEND_URL}/projects/${encodeURIComponent(name)}/agentic-sessions/${encodeURIComponent(sessionName)}/mcp/invoke`

  try {
    const response = await fetch(backendUrl, {
      method: 'POST',
      headers,
      body,
    })

    if (!response.ok) {
      const errorText = await response.text()
      // Preserve structured JSON errors from backend; wrap plain text
      let errorBody: string
      try {
        const parsed = JSON.parse(errorText)
        errorBody = JSON.stringify(parsed)
      } catch {
        errorBody = JSON.stringify({ error: errorText || `HTTP ${response.status}` })
      }
      return new Response(errorBody, {
        status: response.status,
        headers: { 'Content-Type': 'application/json' },
      })
    }

    const data = await response.json()
    return Response.json(data)
  } catch (error) {
    console.error('MCP invoke proxy error:', error)
    return new Response(
      JSON.stringify({
        error: error instanceof Error ? error.message : 'Failed to invoke MCP tool',
      }),
      { status: 500, headers: { 'Content-Type': 'application/json' } }
    )
  }
}
