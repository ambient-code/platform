import { BACKEND_URL } from '@/lib/config';
import { buildForwardHeadersAsync } from '@/lib/auth';

export async function DELETE(
  request: Request,
  { params }: { params: Promise<{ name: string; sessionName: string; repoName: string }> },
) {
  const { name, sessionName, repoName } = await params;
  const headers = await buildForwardHeadersAsync(request);
  const { searchParams } = new URL(request.url);
  const qs = searchParams.toString();
  const suffix = qs ? `?${qs}` : '';

  const resp = await fetch(
    `${BACKEND_URL}/projects/${encodeURIComponent(name)}/agentic-sessions/${encodeURIComponent(sessionName)}/repos/${encodeURIComponent(repoName)}${suffix}`,
    {
      method: 'DELETE',
      headers,
    }
  );

  const data = await resp.text();
  return new Response(data, {
    status: resp.status,
    headers: { 'Content-Type': 'application/json' }
  });
}
