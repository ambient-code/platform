import { resolveAccessToken, buildProxyHeaders } from "@/lib/auth"
import { getRuntimeConfig } from "@/lib/runtime-config"

export const runtime = "nodejs"
export const dynamic = "force-dynamic"

const METHODS_WITH_BODY = new Set(["POST", "PUT", "PATCH"])

async function proxyRequest(
  request: Request,
  { params }: { params: Promise<{ path: string[] }> },
): Promise<Response> {
  const { path } = await params
  if (path.some(s => s === ".." || s === ".")) {
    return Response.json({ error: "invalid_path" }, { status: 400 })
  }

  const config = await getRuntimeConfig()
  const apiServerUrl = config.apiServerUrl

  const pathStr = path.map(s => encodeURIComponent(s)).join("/")
  const url = new URL(`/api/ambient/v1/${pathStr}`, apiServerUrl)
  url.search = new URL(request.url).search

  // Auth priority: custom token > SSO/oauth token > dev-mode token
  let accessToken: string | undefined
  if (config.customToken) {
    accessToken = config.customToken
  } else {
    accessToken = await resolveAccessToken(request)
  }

  if (!accessToken) {
    return Response.json({ error: "Unauthorized" }, { status: 401 })
  }

  const headers: Record<string, string> = buildProxyHeaders(accessToken)

  const accept = request.headers.get("accept")
  if (accept) {
    headers["Accept"] = accept
  }

  const contentType = request.headers.get("content-type")
  if (contentType) {
    headers["content-type"] = contentType
  }

  let body: string | undefined
  if (METHODS_WITH_BODY.has(request.method)) {
    body = await request.text()
  }

  let upstream: Response
  try {
    upstream = await fetch(url.toString(), {
      method: request.method,
      headers,
      body,
    })
  } catch (error: unknown) {
    console.error("[Ambient API proxy] fetch failed:", error instanceof Error ? error.message : error)
    return Response.json(
      { error: "Failed to reach ambient API" },
      { status: 502 },
    )
  }

  if (!upstream.ok) {
    console.error("[Ambient API proxy] upstream error:", upstream.status, pathStr)
  }

  const upstreamContentType = upstream.headers.get("content-type") || ""
  if (
    upstreamContentType.includes("text/event-stream") ||
    upstreamContentType.includes("application/x-ndjson")
  ) {
    const { readable, writable } = new TransformStream()

    if (upstream.body) {
      upstream.body.pipeTo(writable).catch((err: unknown) => {
        if (err instanceof Error && err.name !== "AbortError" && !err.message?.includes("ResponseAborted")) {
          console.error("Ambient API proxy pipe error:", err)
        }
      })
    } else {
      writable.close()
    }

    return new Response(readable, {
      status: upstream.status,
      headers: {
        "Content-Type": upstreamContentType,
        "Cache-Control": "no-cache, no-store, must-revalidate",
        Connection: "keep-alive",
        "X-Accel-Buffering": "no",
      },
    })
  }

  const text = await upstream.text()
  return new Response(text, {
    status: upstream.status,
    headers: { "Content-Type": upstreamContentType || "application/json" },
  })
}

export const GET = proxyRequest
export const POST = proxyRequest
export const PUT = proxyRequest
export const PATCH = proxyRequest
export const DELETE = proxyRequest
