import { NextRequest, NextResponse } from "next/server";
import { buildAuthorizationUrl } from "@/lib/oidc";

function safeReturnTo(value: string | null | undefined): string {
  if (!value) return "/";
  try {
    const parsed = new URL(value, "http://localhost");
    if (parsed.origin !== "http://localhost") return "/";
    return parsed.pathname + parsed.search;
  } catch {
    return "/";
  }
}

export async function GET(request: NextRequest) {
  let redirectUri = process.env.SSO_REDIRECT_URI
    || `${request.nextUrl.origin}/api/auth/sso/callback`;
  if (request.nextUrl.searchParams.has("retried")) {
    const u = new URL(redirectUri);
    u.searchParams.set("retried", "1");
    redirectUri = u.toString();
  }
  const returnTo = safeReturnTo(request.nextUrl.searchParams.get("returnTo"));

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
