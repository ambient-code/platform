import { BACKEND_URL } from '@/lib/config';
import { buildForwardHeadersAsync } from '@/lib/auth';

type Ctx = { params: Promise<{ name: string; sessionName: string }> };

// GET /api/projects/[name]/agentic-sessions/[sessionName]/oauth/google/url
// Proxy to backend OAuth URL generation endpoint with HMAC-signed state
export async function GET(request: Request, { params }: Ctx) {
  try {
    const { name, sessionName } = await params;
    const headers = await buildForwardHeadersAsync(request);
    const response = await fetch(
      `${BACKEND_URL}/projects/${encodeURIComponent(name)}/agentic-sessions/${encodeURIComponent(sessionName)}/oauth/google/url`,
      { headers }
    );
    const text = await response.text();
    return new Response(text, {
      status: response.status,
      headers: { 'Content-Type': 'application/json' },
    });
  } catch (error) {
    console.error('Error fetching Google OAuth URL:', error);
    return Response.json(
      { error: 'Failed to fetch Google OAuth URL' },
      { status: 500 }
    );
  }
}
