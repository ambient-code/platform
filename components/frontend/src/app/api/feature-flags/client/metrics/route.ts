import { env } from '@/lib/env';
import { NextRequest } from 'next/server';

/**
 * POST /api/feature-flags/client/metrics
 * Proxies usage metrics from the Unleash SDK to the Unleash server.
 * This enables impression data and usage tracking in Unleash.
 */
export async function POST(request: NextRequest) {
  const baseUrl = env.UNLEASH_URL?.replace(/\/$/, '');
  const clientKey = env.UNLEASH_CLIENT_KEY;

  // If Unleash isn't configured, just acknowledge the request
  if (!baseUrl || !clientKey) {
    console.log('[Unleash Metrics] Unleash not configured, ignoring metrics');
    return new Response(null, { status: 202 });
  }

  const url = new URL('/api/frontend/client/metrics', baseUrl);

  try {
    const body = await request.json();
    console.log('[Unleash Metrics] Forwarding metrics to:', url.toString());
    console.log('[Unleash Metrics] Payload:', JSON.stringify(body, null, 2));

    const res = await fetch(url.toString(), {
      method: 'POST',
      headers: {
        Authorization: clientKey,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    });

    if (!res.ok) {
      const errorText = await res.text();
      console.error('[Unleash Metrics] Error:', res.status, errorText);
      // Still return 202 to not break the client
      return new Response(null, { status: 202 });
    }

    console.log('[Unleash Metrics] Success:', res.status);
    return new Response(null, { status: 202 });
  } catch (error) {
    console.error('[Unleash Metrics] Fetch error:', error);
    return new Response(null, { status: 202 });
  }
}
