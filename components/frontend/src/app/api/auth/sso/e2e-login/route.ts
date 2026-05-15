import { NextRequest, NextResponse } from "next/server";
import { getSession } from "@/lib/session";

export async function POST(request: NextRequest) {
  // Gate on E2E_TEST_HELPERS rather than NODE_ENV so this route works in
  // Kind/CI where the Docker image sets NODE_ENV=production but E2E tests
  // still need programmatic session creation.
  if (process.env.E2E_TEST_HELPERS !== "true") {
    return NextResponse.json({ error: "Not available" }, { status: 404 });
  }

  const { token } = await request.json() as { token: string };
  if (!token) {
    return NextResponse.json({ error: "Token required" }, { status: 400 });
  }

  const session = await getSession();
  session.accessToken = token;
  session.refreshToken = "";
  session.idToken = "";
  session.expiresAt = Math.floor(Date.now() / 1000) + 86400;
  await session.save();

  return NextResponse.json({ ok: true });
}
