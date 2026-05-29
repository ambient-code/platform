// eslint-disable-next-line @typescript-eslint/no-require-imports
const path = require('path')

const DEFAULT_PATTERNS = 'localhost:*,127.0.0.1:*'

/**
 * Convert a preview-host glob pattern to CSP frame-src source(s).
 *
 * CSP requires a scheme for host:port patterns, so `localhost:*` becomes
 * `http://localhost:* https://localhost:*`.  Subdomain wildcards like
 * `*.example.com` are valid CSP syntax and pass through unchanged.
 */
function toFrameSrcEntries(pattern) {
  if (pattern.includes('://')) {
    return [pattern]
  }
  // CSP only supports a wildcard as the leftmost label (e.g. *.example.com).
  // Mid-domain wildcards like *.apps.rosa.*.openshiftapps.com are invalid CSP.
  // Collapse to a valid prefix wildcard by keeping everything after the last *.
  // e.g. *.apps.rosa.*.openshiftapps.com → *.openshiftapps.com
  let cspPattern = pattern
  const midWildcard = /\*\.[^*]+\*\./
  if (midWildcard.test(pattern)) {
    const lastWildIdx = pattern.lastIndexOf('*.')
    cspPattern = '*.' + pattern.slice(lastWildIdx + 2)
  }
  return [`http://${cspPattern}`, `https://${cspPattern}`]
}

/**
 * Build the CSP frame-src directive from NEXT_PUBLIC_PREVIEW_ALLOWED_HOSTS.
 */
function buildFrameSrc() {
  const raw = process.env.NEXT_PUBLIC_PREVIEW_ALLOWED_HOSTS
  const source = (raw && raw.trim()) || DEFAULT_PATTERNS
  const patterns = source
    .split(',')
    .map((p) => p.trim())
    .filter((p) => p.length > 0)

  const entries = patterns.flatMap(toFrameSrcEntries)
  return ["'self'", ...entries].join(' ')
}

/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'standalone',
  outputFileTracingRoot: path.resolve(__dirname, '../..'),
  transpilePackages: ['ambient-sdk'],
  experimental: {
    staticGenerationMinPagesPerWorker: 100,
  },
  async headers() {
    return [
      {
        source: '/(.*)',
        headers: [
          {
            key: 'Content-Security-Policy',
            value: `frame-src ${buildFrameSrc()};`,
          },
        ],
      },
    ]
  },
}

module.exports = nextConfig
