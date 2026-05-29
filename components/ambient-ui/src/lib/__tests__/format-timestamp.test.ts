import { describe, it, expect } from 'vitest'
import { formatRelativeTime, formatAbsoluteTime, formatDuration } from '../format-timestamp'

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
