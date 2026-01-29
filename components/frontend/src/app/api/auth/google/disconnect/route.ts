/**
 * Google OAuth Disconnect API Route
 * POST /api/auth/google/disconnect
 * Disconnects Google OAuth for current user
 */

import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

export const dynamic = 'force-dynamic'

export async function POST(request: Request) {
  // Build auth headers from the incoming request
  const headers = await buildForwardHeadersAsync(request)

  // Build backend URL
  const backendUrl = `${BACKEND_URL}/auth/google/disconnect`

  try {
    const response = await fetch(backendUrl, {
      method: 'POST',
      headers,
    })

    if (!response.ok) {
      const errorText = await response.text()
      return new Response(JSON.stringify({ error: errorText }), {
        status: response.status,
        headers: { 'Content-Type': 'application/json' },
      })
    }

    const data = await response.json()
    return new Response(JSON.stringify(data), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    })
  } catch (error) {
    console.error('Google OAuth disconnect proxy error:', error)
    return new Response(
      JSON.stringify({
        error: error instanceof Error ? error.message : 'Failed to disconnect',
      }),
      { status: 500, headers: { 'Content-Type': 'application/json' } }
    )
  }
}

