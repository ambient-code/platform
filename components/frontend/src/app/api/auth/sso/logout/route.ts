import { NextRequest, NextResponse } from "next/server";
import { getSession } from "@/lib/session";
import { getEndSessionUrl } from "@/lib/oidc";

export async function GET(request: NextRequest) {
  const session = await getSession();
  const idToken = session.idToken || "";
  session.destroy();

  const postLogoutRedirectUri = process.env.SSO_REDIRECT_URI
    ? new URL(process.env.SSO_REDIRECT_URI).origin
    : request.nextUrl.origin;

  if (process.env.SSO_ISSUER_URL && idToken) {
    const endSessionUrl = await getEndSessionUrl(idToken, postLogoutRedirectUri);
    return NextResponse.redirect(endSessionUrl);
  }

  return NextResponse.redirect(postLogoutRedirectUri);
}
