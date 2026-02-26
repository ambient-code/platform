import { BACKEND_URL } from '@/lib/config';
import { buildForwardHeadersAsync } from '@/lib/auth';

export async function POST(
  request: Request,
  { params }: { params: Promise<{ name: string }> },
) {
  try {
    const { name } = await params;
    const body = await request.text();
    const headers = await buildForwardHeadersAsync(request);
    const response = await fetch(
      `${BACKEND_URL}/projects/${encodeURIComponent(name)}/mcp-config/test`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...headers },
        body,
      },
    );
    const text = await response.text();
    return new Response(text, {
      status: response.status,
      headers: { 'Content-Type': 'application/json' },
    });
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    console.error(`[proxy] POST mcp-config/test failed: ${errorMessage}`);
    if (error instanceof TypeError && (errorMessage.includes('fetch') || errorMessage.includes('network'))) {
      return Response.json({ error: 'Backend service unavailable' }, { status: 503 });
    }
    return Response.json({ error: 'Failed to test MCP server' }, { status: 500 });
  }
}
