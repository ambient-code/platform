import * as client from "openid-client";

let cachedConfig: client.Configuration | null = null;

async function getOIDCConfig(): Promise<client.Configuration> {
  if (cachedConfig) return cachedConfig;

  const issuerURL = process.env.SSO_ISSUER_URL;
  const clientId = process.env.SSO_CLIENT_ID;
  const clientSecret = process.env.SSO_CLIENT_SECRET;

  if (!issuerURL || !clientId || !clientSecret) {
    throw new Error("SSO_ISSUER_URL, SSO_CLIENT_ID, and SSO_CLIENT_SECRET must be set");
  }

  const serverUrl = new URL(issuerURL);
  const useInsecure = serverUrl.protocol === "http:";

  cachedConfig = await client.discovery(
    serverUrl,
    clientId,
    clientSecret,
    undefined,
    useInsecure ? { execute: [client.allowInsecureRequests] } : undefined,
  );
  return cachedConfig;
}

export async function buildAuthorizationUrl(redirectUri: string): Promise<{
  url: string;
  codeVerifier: string;
  state: string;
}> {
  const config = await getOIDCConfig();
  const codeVerifier = client.randomPKCECodeVerifier();
  const codeChallenge = await client.calculatePKCECodeChallenge(codeVerifier);
  const state = client.randomState();

  const redirectTo = client.buildAuthorizationUrl(config, {
    redirect_uri: redirectUri,
    scope: "openid",
    code_challenge: codeChallenge,
    code_challenge_method: "S256",
    state,
  });

  // In Kind/dev, the browser needs to reach Keycloak via an external URL
  // (e.g., localhost:30090) while the server uses the internal cluster URL.
  const publicIssuer = process.env.SSO_PUBLIC_ISSUER_URL;
  if (publicIssuer) {
    const internalIssuer = process.env.SSO_ISSUER_URL || "";
    const url = redirectTo.href.replace(internalIssuer, publicIssuer);
    return { url, codeVerifier, state };
  }

  return { url: redirectTo.href, codeVerifier, state };
}

export async function exchangeCode(
  callbackUrl: URL,
  codeVerifier: string,
  expectedState: string,
): Promise<{
  accessToken: string;
  refreshToken: string;
  idToken: string;
  expiresAt: number;
}> {
  const config = await getOIDCConfig();
  const tokens = await client.authorizationCodeGrant(config, callbackUrl, {
    pkceCodeVerifier: codeVerifier,
    expectedState,
  });

  return {
    accessToken: tokens.access_token,
    refreshToken: tokens.refresh_token ?? "",
    idToken: tokens.id_token ?? "",
    expiresAt: Math.floor(Date.now() / 1000) + (tokens.expires_in ?? 300),
  };
}

export async function refreshOIDCTokens(refreshToken: string): Promise<{
  accessToken: string;
  refreshToken: string;
  idToken: string;
  expiresAt: number;
}> {
  const config = await getOIDCConfig();
  const tokens = await client.refreshTokenGrant(config, refreshToken);

  return {
    accessToken: tokens.access_token,
    refreshToken: tokens.refresh_token ?? refreshToken,
    idToken: tokens.id_token ?? "",
    expiresAt: Math.floor(Date.now() / 1000) + (tokens.expires_in ?? 300),
  };
}

export async function getEndSessionUrl(idTokenHint: string, postLogoutRedirectUri: string): Promise<string> {
  const config = await getOIDCConfig();
  const metadata = config.serverMetadata();
  const endSessionEndpoint = metadata.end_session_endpoint;
  if (!endSessionEndpoint) {
    return postLogoutRedirectUri;
  }
  let endSessionUrl = String(endSessionEndpoint);
  const publicIssuer = process.env.SSO_PUBLIC_ISSUER_URL;
  const internalIssuer = process.env.SSO_ISSUER_URL || "";
  if (publicIssuer && internalIssuer) {
    endSessionUrl = endSessionUrl.replace(internalIssuer, publicIssuer);
  }
  const url = new URL(endSessionUrl);
  url.searchParams.set("id_token_hint", idTokenHint);
  url.searchParams.set("post_logout_redirect_uri", postLogoutRedirectUri);
  return url.href;
}
