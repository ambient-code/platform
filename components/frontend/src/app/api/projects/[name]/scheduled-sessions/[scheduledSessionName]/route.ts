import { BACKEND_URL } from '@/lib/config';
import { buildForwardHeadersAsync } from '@/lib/auth';

type Ctx = { params: Promise<{ name: string; scheduledSessionName: string }> };

// GET /api/projects/[name]/scheduled-sessions/[scheduledSessionName]
export async function GET(request: Request, { params }: Ctx) {
  try {
    const { name, scheduledSessionName } = await params;
    const headers = await buildForwardHeadersAsync(request);
    const response = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/scheduled-sessions/${encodeURIComponent(scheduledSessionName)}`, { headers });
    const text = await response.text();
    return new Response(text, { status: response.status, headers: { 'Content-Type': 'application/json' } });
  } catch (error) {
    console.error('Error fetching scheduled session:', error);
    return Response.json({ error: 'Failed to fetch scheduled session' }, { status: 500 });
  }
}

// PUT /api/projects/[name]/scheduled-sessions/[scheduledSessionName]
export async function PUT(request: Request, { params }: Ctx) {
  try {
    const { name, scheduledSessionName } = await params;
    const body = await request.text();
    const headers = await buildForwardHeadersAsync(request);
    const response = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/scheduled-sessions/${encodeURIComponent(scheduledSessionName)}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...headers },
      body,
    });
    const text = await response.text();
    return new Response(text, { status: response.status, headers: { 'Content-Type': 'application/json' } });
  } catch (error) {
    console.error('Error updating scheduled session:', error);
    return Response.json({ error: 'Failed to update scheduled session' }, { status: 500 });
  }
}

// DELETE /api/projects/[name]/scheduled-sessions/[scheduledSessionName]
export async function DELETE(request: Request, { params }: Ctx) {
  try {
    const { name, scheduledSessionName } = await params;
    const headers = await buildForwardHeadersAsync(request);
    const response = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/scheduled-sessions/${encodeURIComponent(scheduledSessionName)}`, {
      method: 'DELETE',
      headers,
    });
    if (response.status === 204) return new Response(null, { status: 204 });
    const text = await response.text();
    return new Response(text, { status: response.status, headers: { 'Content-Type': 'application/json' } });
  } catch (error) {
    console.error('Error deleting scheduled session:', error);
    return Response.json({ error: 'Failed to delete scheduled session' }, { status: 500 });
  }
}
