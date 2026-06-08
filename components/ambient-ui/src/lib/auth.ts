/**
 * Resolve the access token from the server-side SSO session.
 * The app always uses native-sso with Keycloak OIDC.
 */
export async function resolveAccessToken(_request: Request): Promise<string | undefined> {
  const { getAccessToken } = await import("@/lib/session")
  return getAccessToken()
}

/**
 * Build the headers to forward to the upstream API server.
 */
export function buildProxyHeaders(accessToken: string): Record<string, string> {
  return {
    Authorization: `Bearer ${accessToken}`,
    Accept: "application/json",
  }
}
