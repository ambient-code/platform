import { BACKEND_URL } from '@/lib/config';
import { buildForwardHeadersAsync } from '@/lib/auth';

export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name } = await params;
    const headers = await buildForwardHeadersAsync(request);

    const resp = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/integration-status`, { headers });
    const data = await resp.json().catch(() => ({ github: false }));
    return Response.json(data, { status: resp.status });
  } catch (error) {
    console.error('Error fetching integration status:', error);
    return Response.json({ error: 'Failed to fetch integration status' }, { status: 500 });
  }
}

