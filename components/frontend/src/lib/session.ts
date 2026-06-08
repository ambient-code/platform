import { getIronSession, type SessionOptions } from "iron-session";
import { cookies } from "next/headers";

export interface SessionData {
  accessToken: string;
  refreshToken: string;
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
    console.warn("SSO: session expired with no refresh token, destroying");
    session.destroy();
    return undefined;
  }

  try {
    console.log("SSO: refreshing access token (expired at", new Date(session.expiresAt * 1000).toISOString(), ")");
    const { refreshOIDCTokens } = await import("./oidc");
    const tokens = await refreshOIDCTokens(session.refreshToken);
    session.accessToken = tokens.accessToken;
    session.refreshToken = tokens.refreshToken;
    session.expiresAt = tokens.expiresAt;
    await session.save();
    if (tokens.idToken) {
      const cookieStore = await cookies();
      cookieStore.set("oidc_id_token", tokens.idToken, {
        httpOnly: true,
        secure: process.env.NODE_ENV === "production",
        sameSite: "lax",
        path: "/",
        maxAge: 86400,
      });
    }
    console.log("SSO: token refreshed, new expiry", new Date(tokens.expiresAt * 1000).toISOString());
    return session.accessToken;
  } catch (err) {
    console.error("SSO: token refresh failed, destroying session:", err instanceof Error ? err.message : err);
    session.destroy();
    return undefined;
  }
}
