import { describe, it, expect, vi, afterEach } from 'vitest'

describe('preview-hosts', () => {
  afterEach(() => {
    vi.unstubAllEnvs()
    // Force re-import to pick up new env
    vi.resetModules()
  })

  async function loadModule() {
    const mod = await import('../preview-hosts')
    return mod
  }

  describe('isAllowedPreviewHost', () => {
    it('allows localhost by default', async () => {
      const { isAllowedPreviewHost } = await loadModule()
      expect(isAllowedPreviewHost('http://localhost:3000/page')).toBe(true)
    })

    it('allows localhost with any port', async () => {
      const { isAllowedPreviewHost } = await loadModule()
      expect(isAllowedPreviewHost('http://localhost:8080')).toBe(true)
    })

    it('allows 127.0.0.1 with any port by default', async () => {
      const { isAllowedPreviewHost } = await loadModule()
      expect(isAllowedPreviewHost('http://127.0.0.1:4200')).toBe(true)
    })

    it('rejects hosts not on the allowlist', async () => {
      const { isAllowedPreviewHost } = await loadModule()
      expect(isAllowedPreviewHost('https://evil.example.com')).toBe(false)
    })

    it('rejects invalid URLs', async () => {
      const { isAllowedPreviewHost } = await loadModule()
      expect(isAllowedPreviewHost('not-a-url')).toBe(false)
    })

    it('rejects empty string', async () => {
      const { isAllowedPreviewHost } = await loadModule()
      expect(isAllowedPreviewHost('')).toBe(false)
    })

    it('rejects javascript: protocol', async () => {
      // eslint-disable-next-line no-script-url
      const { isAllowedPreviewHost } = await loadModule()
      expect(isAllowedPreviewHost('javascript:alert(1)')).toBe(false)
    })

    it('rejects data: protocol', async () => {
      const { isAllowedPreviewHost } = await loadModule()
      expect(isAllowedPreviewHost('data:text/html,<h1>hi</h1>')).toBe(false)
    })

    it('only allows http and https protocols', async () => {
      const { isAllowedPreviewHost } = await loadModule()
      expect(isAllowedPreviewHost('ftp://localhost:21')).toBe(false)
    })
  })

  describe('custom allowlist via env var', () => {
    it('allows hosts matching custom glob patterns', async () => {
      vi.stubEnv('NEXT_PUBLIC_PREVIEW_ALLOWED_HOSTS', '*.example.com')
      const { isAllowedPreviewHost } = await loadModule()
      expect(isAllowedPreviewHost('https://app.example.com')).toBe(true)
      expect(isAllowedPreviewHost('https://staging.example.com:8443/path')).toBe(true)
    })

    it('supports wildcard subdomain matching', async () => {
      vi.stubEnv('NEXT_PUBLIC_PREVIEW_ALLOWED_HOSTS', '*.apps.rosa.*.amazonaws.com')
      const { isAllowedPreviewHost } = await loadModule()
      expect(isAllowedPreviewHost('https://myapp.apps.rosa.us-east-1.amazonaws.com')).toBe(true)
    })

    it('supports OpenShift wildcard pattern', async () => {
      vi.stubEnv('NEXT_PUBLIC_PREVIEW_ALLOWED_HOSTS', '*.apps.*.openshiftapps.com')
      const { isAllowedPreviewHost } = await loadModule()
      expect(isAllowedPreviewHost('https://my-route.apps.cluster-abc.openshiftapps.com')).toBe(true)
    })

    it('supports multiple comma-separated patterns', async () => {
      vi.stubEnv('NEXT_PUBLIC_PREVIEW_ALLOWED_HOSTS', '*.example.com,*.internal.net')
      const { isAllowedPreviewHost } = await loadModule()
      expect(isAllowedPreviewHost('https://app.example.com')).toBe(true)
      expect(isAllowedPreviewHost('https://svc.internal.net')).toBe(true)
      expect(isAllowedPreviewHost('https://evil.other.com')).toBe(false)
    })

    it('trims whitespace from patterns', async () => {
      vi.stubEnv('NEXT_PUBLIC_PREVIEW_ALLOWED_HOSTS', ' *.example.com , *.internal.net ')
      const { isAllowedPreviewHost } = await loadModule()
      expect(isAllowedPreviewHost('https://app.example.com')).toBe(true)
    })

    it('handles port wildcards in patterns', async () => {
      vi.stubEnv('NEXT_PUBLIC_PREVIEW_ALLOWED_HOSTS', 'localhost:*')
      const { isAllowedPreviewHost } = await loadModule()
      expect(isAllowedPreviewHost('http://localhost:3000')).toBe(true)
      expect(isAllowedPreviewHost('http://localhost:9999')).toBe(true)
    })

    it('custom env replaces defaults (localhost no longer allowed)', async () => {
      vi.stubEnv('NEXT_PUBLIC_PREVIEW_ALLOWED_HOSTS', '*.example.com')
      const { isAllowedPreviewHost } = await loadModule()
      expect(isAllowedPreviewHost('http://localhost:3000')).toBe(false)
    })
  })

  describe('parseAllowedHosts', () => {
    it('parses comma-separated patterns', async () => {
      const { parseAllowedHosts } = await loadModule()
      const result = parseAllowedHosts('*.example.com,localhost:*')
      expect(result).toHaveLength(2)
    })

    it('filters out empty patterns', async () => {
      const { parseAllowedHosts } = await loadModule()
      const result = parseAllowedHosts('*.example.com,,  ,localhost:*')
      expect(result).toHaveLength(2)
    })

    it('returns default patterns for undefined input', async () => {
      const { parseAllowedHosts } = await loadModule()
      const result = parseAllowedHosts(undefined)
      expect(result.length).toBeGreaterThan(0)
    })
  })

  describe('matchesGlobPattern', () => {
    it('matches exact hostname', async () => {
      const { matchesGlobPattern } = await loadModule()
      expect(matchesGlobPattern('example.com', 'example.com')).toBe(true)
    })

    it('does not match different hostname', async () => {
      const { matchesGlobPattern } = await loadModule()
      expect(matchesGlobPattern('other.com', 'example.com')).toBe(false)
    })

    it('matches wildcard prefix', async () => {
      const { matchesGlobPattern } = await loadModule()
      expect(matchesGlobPattern('app.example.com', '*.example.com')).toBe(true)
    })

    it('matches deep subdomain with wildcard', async () => {
      const { matchesGlobPattern } = await loadModule()
      expect(matchesGlobPattern('deep.sub.example.com', '*.example.com')).toBe(true)
    })

    it('matches wildcard in the middle', async () => {
      const { matchesGlobPattern } = await loadModule()
      expect(matchesGlobPattern('app.apps.rosa.us-east-1.amazonaws.com', '*.apps.rosa.*.amazonaws.com')).toBe(true)
    })

    it('does not match if non-wildcard segments differ', async () => {
      const { matchesGlobPattern } = await loadModule()
      expect(matchesGlobPattern('app.apps.other.us-east-1.amazonaws.com', '*.apps.rosa.*.amazonaws.com')).toBe(false)
    })
  })
})
