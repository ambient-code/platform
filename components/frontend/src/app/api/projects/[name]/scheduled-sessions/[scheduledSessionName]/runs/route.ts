import { BACKEND_URL } from '@/lib/config';
import { buildForwardHeadersAsync } from '@/lib/auth';

export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string; scheduledSessionName: string }> }
) {
  try {
    const { name, scheduledSessionName } = await params;
    const headers = await buildForwardHeadersAsync(request);
    const response = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/scheduled-sessions/${encodeURIComponent(scheduledSessionName)}/runs`, { headers });
    const text = await response.text();
    return new Response(text, { status: response.status, headers: { 'Content-Type': 'application/json' } });
  } catch (error) {
    console.error('Error listing scheduled session runs:', error);
    return Response.json({ error: 'Failed to list scheduled session runs' }, { status: 500 });
  }
}
