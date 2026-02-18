/**
 * AG-UI SSE Events Proxy
 * Proxies the backend AG-UI SSE stream through Next.js for Bearer auth compatibility.
 * 
 * Browser EventSource cannot set Authorization headers, so we proxy through
 * the Next.js same-origin API to inject auth headers server-side.
 * 
 * See: https://docs.ag-ui.com/quickstart/introduction
 */

import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

export const runtime = 'nodejs'
export const dynamic = 'force-dynamic'

export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string; sessionName: string }> },
) {
  const { name, sessionName } = await params
  const url = new URL(request.url)
  const runId = url.searchParams.get('runId') || ''

  // Build auth headers from the incoming request
  const headers = await buildForwardHeadersAsync(request)

  // Remove Content-Type as we're making a GET request for SSE
  delete headers['Content-Type']

  // Build backend URL
  let backendUrl = `${BACKEND_URL}/projects/${encodeURIComponent(name)}/agentic-sessions/${encodeURIComponent(sessionName)}/agui/events`
  if (runId) {
    backendUrl += `?runId=${encodeURIComponent(runId)}`
  }

  try {
    // Fetch from backend SSE endpoint
    const response = await fetch(backendUrl, {
      method: 'GET',
      headers: {
        ...headers,
        Accept: 'text/event-stream',
        'Cache-Control': 'no-cache',
      },
      // @ts-expect-error - Node.js fetch supports duplex for streaming
      duplex: 'half',
    })

    if (!response.ok) {
      const errorText = await response.text()
      return new Response(JSON.stringify({ error: errorText }), {
        status: response.status,
        headers: { 'Content-Type': 'application/json' },
      })
    }

    // Pipe the SSE stream through
    const { readable, writable } = new TransformStream()
    
    // Forward the body in a non-blocking way
    if (response.body) {
      response.body.pipeTo(writable).catch((err) => {
        // ResponseAborted is normal when client disconnects, don't log as error
        if (err?.name !== 'AbortError' && !err?.message?.includes('ResponseAborted')) {
          console.error('AG-UI SSE proxy pipe error:', err)
        }
      })
    }

    return new Response(readable, {
      status: 200,
      headers: {
        'Content-Type': 'text/event-stream',
        'Cache-Control': 'no-cache, no-store, must-revalidate',
        Connection: 'keep-alive',
        'X-Accel-Buffering': 'no',
      },
    })
  } catch (error) {
    // Don't log ECONNREFUSED as error during backend restarts - it's expected
    const isConnRefused = error && typeof error === 'object' && 'code' in error && error.code === 'ECONNREFUSED'
    if (!isConnRefused) {
      console.error('AG-UI SSE proxy error:', error)
    } else {
      console.log('Backend temporarily unavailable (ECONNREFUSED), client will retry')
    }
    return new Response(
      JSON.stringify({ error: 'Failed to connect to AG-UI event stream' }),
      { status: 503, headers: { 'Content-Type': 'application/json' } },
    )
  }
}

