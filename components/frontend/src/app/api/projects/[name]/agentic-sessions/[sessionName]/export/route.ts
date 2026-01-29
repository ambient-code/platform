/**
 * Session Export Endpoint Proxy
 * Downloads session chat data as JSON.
 * Supports both AG-UI format and legacy message format.
 */

import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string; sessionName: string }> },
) {
  const { name, sessionName } = await params
  const headers = await buildForwardHeadersAsync(request)

  const backendUrl = `${BACKEND_URL}/projects/${encodeURIComponent(name)}/agentic-sessions/${encodeURIComponent(sessionName)}/export`

  const resp = await fetch(backendUrl, {
    method: 'GET',
    headers,
  })

  const data = await resp.text()
  
  // Forward headers for file download
  const responseHeaders: Record<string, string> = {
    'Content-Type': 'application/json',
  }
  
  // Forward Content-Disposition if present (for download filename)
  const contentDisposition = resp.headers.get('Content-Disposition')
  if (contentDisposition) {
    responseHeaders['Content-Disposition'] = contentDisposition
  }

  return new Response(data, {
    status: resp.status,
    headers: responseHeaders,
  })
}

