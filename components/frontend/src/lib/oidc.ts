import * as client from "openid-client";

const DISCOVERY_TTL_MS = 5 * 60 * 1000;

let cachedConfig: client.Configuration | null = null;
let cachedAt = 0;

async function getOIDCConfig(): Promise<client.Configuration> {
  if (cachedConfig && Date.now() - cachedAt < DISCOVERY_TTL_MS) {
    return cachedConfig;
  }

  const issuerURL = process.env.SSO_ISSUER_URL;
  const clientId = process.env.SSO_CLIENT_ID;
  const clientSecret = process.env.SSO_CLIENT_SECRET;

  if (!issuerURL || !clientId || !clientSecret) {
    throw new Error("SSO_ISSUER_URL, SSO_CLIENT_ID, and SSO_CLIENT_SECRET must be set");
  }

  const serverUrl = new URL(issuerURL);
  const useInsecure = serverUrl.protocol === "http:";

  // With hostname-backchannel-dynamic, the discovery response's issuer
  // (public URL) differs from the fetch URL (internal). openid-client v6
  // rejects this mismatch. Fetch discovery manually and construct config.
  // See: https://github.com/panva/openid-client/issues/737
  const wellKnownUrl = `${issuerURL}/.well-known/openid-configuration`;
  const resp = await fetch(wellKnownUrl);
  if (!resp.ok) {
    throw new Error(`OIDC discovery failed: ${resp.status}`);
  }
  const metadata = await resp.json();

  cachedConfig = new client.Configuration(
    metadata as client.ServerMetadata,
    clientId,
    clientSecret,
  );

  if (useInsecure) {
    client.allowInsecureRequests(cachedConfig);
  }

  cachedAt = Date.now();
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
    scope: "openid email profile",
    code_challenge: codeChallenge,
    code_challenge_method: "S256",
    state,
  });

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

export async function getEndSessionUrl(postLogoutRedirectUri: string, idTokenHint?: string): Promise<string> {
  const config = await getOIDCConfig();
  const metadata = config.serverMetadata();
  const endSessionEndpoint = metadata.end_session_endpoint;
  if (!endSessionEndpoint) {
    return postLogoutRedirectUri;
  }
  const url = new URL(String(endSessionEndpoint));
  if (idTokenHint) {
    url.searchParams.set("id_token_hint", idTokenHint);
  } else {
    url.searchParams.set("client_id", process.env.SSO_CLIENT_ID || "");
  }
  url.searchParams.set("post_logout_redirect_uri", postLogoutRedirectUri);
  return url.href;
}
