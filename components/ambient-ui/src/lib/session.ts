import { getIronSession, type SessionOptions } from "iron-session"
import { randomBytes } from "crypto"
import { cookies } from "next/headers"
import { env } from "@/lib/env"

export type SessionData = {
  accessToken: string
  refreshToken: string
  expiresAt: number
  customApiServerUrl?: string
}

export type ContextSessionData = {
  customToken?: string
}

const devSessionSecret = randomBytes(32).toString("hex")

function getSessionOptions(): SessionOptions {
  const secret = env.SESSION_SECRET
  if (!secret) {
    if (process.env.NODE_ENV === "production") {
      throw new Error("SESSION_SECRET must be set in production")
    }
    return {
      password: devSessionSecret,
      cookieName: "ambient-ui-session",
      cookieOptions: {
        secure: false,
        httpOnly: true,
        sameSite: "lax" as const,
        path: "/",
      },
    }
  }

  return {
    password: secret,
    cookieName: "ambient-ui-session",
    cookieOptions: {
      secure: process.env.NODE_ENV === "production",
      httpOnly: true,
      sameSite: "lax" as const,
      path: "/",
    },
  }
}

export async function getSession() {
  return getIronSession<SessionData>(await cookies(), getSessionOptions())
}

export async function getContextSession() {
  const opts = getSessionOptions()
  return getIronSession<ContextSessionData>(await cookies(), {
    ...opts,
    cookieName: `${opts.cookieName}-ctx`,
  })
}

export async function getAccessToken(): Promise<string | undefined> {
  const session = await getSession()
  if (!session.accessToken) return undefined

  if (Date.now() / 1000 < session.expiresAt - 60) {
    return session.accessToken
  }

  if (!session.refreshToken) {
    console.warn("SSO: session expired with no refresh token, destroying")
    session.destroy()
    return undefined
  }

  try {
    console.log("SSO: refreshing access token (expired at", new Date(session.expiresAt * 1000).toISOString(), ")")
    const { refreshOIDCTokens } = await import("./oidc")
    const tokens = await refreshOIDCTokens(session.refreshToken)
    session.accessToken = tokens.accessToken
    session.refreshToken = tokens.refreshToken
    session.expiresAt = tokens.expiresAt
    await session.save()
    if (tokens.idToken) {
      const cookieStore = await cookies()
      cookieStore.set("oidc_id_token", tokens.idToken, {
        httpOnly: true,
        secure: process.env.NODE_ENV === "production",
        sameSite: "lax",
        path: "/",
        maxAge: 86400,
      })
    }
    console.log("SSO: token refreshed, new expiry", new Date(tokens.expiresAt * 1000).toISOString())
    return session.accessToken
  } catch (err) {
    console.error("SSO: token refresh failed, destroying session:", err instanceof Error ? err.message : err)
    session.destroy()
    return undefined
  }
}
