import { describe, it, expect, vi, afterEach } from 'vitest'

describe('preview-proxy', () => {
  afterEach(() => {
    vi.unstubAllEnvs()
    vi.resetModules()
  })

  async function loadModule() {
    return await import('../preview-proxy')
  }

  describe('validatePreviewUrl', () => {
    it('returns valid with parsed URL for http URL', async () => {
      vi.stubEnv('NEXT_PUBLIC_PREVIEW_ALLOWED_HOSTS', 'localhost:*')
      const { validatePreviewUrl } = await loadModule()
      const result = validatePreviewUrl('http://localhost:3000/page')
      expect(result.valid).toBe(true)
      if (result.valid) {
        expect(result.parsed).toBeInstanceOf(URL)
        expect(result.parsed.hostname).toBe('localhost')
      }
    })

    it('returns valid with parsed URL for https URL', async () => {
      vi.stubEnv('NEXT_PUBLIC_PREVIEW_ALLOWED_HOSTS', '*.example.com')
      const { validatePreviewUrl } = await loadModule()
      const result = validatePreviewUrl('https://app.example.com/dashboard')
      expect(result.valid).toBe(true)
      if (result.valid) {
        expect(result.parsed).toBeInstanceOf(URL)
        expect(result.parsed.hostname).toBe('app.example.com')
      }
    })

    it('returns invalid for a non-parseable URL', async () => {
      const { validatePreviewUrl } = await loadModule()
      const result = validatePreviewUrl('not a url at all')
      expect(result.valid).toBe(false)
      if (!result.valid) {
        expect(result.reason).toBeDefined()
      }
    })

    it('returns invalid for javascript: protocol', async () => {
      const { validatePreviewUrl } = await loadModule()
      // eslint-disable-next-line no-script-url
      const result = validatePreviewUrl('javascript:alert(1)')
      expect(result.valid).toBe(false)
      if (!result.valid) {
        expect(result.reason).toBeDefined()
      }
    })

    it('returns invalid for ftp: protocol', async () => {
      const { validatePreviewUrl } = await loadModule()
      const result = validatePreviewUrl('ftp://files.example.com/readme')
      expect(result.valid).toBe(false)
      if (!result.valid) {
        expect(result.reason).toBeDefined()
      }
    })

    it('returns invalid with allowlist reason for URL not on allowlist', async () => {
      vi.stubEnv('NEXT_PUBLIC_PREVIEW_ALLOWED_HOSTS', 'localhost:*')
      const { validatePreviewUrl } = await loadModule()
      const result = validatePreviewUrl('https://evil.example.com')
      expect(result.valid).toBe(false)
      if (!result.valid) {
        expect(result.reason).toContain('not on the trusted preview hosts allowlist')
      }
    })

    it('returns valid for URL on allowlist', async () => {
      vi.stubEnv('NEXT_PUBLIC_PREVIEW_ALLOWED_HOSTS', '*.example.com')
      const { validatePreviewUrl } = await loadModule()
      const result = validatePreviewUrl('https://app.example.com/page')
      expect(result.valid).toBe(true)
    })
  })

  describe('stripFrameBlockingHeaders', () => {
    it('removes X-Frame-Options header', async () => {
      const { stripFrameBlockingHeaders } = await loadModule()
      const headers = new Headers({
        'X-Frame-Options': 'DENY',
        'Content-Type': 'text/html',
      })
      const result = stripFrameBlockingHeaders(headers)
      expect(result.has('x-frame-options')).toBe(false)
      expect(result.get('content-type')).toBe('text/html')
    })

    it('removes Content-Security-Policy-Report-Only header', async () => {
      const { stripFrameBlockingHeaders } = await loadModule()
      const headers = new Headers({
        'Content-Security-Policy-Report-Only': "default-src 'self'",
        'Content-Type': 'text/html',
      })
      const result = stripFrameBlockingHeaders(headers)
      expect(result.has('content-security-policy-report-only')).toBe(false)
      expect(result.get('content-type')).toBe('text/html')
    })

    it('strips frame-ancestors from CSP while preserving other directives', async () => {
      const { stripFrameBlockingHeaders } = await loadModule()
      const headers = new Headers({
        'Content-Security-Policy':
          "default-src 'self'; frame-ancestors 'none'; script-src 'unsafe-inline'",
      })
      const result = stripFrameBlockingHeaders(headers)
      const csp = result.get('content-security-policy')
      expect(csp).not.toContain('frame-ancestors')
      expect(csp).toContain("default-src 'self'")
      expect(csp).toContain("script-src 'unsafe-inline'")
    })

    it('preserves non-frame headers unchanged', async () => {
      const { stripFrameBlockingHeaders } = await loadModule()
      const headers = new Headers({
        'Content-Type': 'application/json',
        'Cache-Control': 'no-cache',
      })
      const result = stripFrameBlockingHeaders(headers)
      expect(result.get('content-type')).toBe('application/json')
      expect(result.get('cache-control')).toBe('no-cache')
    })

    it('deletes CSP header entirely when only frame-ancestors is present', async () => {
      const { stripFrameBlockingHeaders } = await loadModule()
      const headers = new Headers({
        'Content-Security-Policy': "frame-ancestors 'self'",
      })
      const result = stripFrameBlockingHeaders(headers)
      expect(result.has('content-security-policy')).toBe(false)
    })

    it('handles missing headers gracefully (no-op)', async () => {
      const { stripFrameBlockingHeaders } = await loadModule()
      const headers = new Headers({
        'Content-Type': 'text/plain',
      })
      const result = stripFrameBlockingHeaders(headers)
      expect(result.get('content-type')).toBe('text/plain')
      expect(result.has('x-frame-options')).toBe(false)
      expect(result.has('content-security-policy')).toBe(false)
    })
  })

  describe('injectBaseTag', () => {
    it('injects base tag and interceptor script after <head>', async () => {
      const { injectBaseTag } = await loadModule()
      const html = '<html><head><title>Test</title></head></html>'
      const result = injectBaseTag(html, 'https://app.example.com/')
      expect(result).toContain('<base href="https://app.example.com/">')
      expect(result).toContain('<script>')
      expect(result).toContain('/api/preview-proxy?url=')
      expect(result.indexOf('<head>')).toBeLessThan(result.indexOf('<base'))
    })

    it('injects after <head> with attributes', async () => {
      const { injectBaseTag } = await loadModule()
      const html = '<html><head lang="en"><title>Test</title></head></html>'
      const result = injectBaseTag(html, 'https://app.example.com/')
      expect(result).toContain('<base href="https://app.example.com/">')
      expect(result).toContain('<script>')
      expect(result.indexOf('<head lang="en">')).toBeLessThan(result.indexOf('<base'))
    })

    it('prepends when no <head> tag is present', async () => {
      const { injectBaseTag } = await loadModule()
      const html = '<div>Content</div>'
      const result = injectBaseTag(html, 'https://app.example.com/')
      expect(result).toContain('<base href="https://app.example.com/">')
      expect(result).toContain('<div>Content</div>')
      expect(result.indexOf('<base')).toBe(0)
    })

    it('skips base tag but still injects script when <base already exists', async () => {
      const { injectBaseTag } = await loadModule()
      const html = '<html><head><base href="/existing/"><title>Test</title></head></html>'
      const result = injectBaseTag(html, 'https://app.example.com/')
      expect(result).toContain('<script>')
      expect(result).not.toContain('<base href="https://app.example.com/">')
      expect(result).toContain('<base href="/existing/">')
    })

    it('skips base tag case-insensitively', async () => {
      const { injectBaseTag } = await loadModule()
      const html = '<html><head><BASE href="/existing/"><title>Test</title></head></html>'
      const result = injectBaseTag(html, 'https://app.example.com/')
      expect(result).toContain('<script>')
      expect(result).not.toContain('<base href="https://app.example.com/">')
    })
  })

  describe('buildBaseHref', () => {
    it('returns path as-is when it ends with /', async () => {
      const { buildBaseHref } = await loadModule()
      const url = new URL('https://app.example.com/dashboard/')
      expect(buildBaseHref(url)).toBe('https://app.example.com/dashboard/')
    })

    it('trims to last / when path does not end with /', async () => {
      const { buildBaseHref } = await loadModule()
      const url = new URL('https://app.example.com/dashboard/page')
      expect(buildBaseHref(url)).toBe('https://app.example.com/dashboard/')
    })

    it('returns origin with trailing / for root path', async () => {
      const { buildBaseHref } = await loadModule()
      const url = new URL('https://app.example.com/')
      expect(buildBaseHref(url)).toBe('https://app.example.com/')
    })

    it('returns origin with trailing / when no path', async () => {
      const { buildBaseHref } = await loadModule()
      const url = new URL('https://app.example.com')
      expect(buildBaseHref(url)).toBe('https://app.example.com/')
    })
  })
})
