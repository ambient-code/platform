/**
 * OAuth2 Callback Proxy Route
 * GET /oauth2callback?code=...&state=...
 * Proxies OAuth callbacks to the backend and returns HTML success page
 */

import { BACKEND_URL } from '@/lib/config'

export const dynamic = 'force-dynamic'

/**
 * Escape HTML special characters to prevent XSS attacks
 */
function escapeHtml(str: string): string {
  return str
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

export async function GET(request: Request) {
  const url = new URL(request.url)
  const searchParams = url.searchParams

  // Build backend URL with query parameters
  const backendUrl = `${BACKEND_URL.replace('/api', '')}/oauth2callback?${searchParams.toString()}`

  try {
    const response = await fetch(backendUrl, {
      method: 'GET',
    })

    if (!response.ok) {
      let errorText = 'OAuth callback failed'
      try {
        errorText = await response.text()
      } catch {
        // Use default error message if response body can't be read
      }
      return new Response(
        `
        <!DOCTYPE html>
        <html>
          <head>
            <title>Authentication Failed</title>
            <style>
              body { font-family: system-ui; max-width: 600px; margin: 100px auto; text-align: center; }
              .error { color: #dc2626; }
            </style>
          </head>
          <body>
            <h1 class="error">❌ Authentication Failed</h1>
            <p>${escapeHtml(errorText)}</p>
            <p>You can close this window and try again.</p>
          </body>
        </html>
        `,
        {
          status: response.status,
          headers: { 'Content-Type': 'text/html' },
        }
      )
    }

    // Success - return HTML page that can be closed
    return new Response(
      `
      <!DOCTYPE html>
      <html>
        <head>
          <title>Authentication Successful</title>
          <style>
            body { font-family: system-ui; max-width: 600px; margin: 100px auto; text-align: center; }
            .success { color: #16a34a; }
          </style>
        </head>
        <body>
          <h1 class="success">✅ Authentication Successful</h1>
          <p>Your Google Drive has been connected successfully.</p>
          <p>You can close this window and return to the application.</p>
          <script>
            // Auto-close after 3 seconds
            setTimeout(() => {
              window.close();
            }, 3000);
          </script>
        </body>
      </html>
      `,
      {
        status: 200,
        headers: { 'Content-Type': 'text/html' },
      }
    )
  } catch (error) {
    console.error('OAuth callback proxy error:', error)
    return new Response(
      `
      <!DOCTYPE html>
      <html>
        <head>
          <title>Error</title>
          <style>
            body { font-family: system-ui; max-width: 600px; margin: 100px auto; text-align: center; }
            .error { color: #dc2626; }
          </style>
        </head>
        <body>
          <h1 class="error">❌ Error</h1>
          <p>${error instanceof Error ? escapeHtml(error.message) : 'Failed to process OAuth callback'}</p>
          <p>You can close this window and try again.</p>
        </body>
      </html>
      `,
      {
        status: 500,
        headers: { 'Content-Type': 'text/html' },
      }
    )
  }
}
