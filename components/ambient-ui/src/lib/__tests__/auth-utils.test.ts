import { describe, it, expect } from 'vitest'
import { safeReturnTo } from '../auth-utils'

describe('safeReturnTo', () => {
  it('returns "/" for null input', () => {
    expect(safeReturnTo(null)).toBe('/')
  })

  it('returns "/" for undefined input', () => {
    expect(safeReturnTo(undefined)).toBe('/')
  })

  it('returns "/" for empty string', () => {
    expect(safeReturnTo('')).toBe('/')
  })

  it('returns the path for a valid relative URL', () => {
    expect(safeReturnTo('/dashboard')).toBe('/dashboard')
  })

  it('preserves query parameters on relative URLs', () => {
    expect(safeReturnTo('/fleet?tab=logs')).toBe('/fleet?tab=logs')
  })

  it('returns the path for a nested relative URL', () => {
    expect(safeReturnTo('/proj-123/fleet/sess-001')).toBe(
      '/proj-123/fleet/sess-001',
    )
  })

  it('rejects absolute URLs to foreign origins (open redirect prevention)', () => {
    expect(safeReturnTo('https://evil.example.com/phish')).toBe('/')
  })

  it('rejects protocol-relative URLs', () => {
    expect(safeReturnTo('//evil.example.com/phish')).toBe('/')
  })

  it('rejects javascript: protocol URLs', () => {
    // eslint-disable-next-line no-script-url
    expect(safeReturnTo('javascript:alert(1)')).toBe('/')
  })

  it('rejects data: protocol URLs', () => {
    expect(safeReturnTo('data:text/html,<h1>hi</h1>')).toBe('/')
  })

  it('strips hash fragments (only path + search)', () => {
    // URL constructor parses the hash into pathname on localhost
    const result = safeReturnTo('/page#section')
    expect(result).toBe('/page')
  })

  it('handles "/" as a valid return path', () => {
    expect(safeReturnTo('/')).toBe('/')
  })
})
