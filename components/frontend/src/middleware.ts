import { NextRequest, NextResponse } from "next/server";

export function middleware(request: NextRequest) {
  if (process.env.SSO_ENABLED !== "true") {
    return NextResponse.next();
  }

  const sessionCookie = request.cookies.get("ambient-session");
  if (sessionCookie) {
    return NextResponse.next();
  }

  const loginUrl = new URL("/api/auth/sso/login", request.url);
  loginUrl.searchParams.set("returnTo", request.nextUrl.pathname);
  return NextResponse.redirect(loginUrl);
}

export const config = {
  matcher: [
    "/((?!api|_next|favicon\\.ico|.*\\.(?:svg|png|jpg|jpeg|gif|webp|ico|css|js|map)).*)",
  ],
};
