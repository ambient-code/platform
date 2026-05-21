import { NextRequest, NextResponse } from "next/server";
import { cookies } from "next/headers";
import { exchangeCode } from "@/lib/oidc";
import { getSession } from "@/lib/session";
import { safeReturnTo } from "@/lib/auth-utils";

export async function GET(request: NextRequest) {
  const cookieStore = await cookies();
  const codeVerifier = cookieStore.get("oidc_code_verifier")?.value;
  const expectedState = cookieStore.get("oidc_state")?.value;
  const returnTo = safeReturnTo(cookieStore.get("oidc_return_to")?.value);

  const publicOrigin = process.env.SSO_REDIRECT_URI
    ? new URL(process.env.SSO_REDIRECT_URI).origin
    : request.nextUrl.origin;

  if (!codeVerifier || !expectedState) {
    if (request.nextUrl.searchParams.has("retried")) {
      return NextResponse.json(
        { error: "SSO login failed: OIDC session cookies are missing. Your browser may be blocking cookies required for authentication." },
        { status: 400 },
      );
    }
    const loginUrl = new URL("/api/auth/sso/login", publicOrigin);
    loginUrl.searchParams.set("returnTo", returnTo);
    loginUrl.searchParams.set("retried", "1");
    return NextResponse.redirect(loginUrl);
  }

  try {
    const incomingUrl = new URL(request.url);
    const baseRedirectUri = process.env.SSO_REDIRECT_URI || `${incomingUrl.origin}/api/auth/sso/callback`;
    const callbackUrl = new URL(baseRedirectUri);
    incomingUrl.searchParams.forEach((value, key) => {
      callbackUrl.searchParams.set(key, value);
    });

    const tokens = await exchangeCode(callbackUrl, codeVerifier, expectedState);
    const session = await getSession();
    session.accessToken = tokens.accessToken;
    session.refreshToken = tokens.refreshToken;
    session.expiresAt = tokens.expiresAt;
    await session.save();

    cookieStore.delete("oidc_code_verifier");
    cookieStore.delete("oidc_state");
    cookieStore.delete("oidc_return_to");

    const origin = process.env.SSO_REDIRECT_URI
      ? new URL(process.env.SSO_REDIRECT_URI).origin
      : request.nextUrl.origin;
    const response = NextResponse.redirect(new URL(returnTo, origin));
    if (tokens.idToken) {
      response.cookies.set("oidc_id_token", tokens.idToken, {
        httpOnly: true,
        secure: process.env.NODE_ENV === "production",
        sameSite: "lax",
        path: "/",
        maxAge: 86400,
      });
    }
    return response;
  } catch (err) {
    console.error("OIDC callback failed:", err instanceof Error ? err.message : err);
    cookieStore.delete("oidc_code_verifier");
    cookieStore.delete("oidc_state");
    cookieStore.delete("oidc_return_to");
    const loginUrl = new URL("/api/auth/sso/login", publicOrigin);
    loginUrl.searchParams.set("returnTo", returnTo);
    return NextResponse.redirect(loginUrl);
  }
}
