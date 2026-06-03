import { describe, it, expect } from 'vitest'
import { formatRelativeTime, formatAbsoluteTime, formatDuration, formatPreciseDuration } from '../format-timestamp'

describe('formatRelativeTime', () => {
  it('returns a human-readable relative time string', () => {
    const fiveMinAgo = new Date(Date.now() - 5 * 60 * 1000).toISOString()
    const result = formatRelativeTime(fiveMinAgo)
    expect(result).toContain('ago')
  })
})

describe('formatAbsoluteTime', () => {
  it('returns a formatted date string', () => {
    const result = formatAbsoluteTime('2026-05-28T10:42:18Z')
    expect(result).toContain('2026')
    expect(result).toContain('28')
  })
})

describe('formatDuration', () => {
  it('returns duration between two timestamps', () => {
    const start = '2026-05-28T10:00:00Z'
    const end = '2026-05-28T10:30:00Z'
    const result = formatDuration(start, end)
    expect(result).toContain('30')
    expect(result).toContain('minute')
  })

  it('computes duration to now when no end time', () => {
    const recent = new Date(Date.now() - 60 * 1000).toISOString()
    const result = formatDuration(recent)
    expect(result).toBeTruthy()
  })
})

describe('formatPreciseDuration', () => {
  it('formats seconds only', () => {
    const start = '2026-05-28T10:00:00Z'
    const end = '2026-05-28T10:00:45Z'
    expect(formatPreciseDuration(start, end)).toBe('45s')
  })

  it('formats minutes and seconds', () => {
    const start = '2026-05-28T10:00:00Z'
    const end = '2026-05-28T10:05:30Z'
    expect(formatPreciseDuration(start, end)).toBe('5m 30s')
  })

  it('formats hours and minutes', () => {
    const start = '2026-05-28T10:00:00Z'
    const end = '2026-05-28T12:03:00Z'
    expect(formatPreciseDuration(start, end)).toBe('2h 3m')
  })

  it('formats days and hours', () => {
    const start = '2026-05-28T10:00:00Z'
    const end = '2026-05-30T14:00:00Z'
    expect(formatPreciseDuration(start, end)).toBe('2d 4h')
  })

  it('returns 0s for zero duration', () => {
    const ts = '2026-05-28T10:00:00Z'
    expect(formatPreciseDuration(ts, ts)).toBe('0s')
  })

  it('returns 0s when end is before start', () => {
    const start = '2026-05-28T12:00:00Z'
    const end = '2026-05-28T10:00:00Z'
    expect(formatPreciseDuration(start, end)).toBe('0s')
  })

  it('computes duration to now when no end time', () => {
    const recent = new Date(Date.now() - 90 * 1000).toISOString()
    const result = formatPreciseDuration(recent)
    expect(result).toMatch(/^1m \d+s$/)
  })
})
