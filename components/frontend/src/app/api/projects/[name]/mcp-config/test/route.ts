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
    console.error('Error proxying POST mcp-config/test:', error);
    return Response.json({ error: 'Failed to test MCP server' }, { status: 500 });
  }
}
