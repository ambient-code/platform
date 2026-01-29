import { BACKEND_URL } from '@/lib/config';
import { buildForwardHeadersAsync } from '@/lib/auth';

type Ctx = { params: Promise<{ name: string; sessionName: string }> };

// PUT /api/projects/[name]/agentic-sessions/[sessionName]/displayname
export async function PUT(request: Request, { params }: Ctx) {
  try {
    const { name, sessionName } = await params;
    const body = await request.text();
    const headers = await buildForwardHeadersAsync(request);
    const response = await fetch(
      `${BACKEND_URL}/projects/${encodeURIComponent(name)}/agentic-sessions/${encodeURIComponent(sessionName)}/displayname`,
      {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', ...headers },
        body,
      }
    );
    const text = await response.text();
    return new Response(text, { status: response.status, headers: { 'Content-Type': 'application/json' } });
  } catch (error) {
    console.error('Error updating session display name:', error);
    return Response.json({ error: 'Failed to update session display name' }, { status: 500 });
  }
}

