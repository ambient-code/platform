import { NextRequest, NextResponse } from "next/server"

function getAuthMode(): "native-sso" | "oauth-proxy" | "none" {
  return (process.env.AUTH_MODE || "none") as "native-sso" | "oauth-proxy" | "none"
}

export function proxy(request: NextRequest) {
  const authMode = getAuthMode()

  if (authMode === "none") {
    return NextResponse.next()
  }

  if (authMode === "oauth-proxy") {
    const forwardedUser = request.headers.get("x-forwarded-user")
    if (!forwardedUser) {
      return NextResponse.json(
        { error: "Unauthorized: missing X-Forwarded-User header" },
        { status: 401 },
      )
    }
    return NextResponse.next()
  }

  // native-sso: check for session cookie
  const sessionCookie = request.cookies.get("ambient-ui-session")
  if (sessionCookie) {
    return NextResponse.next()
  }

  // RSC/fetch requests can't follow cross-origin redirects to Keycloak.
  // Return 401 so the client-side can handle session expiry.
  const isRSC = request.headers.get("rsc") === "1"
    || request.headers.get("next-router-state-tree") !== null
  const isFetch = request.headers.get("accept")?.includes("application/json")
    || request.headers.get("x-requested-with") === "XMLHttpRequest"

  if (isRSC || isFetch) {
    return NextResponse.json(
      { error: "Session expired" },
      { status: 401 },
    )
  }

  const baseUrl = process.env.SSO_REDIRECT_URI
    ? new URL(process.env.SSO_REDIRECT_URI).origin
    : request.nextUrl.origin
  const loginUrl = new URL("/api/auth/sso/login", baseUrl)
  loginUrl.searchParams.set("returnTo", request.nextUrl.pathname)
  return NextResponse.redirect(loginUrl)
}

export const config = {
  matcher: [
    "/((?!api|_next|favicon\\.ico|.*\\.(?:svg|png|jpg|jpeg|gif|webp|ico|css|js|map)).*)",
  ],
}
