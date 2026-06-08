import { NextRequest, NextResponse } from "next/server"
import { cookies } from "next/headers"
import { getSession } from "@/lib/session"
import { getEndSessionUrl } from "@/lib/oidc"
import { env } from "@/lib/env"

export async function GET(request: NextRequest) {
  const cookieStore = await cookies()
  const idToken = cookieStore.get("oidc_id_token")?.value
  const session = await getSession()
  session.destroy()
  cookieStore.delete("oidc_id_token")

  const origin = env.SSO_REDIRECT_URI
    ? new URL(env.SSO_REDIRECT_URI).origin
    : request.nextUrl.origin
  const postLogoutRedirectUri = `${origin}/logged-out`

  if (env.SSO_ISSUER_URL) {
    const endSessionUrl = await getEndSessionUrl(postLogoutRedirectUri, idToken)
    return NextResponse.redirect(endSessionUrl)
  }

  return NextResponse.redirect(postLogoutRedirectUri)
}
