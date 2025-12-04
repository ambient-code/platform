import { BACKEND_URL } from '@/lib/config';
import { buildForwardHeadersAsync } from '@/lib/auth';

export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name } = await params;
    const headers = await buildForwardHeadersAsync(request);

    const resp = await fetch(
      `${BACKEND_URL}/projects/${encodeURIComponent(name)}/workspace-container`,
      { headers }
    );
    const data = await resp.json().catch(() => ({}));
    return Response.json(data, { status: resp.status });
  } catch (error) {
    console.error('Error fetching workspace container settings:', error);
    return Response.json(
      { error: 'Failed to fetch workspace container settings' },
      { status: 500 }
    );
  }
}

export async function PUT(
  request: Request,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name } = await params;
    const headers = await buildForwardHeadersAsync(request);
    const body = await request.json();

    const resp = await fetch(
      `${BACKEND_URL}/projects/${encodeURIComponent(name)}/workspace-container`,
      {
        method: 'PUT',
        headers: {
          ...headers,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(body),
      }
    );
    const data = await resp.json().catch(() => ({}));
    return Response.json(data, { status: resp.status });
  } catch (error) {
    console.error('Error updating workspace container settings:', error);
    return Response.json(
      { error: 'Failed to update workspace container settings' },
      { status: 500 }
    );
  }
}
