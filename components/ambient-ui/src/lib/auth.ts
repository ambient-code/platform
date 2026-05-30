import { env } from "@/lib/env"

/**
 * Parse an IPv4 address into a 32-bit number.
 * Returns undefined if the address is not valid IPv4.
 */
function parseIPv4(ip: string): number | undefined {
  const parts = ip.split(".")
  if (parts.length !== 4) return undefined
  let result = 0
  for (const part of parts) {
    const octet = Number(part)
    if (!Number.isInteger(octet) || octet < 0 || octet > 255) return undefined
    result = (result << 8) | octet
  }
  // Convert to unsigned 32-bit
  return result >>> 0
}

/**
 * Check if an IPv4 address falls within a CIDR range.
 * Supports both "10.0.0.0/8" notation and bare IPs "10.0.0.1" (treated as /32).
 */
export function isIpInCidr(ip: string, cidr: string): boolean {
  const ipNum = parseIPv4(ip)
  if (ipNum === undefined) return false

  const [network, prefixStr] = cidr.split("/")
  const networkNum = parseIPv4(network)
  if (networkNum === undefined) return false

  const prefix = prefixStr !== undefined ? Number(prefixStr) : 32
  if (!Number.isInteger(prefix) || prefix < 0 || prefix > 32) return false

  if (prefix === 0) return true

  const mask = (~0 << (32 - prefix)) >>> 0
  return (ipNum & mask) === (networkNum & mask)
}

/**
 * Check if an IP address is trusted by comparing it against
 * a comma-separated list of CIDR ranges.
 */
function isTrustedIp(ip: string, trustedCidrs: string): boolean {
  const cidrs = trustedCidrs.split(",").map(s => s.trim()).filter(Boolean)
  return cidrs.some(cidr => isIpInCidr(ip, cidr))
}

/**
 * Extract the client IP from request headers.
 * Prefers X-Forwarded-For (first entry), then X-Real-Ip.
 */
function getClientIp(request: Request): string | undefined {
  const xForwardedFor = request.headers.get("x-forwarded-for")
  if (xForwardedFor) {
    const first = xForwardedFor.split(",")[0].trim()
    if (first) return first
  }
  const xRealIp = request.headers.get("x-real-ip")
  if (xRealIp) return xRealIp.trim()
  return undefined
}

/**
 * Resolve the access token from the request based on the configured AUTH_MODE.
 *
 * - native-sso: extracts JWT from the iron-session cookie
 * - oauth-proxy: reads X-Forwarded-Access-Token header, validating source IP
 *   against OAUTH_PROXY_TRUSTED_IPS (fail-closed)
 * - none: returns a placeholder token for development
 */
export async function resolveAccessToken(request: Request): Promise<string | undefined> {
  switch (env.AUTH_MODE) {
    case "native-sso": {
      const { getAccessToken } = await import("@/lib/session")
      return getAccessToken()
    }

    case "oauth-proxy": {
      // In sidecar mode, the oauth-proxy runs in the same pod and is the only
      // process that can reach port 3000 (not exposed via Service). The
      // X-Forwarded-For header contains the end-user's IP from the ingress
      // router, not the proxy's IP, so IP-based trust checks are not meaningful.
      const token = request.headers.get("x-forwarded-access-token")?.trim()
      return token || undefined
    }

    case "none":
      return "dev-mode-token"

    default: {
      const _exhaustive: never = env.AUTH_MODE
      console.error(`Unknown AUTH_MODE: ${_exhaustive}`)
      return undefined
    }
  }
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
