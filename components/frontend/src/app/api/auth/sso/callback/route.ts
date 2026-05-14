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
    return NextResponse.json(
      { error: "Missing OIDC state — please try logging in again" },
      { status: 400 },
    );
  }

  try {
    const incomingUrl = new URL(request.url);
    const baseRedirectUri = process.env.SSO_REDIRECT_URI || `${incomingUrl.origin}/api/auth/sso/callback`;
    const callbackUrl = new URL(baseRedirectUri);
    incomingUrl.searchParams.forEach((value, key) => {
      callbackUrl.searchParams.set(key, value);
    });

    // Keycloak sends iss matching KC_HOSTNAME (internal URL). If it somehow
    // doesn't match (e.g., KC_HOSTNAME not set), remap to the configured issuer.
    const internalIssuer = process.env.SSO_ISSUER_URL;
    const callbackIss = callbackUrl.searchParams.get("iss");
    if (internalIssuer && callbackIss && callbackIss !== internalIssuer) {
      callbackUrl.searchParams.set("iss", internalIssuer);
    }
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

    return NextResponse.redirect(new URL(returnTo, request.nextUrl.origin));
  } catch (err) {
    const message = err instanceof Error ? err.message : "Unknown error";
    return NextResponse.json(
      { error: "OIDC callback failed", detail: message },
      { status: 500 },
    );
  }
}
