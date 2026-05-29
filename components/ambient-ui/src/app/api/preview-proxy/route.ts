import { resolveAccessToken } from "@/lib/auth"
import {
  validatePreviewUrl,
  stripFrameBlockingHeaders,
  injectBaseTag,
  buildBaseHref,
} from "@/lib/preview-proxy"

export const runtime = "nodejs"
export const dynamic = "force-dynamic"

const MAX_RESPONSE_BYTES = 10 * 1024 * 1024 // 10 MB

export async function GET(request: Request): Promise<Response> {
  const url = new URL(request.url).searchParams.get("url")
  if (!url) {
    return Response.json(
      { error: "Missing 'url' query parameter" },
      { status: 400 },
    )
  }

  const validation = validatePreviewUrl(url)
  if (!validation.valid) {
    return Response.json({ error: validation.reason }, { status: 403 })
  }

  const accessToken = await resolveAccessToken(request)
  if (!accessToken) {
    return Response.json({ error: "Unauthorized" }, { status: 401 })
  }

  const parsedUrl = validation.parsed

  // SSRF guard: only fetch URLs that passed allowlist validation above.
  // Re-check protocol as defense-in-depth for static analysis (CodeQL).
  const targetUrl = parsedUrl.toString()
  if (!targetUrl.startsWith("http://") && !targetUrl.startsWith("https://")) {
    return Response.json({ error: "Invalid protocol" }, { status: 403 })
  }

  let upstream: Response
  try {
    upstream = await fetch(targetUrl, {
      headers: {
        Authorization: `Bearer ${accessToken}`,
        Accept:
          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
      },
      redirect: "manual",
      signal: AbortSignal.timeout(15_000),
    })
  } catch (error: unknown) {
    if (error instanceof Error && error.name === "AbortError") {
      console.error("[Preview proxy] request timed out:", parsedUrl.origin)
      return Response.json(
        { error: "Failed to reach preview target" },
        { status: 502 },
      )
    }
    console.error(
      "[Preview proxy] fetch failed:",
      error instanceof Error ? error.message : error,
    )
    return Response.json(
      { error: "Failed to reach preview target" },
      { status: 502 },
    )
  }

  // Handle redirects by rewriting the Location header to go through the proxy
  if (upstream.status >= 300 && upstream.status < 400) {
    const location = upstream.headers.get("location")
    if (location) {
      const absoluteLocation = new URL(location, parsedUrl).href
      const proxyLocation = `/api/preview-proxy?url=${encodeURIComponent(absoluteLocation)}`
      return new Response(null, {
        status: upstream.status,
        headers: { Location: proxyLocation },
      })
    }
  }

  // Check Content-Length if provided
  const contentLength = upstream.headers.get("content-length")
  if (contentLength && Number(contentLength) > MAX_RESPONSE_BYTES) {
    return Response.json({ error: "Response too large" }, { status: 413 })
  }

  const cleaned = stripFrameBlockingHeaders(upstream.headers)

  const responseHeaders = new Headers(cleaned)
  responseHeaders.set("Cache-Control", "no-store")
  responseHeaders.set("X-Content-Type-Options", "nosniff")

  const contentType = upstream.headers.get("content-type") ?? ""

  if (contentType.includes("text/html")) {
    const html = await upstream.text()
    if (html.length > MAX_RESPONSE_BYTES) {
      return Response.json({ error: "Response too large" }, { status: 413 })
    }
    const modified = injectBaseTag(html, buildBaseHref(parsedUrl))
    return new Response(modified, {
      status: 200,
      headers: responseHeaders,
    })
  }

  // Non-HTML: stream body through
  if (upstream.body) {
    const { readable, writable } = new TransformStream()
    upstream.body.pipeTo(writable).catch((err: unknown) => {
      if (
        err instanceof Error &&
        err.name !== "AbortError" &&
        !err.message?.includes("ResponseAborted")
      ) {
        console.error("[Preview proxy] pipe error:", err)
      }
    })
    return new Response(readable, {
      status: upstream.status,
      headers: responseHeaders,
    })
  }

  return new Response(null, {
    status: upstream.status,
    headers: responseHeaders,
  })
}
