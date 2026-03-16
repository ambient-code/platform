import { BACKEND_URL } from '@/lib/config';
import { buildForwardHeadersAsync } from '@/lib/auth';

export async function POST(
  request: Request,
  { params }: { params: Promise<{ name: string; scheduledSessionName: string }> }
) {
  try {
    const { name, scheduledSessionName } = await params;
    const headers = await buildForwardHeadersAsync(request);
    const response = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/scheduled-sessions/${encodeURIComponent(scheduledSessionName)}/suspend`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...headers },
    });
    const text = await response.text();
    return new Response(text, { status: response.status, headers: { 'Content-Type': 'application/json' } });
  } catch (error) {
    console.error('Error suspending scheduled session:', error);
    return Response.json({ error: 'Failed to suspend scheduled session' }, { status: 500 });
  }
}
