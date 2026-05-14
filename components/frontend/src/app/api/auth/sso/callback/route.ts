import { NextRequest, NextResponse } from "next/server";
import { cookies } from "next/headers";
import { exchangeCode } from "@/lib/oidc";
import { getSession } from "@/lib/session";

export async function GET(request: NextRequest) {
  const cookieStore = await cookies();
  const codeVerifier = cookieStore.get("oidc_code_verifier")?.value;
  const expectedState = cookieStore.get("oidc_state")?.value;
  const returnTo = cookieStore.get("oidc_return_to")?.value || "/";

  if (!codeVerifier || !expectedState) {
    const loginUrl = new URL("/api/auth/sso/login", request.url);
    loginUrl.searchParams.set("returnTo", returnTo);
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
    session.idToken = tokens.idToken;
    session.expiresAt = tokens.expiresAt;
    await session.save();

    cookieStore.delete("oidc_code_verifier");
    cookieStore.delete("oidc_state");
    cookieStore.delete("oidc_return_to");

    const origin = process.env.SSO_REDIRECT_URI
      ? new URL(process.env.SSO_REDIRECT_URI).origin
      : request.nextUrl.origin;
    return NextResponse.redirect(new URL(returnTo, origin));
  } catch (err) {
    console.error("OIDC callback failed:", err instanceof Error ? err.message : err);
    cookieStore.delete("oidc_code_verifier");
    cookieStore.delete("oidc_state");
    cookieStore.delete("oidc_return_to");
    const loginUrl = new URL("/api/auth/sso/login", request.url);
    loginUrl.searchParams.set("returnTo", returnTo);
    return NextResponse.redirect(loginUrl);
  }
}
