import { env } from '@/lib/env';
import { NextRequest } from 'next/server';

/**
 * GET /api/feature-flags
 * Proxies to Unleash Frontend API when UNLEASH_URL and UNLEASH_CLIENT_KEY are set.
 * Returns empty toggles when Unleash is not configured (SDK still works, all flags off).
 * Used by @unleash/proxy-client-react so the client never sees the real Unleash URL or key.
 */
export async function GET(request: NextRequest) {
  const baseUrl = env.UNLEASH_URL?.replace(/\/$/, '');
  const clientKey = env.UNLEASH_CLIENT_KEY;

  if (!baseUrl || !clientKey) {
    return Response.json({ toggles: [] });
  }

  const url = new URL('/api/frontend', baseUrl);
  // Forward query params (e.g. projectId) if needed for strategies
  request.nextUrl.searchParams.forEach((value, key) => {
    url.searchParams.set(key, value);
  });

  try {
    const res = await fetch(url.toString(), {
      method: 'GET',
      headers: {
        Authorization: clientKey,
        'Content-Type': 'application/json',
      },
      next: { revalidate: 15 },
    });

    if (!res.ok) {
      console.error('Unleash proxy error:', res.status, await res.text());
      return Response.json({ toggles: [] });
    }

    const data = await res.json();
    return Response.json(data);
  } catch (error) {
    console.error('Unleash proxy fetch error:', error);
    return Response.json({ toggles: [] });
  }
}
