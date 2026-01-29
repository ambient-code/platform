import { BACKEND_URL } from '@/lib/config';
import { buildForwardHeadersAsync } from '@/lib/auth';

export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string; sessionName: string }> },
) {
  const { name, sessionName } = await params;
  const headers = await buildForwardHeadersAsync(request);

  const resp = await fetch(
    `${BACKEND_URL}/projects/${encodeURIComponent(name)}/agentic-sessions/${encodeURIComponent(sessionName)}/repos/status`,
    {
      method: 'GET',
      headers,
    }
  );

  if (!resp.ok) {
    return new Response(
      JSON.stringify({ error: 'Failed to fetch repos status', repos: [] }),
      { status: 200, headers: { 'Content-Type': 'application/json' } }
    );
  }

  const data = await resp.text();
  return new Response(data, {
    status: resp.status,
    headers: { 'Content-Type': 'application/json' }
  });
}
