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
    const tokens = await exchangeCode(request.nextUrl, codeVerifier, expectedState);
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
