import { NextRequest, NextResponse } from "next/server";
import { cookies } from "next/headers";
import { buildAuthorizationUrl } from "@/lib/oidc";

export async function GET(request: NextRequest) {
  const redirectUri = process.env.SSO_REDIRECT_URI
    || `${request.nextUrl.origin}/api/auth/sso/callback`;
  const returnTo = request.nextUrl.searchParams.get("returnTo") || "/";

  const { url, codeVerifier, state } = await buildAuthorizationUrl(redirectUri);

  const cookieStore = await cookies();
  cookieStore.set("oidc_code_verifier", codeVerifier, {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    path: "/",
    maxAge: 600,
  });
  cookieStore.set("oidc_state", state, {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    path: "/",
    maxAge: 600,
  });
  cookieStore.set("oidc_return_to", returnTo, {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    path: "/",
    maxAge: 600,
  });

  return NextResponse.redirect(url);
}
