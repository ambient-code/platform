import { getSession } from "@/lib/session"

export const runtime = "nodejs"
export const dynamic = "force-dynamic"

function computeInitials(name: string | undefined, username: string | undefined): string {
  if (name) {
    const parts = name.trim().split(/\s+/)
    if (parts.length >= 2) {
      return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
    }
    if (parts.length === 1 && parts[0].length > 0) {
      return parts[0][0].toUpperCase()
    }
  }
  if (username && username.length > 0) {
    return username[0].toUpperCase()
  }
  return "?"
}

export async function GET() {
  try {
    const session = await getSession()

    if (!session.accessToken) {
      return Response.json({ authenticated: false })
    }

    let username = "unknown"
    let name = ""
    let email = ""

    // Token was verified at OIDC callback; safe to decode without re-verification.
    try {
      const parts = session.accessToken.split(".")
      if (parts.length === 3) {
        const payload: Record<string, unknown> = JSON.parse(
          Buffer.from(parts[1], "base64url").toString()
        )
        username = typeof payload.preferred_username === "string"
          ? payload.preferred_username
          : "unknown"
        name = typeof payload.name === "string"
          ? payload.name
          : [payload.given_name, payload.family_name]
              .filter(v => typeof v === "string")
              .join(" ")
        email = typeof payload.email === "string" ? payload.email : ""
      }
    } catch {
      // Not a JWT or malformed — use defaults
    }

    const initials = computeInitials(name, username)

    return Response.json({
      authenticated: true,
      username,
      name,
      email,
      initials,
    })
  } catch (err) {
    console.error("[/api/me] session read failed:", err)
    return Response.json({ authenticated: false })
  }
}
