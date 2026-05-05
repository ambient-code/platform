import { NextRequest, NextResponse } from 'next/server';
import { buildForwardHeadersAsync } from '@/lib/auth';
import { API_SERVER_URL } from '@/lib/config';

export const runtime = 'nodejs';
export const dynamic = 'force-dynamic';

async function proxyRequest(
  request: NextRequest,
  { params }: { params: Promise<{ path: string[] }> },
): Promise<Response> {
  const { path } = await params;
  if (path.some(s => s === '..' || s === '.')) {
    return NextResponse.json({ error: 'invalid_path' }, { status: 400 });
  }
  const pathStr = path.map(s => encodeURIComponent(s)).join('/');
  const url = new URL(`/api/ambient/v1/${pathStr}`, API_SERVER_URL);
  url.search = request.nextUrl.search;

  const headers = await buildForwardHeadersAsync(request);

  // Forward the client's Accept header so the upstream can respond with the
  // correct content type (e.g. text/event-stream for SSE endpoints).
  const accept = request.headers.get('accept');
  if (accept) {
    headers['Accept'] = accept;
  }

  // Forward content-type for requests with bodies
  const contentType = request.headers.get('content-type');
  if (contentType) {
    headers['content-type'] = contentType;
  }

  let upstream: Response;
  try {
    upstream = await fetch(url.toString(), {
      method: request.method,
      headers,
      body: request.body,
      // @ts-expect-error -- Node.js fetch supports duplex for streaming request bodies
      duplex: 'half',
    });
  } catch (error: unknown) {
    console.error('[Ambient API proxy] fetch failed:', error instanceof Error ? error.message : error);
    return NextResponse.json(
      { error: 'upstream_unavailable' },
      { status: 502 },
    );
  }

  if (!upstream.ok) {
    console.error('[Ambient API proxy] upstream error:', upstream.status, pathStr);
  }

  // SSE/streaming: pipe through without buffering
  const upstreamContentType = upstream.headers.get('content-type') || '';
  if (
    upstreamContentType.includes('text/event-stream') ||
    upstreamContentType.includes('application/x-ndjson')
  ) {
    const { readable, writable } = new TransformStream();

    if (upstream.body) {
      upstream.body.pipeTo(writable).catch((err: unknown) => {
        // AbortError / ResponseAborted is normal when client disconnects
        if (err instanceof Error && err.name !== 'AbortError' && !err.message?.includes('ResponseAborted')) {
          console.error('Ambient API proxy pipe error:', err);
        }
      });
    }

    return new Response(readable, {
      status: upstream.status,
      headers: {
        'Content-Type': upstreamContentType,
        'Cache-Control': 'no-cache, no-store, must-revalidate',
        Connection: 'keep-alive',
        'X-Accel-Buffering': 'no',
      },
    });
  }

  // Non-streaming: buffer and forward
  const body = await upstream.arrayBuffer();
  return new Response(body, {
    status: upstream.status,
    headers: {
      'Content-Type': upstreamContentType || 'application/json',
    },
  });
}

export const GET = proxyRequest;
export const POST = proxyRequest;
export const PUT = proxyRequest;
export const PATCH = proxyRequest;
export const DELETE = proxyRequest;
