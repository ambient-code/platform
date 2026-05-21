import { NextRequest, NextResponse } from "next/server";
import { getSession } from "@/lib/session";

export async function POST(request: NextRequest) {
  // Double guard: E2E_TEST_HELPERS must be explicitly set AND we must not
  // be in an environment that looks like production (has a public route).
  // This prevents accidental auth bypass if E2E_TEST_HELPERS leaks into
  // a production overlay.
  if (
    process.env.E2E_TEST_HELPERS !== "true" ||
    process.env.AMBIENT_ENV === "production"
  ) {
    return NextResponse.json({ error: "Not available" }, { status: 404 });
  }

  const { token } = await request.json() as { token: string };
  if (!token) {
    return NextResponse.json({ error: "Token required" }, { status: 400 });
  }

  const session = await getSession();
  session.accessToken = token;
  session.refreshToken = "";
  session.expiresAt = Math.floor(Date.now() / 1000) + 86400;
  await session.save();

  return NextResponse.json({ ok: true });
}
