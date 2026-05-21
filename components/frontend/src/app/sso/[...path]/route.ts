import { NextRequest, NextResponse } from "next/server";

function parseSSOOrigin(): string | null {
  try {
    return process.env.SSO_ISSUER_URL
      ? new URL(process.env.SSO_ISSUER_URL).origin
      : null;
  } catch {
    console.error("SSO_ISSUER_URL is not a valid URL:", process.env.SSO_ISSUER_URL);
    return null;
  }
}
const SSO_ORIGIN = parseSSOOrigin();

export async function GET(request: NextRequest, { params }: { params: Promise<{ path: string[] }> }) {
  return proxyToKeycloak(request, await params);
}

export async function POST(request: NextRequest, { params }: { params: Promise<{ path: string[] }> }) {
  return proxyToKeycloak(request, await params);
}

const ALLOWED_PATH_PREFIX = /^realms\/[^/]+\/(protocol\/openid-connect|\.well-known|login-actions|resources)\b/;

async function proxyToKeycloak(request: NextRequest, params: { path: string[] }) {
  if (!SSO_ORIGIN) {
    return NextResponse.json({ error: "SSO not configured" }, { status: 503 });
  }

  const path = params.path.join("/");
  if (!ALLOWED_PATH_PREFIX.test(path)) {
    return NextResponse.json({ error: "Not found" }, { status: 404 });
  }
  const target = new URL(`/${path}`, SSO_ORIGIN);
  target.search = request.nextUrl.search;

  const headers = new Headers();
  for (const [key, value] of request.headers.entries()) {
    if (key === "host" || key === "connection") continue;
    headers.set(key, value);
  }
  headers.set("host", target.host);

  const resp = await fetch(target.href, {
    method: request.method,
    headers,
    body: request.method !== "GET" ? await request.text() : undefined,
    redirect: "manual",
  });

  const responseHeaders = new Headers();
  for (const [key, value] of resp.headers.entries()) {
    if (key === "transfer-encoding") continue;
    // Rewrite Location headers from internal to proxy URL
    if (key === "location" && SSO_ORIGIN) {
      responseHeaders.set(key, value.replaceAll(SSO_ORIGIN, request.nextUrl.origin + "/sso"));
    } else {
      responseHeaders.set(key, value);
    }
  }

  return new NextResponse(resp.body, {
    status: resp.status,
    headers: responseHeaders,
  });
}
