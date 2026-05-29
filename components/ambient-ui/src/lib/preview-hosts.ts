const DEFAULT_PATTERNS = 'localhost:*,127.0.0.1:*'

const ALLOWED_PROTOCOLS = new Set(['http:', 'https:'])

/**
 * Parse a comma-separated list of glob patterns into an array.
 * Returns default patterns when input is undefined or empty.
 */
export function parseAllowedHosts(raw: string | undefined): string[] {
  const source = raw?.trim() || DEFAULT_PATTERNS
  return source
    .split(',')
    .map((p) => p.trim())
    .filter((p) => p.length > 0)
}

/**
 * Test whether a hostname (with optional port) matches a glob pattern.
 *
 * Supported glob syntax:
 * - `*` matches one or more characters (greedy, including dots)
 * - Literal segments must match exactly
 *
 * Examples:
 * - `*.example.com` matches `app.example.com` and `deep.sub.example.com`
 * - `*.apps.rosa.*.amazonaws.com` matches `myapp.apps.rosa.us-east-1.amazonaws.com`
 * - `localhost:*` matches `localhost:3000`
 */
export function matchesGlobPattern(hostWithPort: string, pattern: string): boolean {
  // Convert glob pattern to regex:
  // - Escape regex special chars (except *)
  // - Replace * with .+ (one or more chars)
  const escaped = pattern.replace(/[.+?^${}()|[\]\\]/g, '\\$&')
  const regexStr = `^${escaped.replace(/\*/g, '.+')}$`
  const regex = new RegExp(regexStr)
  return regex.test(hostWithPort)
}

/**
 * Check if a URL is on the trusted preview hosts allowlist.
 *
 * - Only `http:` and `https:` protocols are allowed.
 * - The hostname (with port if present) is matched against glob patterns
 *   from `NEXT_PUBLIC_PREVIEW_ALLOWED_HOSTS` env var.
 * - Default allowlist: `localhost:*,127.0.0.1:*`
 */
export function isAllowedPreviewHost(url: string): boolean {
  let parsed: URL
  try {
    parsed = new URL(url)
  } catch {
    return false
  }

  if (!ALLOWED_PROTOCOLS.has(parsed.protocol)) {
    return false
  }

  const patterns = parseAllowedHosts(
    process.env.NEXT_PUBLIC_PREVIEW_ALLOWED_HOSTS
  )

  // Match against host (includes port if specified in the URL)
  // Also try matching hostname-only for patterns without a port component
  const hostWithPort = parsed.host // e.g. "localhost:3000" or "example.com"
  const hostnameOnly = parsed.hostname // e.g. "localhost" or "example.com"

  return patterns.some((pattern) => {
    if (matchesGlobPattern(hostWithPort, pattern)) {
      return true
    }
    // If the pattern has no port component, try matching hostname only
    if (!pattern.includes(':')) {
      return matchesGlobPattern(hostnameOnly, pattern)
    }
    // If the pattern has a port wildcard (e.g. localhost:*) and the URL has
    // no port, try matching hostname-only against the part before the colon
    if (pattern.endsWith(':*')) {
      const patternHost = pattern.slice(0, -2)
      return matchesGlobPattern(hostnameOnly, patternHost)
    }
    return false
  })
}
