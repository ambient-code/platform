import { buildForwardHeadersAsync } from '@/lib/auth'
import { BACKEND_URL } from '@/lib/config';

// Map file extensions to MIME types for inline rendering
const INLINE_MIME_TYPES: Record<string, string> = {
  pdf: 'application/pdf', html: 'text/html', htm: 'text/html',
  png: 'image/png', jpg: 'image/jpeg', jpeg: 'image/jpeg',
  gif: 'image/gif', svg: 'image/svg+xml', webp: 'image/webp',
  bmp: 'image/bmp', ico: 'image/x-icon',
}

export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string; sessionName: string; path: string[] }> },
) {
  const { name, sessionName, path } = await params
  const headers = await buildForwardHeadersAsync(request)
  const rel = path.map(s => encodeURIComponent(s)).join('/')
  const resp = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/agentic-sessions/${encodeURIComponent(sessionName)}/workspace/${rel}`, { headers })
  const buf = await resp.arrayBuffer()

  // Determine content type: use extension-based detection if backend returns generic octet-stream
  let contentType = resp.headers.get('content-type') || 'application/octet-stream'
  const fileName = path[path.length - 1] || ''
  const ext = fileName.split('.').pop()?.toLowerCase() || ''

  if (contentType === 'application/octet-stream' && INLINE_MIME_TYPES[ext]) {
    contentType = INLINE_MIME_TYPES[ext]
  }

  // Set Content-Disposition: inline for renderable types so browsers display instead of download
  const responseHeaders: Record<string, string> = { 'Content-Type': contentType }
  if (INLINE_MIME_TYPES[ext]) {
    responseHeaders['Content-Disposition'] = 'inline'
  }

  return new Response(buf, { status: resp.status, headers: responseHeaders })
}


export async function PUT(
  request: Request,
  { params }: { params: Promise<{ name: string; sessionName: string; path: string[] }> },
) {
  const { name, sessionName, path } = await params
  const headers = await buildForwardHeadersAsync(request)
  const rel = path.map(s => encodeURIComponent(s)).join('/')
  const contentType = request.headers.get('content-type') || 'text/plain; charset=utf-8'
  const textBody = await request.text()
  const resp = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/agentic-sessions/${encodeURIComponent(sessionName)}/workspace/${rel}`, {
    method: 'PUT',
    headers: { ...headers, 'Content-Type': contentType },
    body: textBody,
  })
  const respBody = await resp.text()
  return new Response(respBody, { status: resp.status, headers: { 'Content-Type': 'application/json' } })
}

export async function DELETE(
  request: Request,
  { params }: { params: Promise<{ name: string; sessionName: string; path: string[] }> },
) {
  const { name, sessionName, path } = await params
  const headers = await buildForwardHeadersAsync(request)
  const rel = path.map(s => encodeURIComponent(s)).join('/')
  const resp = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/agentic-sessions/${encodeURIComponent(sessionName)}/workspace/${rel}`, {
    method: 'DELETE',
    headers,
  })
  const respBody = await resp.text()
  return new Response(respBody, { status: resp.status, headers: { 'Content-Type': 'application/json' } })
}
