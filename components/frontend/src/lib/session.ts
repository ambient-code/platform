import { getIronSession, type SessionOptions } from "iron-session";
import { cookies } from "next/headers";

export interface SessionData {
  accessToken: string;
  refreshToken: string;
  idToken: string;
  expiresAt: number;
}

const sessionOptions: SessionOptions = {
  password: process.env.SESSION_SECRET || "dev-session-secret-must-be-at-least-32-chars-long",
  cookieName: "ambient-session",
  cookieOptions: {
    secure: process.env.NODE_ENV === "production",
    httpOnly: true,
    sameSite: "lax" as const,
    path: "/",
  },
};

export async function getSession() {
  return getIronSession<SessionData>(await cookies(), sessionOptions);
}

export async function getAccessToken(): Promise<string | undefined> {
  const session = await getSession();
  if (!session.accessToken) return undefined;

  if (Date.now() / 1000 < session.expiresAt - 60) {
    return session.accessToken;
  }

  if (!session.refreshToken) {
    session.destroy();
    return undefined;
  }

  try {
    const { refreshOIDCTokens } = await import("./oidc");
    const tokens = await refreshOIDCTokens(session.refreshToken);
    session.accessToken = tokens.accessToken;
    session.refreshToken = tokens.refreshToken;
    session.idToken = tokens.idToken;
    session.expiresAt = tokens.expiresAt;
    await session.save();
    return session.accessToken;
  } catch {
    session.destroy();
    return undefined;
  }
}
