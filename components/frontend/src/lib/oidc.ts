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

  cachedConfig = await client.discovery(
    serverUrl,
    clientId,
    clientSecret,
    undefined,
    useInsecure ? { execute: [client.allowInsecureRequests] } : undefined,
  );
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

  // In production, the browser and server use the same Keycloak URL, so the
  // standard library flow works (full ID token validation including iss check).
  // In dev (Kind), the browser reaches Keycloak via localhost:30090 while the
  // server uses keycloak-service:8080 — the ID token iss claim won't match the
  // discovery issuer. Fall back to a manual token exchange in that case.
  const hasSplitUrls = !!process.env.SSO_PUBLIC_ISSUER_URL
    && process.env.SSO_PUBLIC_ISSUER_URL !== process.env.SSO_ISSUER_URL;

  if (!hasSplitUrls) {
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

  // Split-URL dev mode: manual token exchange (state + PKCE still validated)
  const returnedState = callbackUrl.searchParams.get("state");
  if (returnedState !== expectedState) {
    throw new Error("OIDC state mismatch");
  }

  const code = callbackUrl.searchParams.get("code");
  if (!code) {
    throw new Error("Missing authorization code in callback");
  }

  const metadata = config.serverMetadata();
  const tokenEndpoint = String(metadata.token_endpoint);
  const redirectUri = process.env.SSO_REDIRECT_URI || callbackUrl.origin + callbackUrl.pathname;

  const body = new URLSearchParams({
    grant_type: "authorization_code",
    code,
    redirect_uri: redirectUri,
    client_id: process.env.SSO_CLIENT_ID!,
    client_secret: process.env.SSO_CLIENT_SECRET!,
    code_verifier: codeVerifier,
  });

  const resp = await fetch(tokenEndpoint, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: body.toString(),
  });

  if (!resp.ok) {
    const text = await resp.text();
    throw new Error(`Token exchange failed (${resp.status}): ${text}`);
  }

  const tokens = await resp.json() as {
    access_token: string;
    refresh_token?: string;
    id_token?: string;
    expires_in?: number;
  };

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
  const hasSplitUrls = !!process.env.SSO_PUBLIC_ISSUER_URL
    && process.env.SSO_PUBLIC_ISSUER_URL !== process.env.SSO_ISSUER_URL;

  if (!hasSplitUrls) {
    const tokens = await client.refreshTokenGrant(config, refreshToken);
    return {
      accessToken: tokens.access_token,
      refreshToken: tokens.refresh_token ?? refreshToken,
      idToken: tokens.id_token ?? "",
      expiresAt: Math.floor(Date.now() / 1000) + (tokens.expires_in ?? 300),
    };
  }

  const metadata = config.serverMetadata();
  const tokenEndpoint = String(metadata.token_endpoint);

  const body = new URLSearchParams({
    grant_type: "refresh_token",
    refresh_token: refreshToken,
    client_id: process.env.SSO_CLIENT_ID!,
    client_secret: process.env.SSO_CLIENT_SECRET!,
  });

  const resp = await fetch(tokenEndpoint, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: body.toString(),
  });

  if (!resp.ok) {
    const text = await resp.text();
    throw new Error(`Token refresh failed (${resp.status}): ${text}`);
  }

  const tokens = await resp.json() as {
    access_token: string;
    refresh_token?: string;
    id_token?: string;
    expires_in?: number;
  };

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
