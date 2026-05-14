import { NextRequest, NextResponse } from "next/server";
import { buildAuthorizationUrl } from "@/lib/oidc";

export async function GET(request: NextRequest) {
  const redirectUri = process.env.SSO_REDIRECT_URI
    || `${request.nextUrl.origin}/api/auth/sso/callback`;
  const returnTo = request.nextUrl.searchParams.get("returnTo") || "/";

  const { url, codeVerifier, state } = await buildAuthorizationUrl(redirectUri);

  const response = NextResponse.redirect(url);
  const cookieOpts = {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax" as const,
    path: "/",
    maxAge: 600,
  };
  response.cookies.set("oidc_code_verifier", codeVerifier, cookieOpts);
  response.cookies.set("oidc_state", state, cookieOpts);
  response.cookies.set("oidc_return_to", returnTo, cookieOpts);

  return response;
}
