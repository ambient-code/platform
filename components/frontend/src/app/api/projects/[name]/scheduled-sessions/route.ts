import { BACKEND_URL } from '@/lib/config';
import { buildForwardHeadersAsync } from '@/lib/auth';

// GET /api/projects/[name]/scheduled-sessions - List scheduled sessions
export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name } = await params;
    const headers = await buildForwardHeadersAsync(request);
    const response = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/scheduled-sessions`, { headers });
    const text = await response.text();
    return new Response(text, { status: response.status, headers: { 'Content-Type': 'application/json' } });
  } catch (error) {
    console.error('Error listing scheduled sessions:', error);
    return Response.json({ error: 'Failed to list scheduled sessions' }, { status: 500 });
  }
}

// POST /api/projects/[name]/scheduled-sessions - Create a scheduled session
export async function POST(
  request: Request,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name } = await params;
    const body = await request.text();
    const headers = await buildForwardHeadersAsync(request);
    const response = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/scheduled-sessions`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...headers },
      body,
    });
    const text = await response.text();
    return new Response(text, { status: response.status, headers: { 'Content-Type': 'application/json' } });
  } catch (error) {
    console.error('Error creating scheduled session:', error);
    return Response.json({ error: 'Failed to create scheduled session' }, { status: 500 });
  }
}
