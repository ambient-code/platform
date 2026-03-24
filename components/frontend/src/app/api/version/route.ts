import { BACKEND_URL } from '@/lib/config';

/**
 * GET /api/version
 * Proxies to the backend version endpoint
 */
export async function GET() {
  try {
    const response = await fetch(`${BACKEND_URL}/version`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({ error: 'Unknown error' }));
      return Response.json(errorData, { status: response.status });
    }

    const data = await response.json();
    return Response.json(data);
  } catch (error) {
    console.error('Error fetching version:', error);
    return Response.json({ error: 'Failed to fetch version' }, { status: 500 });
  }
}
