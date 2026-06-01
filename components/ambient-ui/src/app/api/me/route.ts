import { getSession } from "@/lib/session"

export const runtime = "nodejs"
export const dynamic = "force-dynamic"

export async function GET() {
  try {
    const session = await getSession()
    const hasToken = !!session.accessToken
    const expiresAt = session.expiresAt
      ? new Date(session.expiresAt * 1000).toISOString()
      : null

    let claims: Record<string, unknown> | null = null
    if (session.accessToken) {
      try {
        const parts = session.accessToken.split(".")
        if (parts.length === 3) {
          const payload = JSON.parse(
            Buffer.from(parts[1], "base64url").toString()
          )
          claims = {
            sub: payload.sub,
            preferred_username: payload.preferred_username,
            given_name: payload.given_name,
            family_name: payload.family_name,
            email: payload.email,
            name: payload.name,
          }
        }
      } catch {
        // Not a JWT or malformed
      }
    }

    return Response.json({
      authenticated: hasToken,
      expiresAt,
      claims,
    })
  } catch (err) {
    return Response.json({
      authenticated: false,
      error: err instanceof Error ? err.message : "Session read failed",
    })
  }
}
