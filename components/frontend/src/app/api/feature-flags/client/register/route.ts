import { env } from '@/lib/env';
import { NextRequest } from 'next/server';

/**
 * POST /api/feature-flags/client/register
 * Proxies client registration from the Unleash SDK to the Unleash server.
 * This allows Unleash to track connected clients/applications.
 */
export async function POST(request: NextRequest) {
  const baseUrl = env.UNLEASH_URL?.replace(/\/$/, '');
  const clientKey = env.UNLEASH_CLIENT_KEY;

  // If Unleash isn't configured, just acknowledge the request
  if (!baseUrl || !clientKey) {
    return new Response(null, { status: 202 });
  }

  const url = new URL('/api/frontend/client/register', baseUrl);

  try {
    const body = await request.json();

    const res = await fetch(url.toString(), {
      method: 'POST',
      headers: {
        Authorization: clientKey,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    });

    if (!res.ok) {
      console.error('Unleash register proxy error:', res.status, await res.text());
      // Still return 202 to not break the client
      return new Response(null, { status: 202 });
    }

    return new Response(null, { status: 202 });
  } catch (error) {
    console.error('Unleash register proxy fetch error:', error);
    return new Response(null, { status: 202 });
  }
}
