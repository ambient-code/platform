import { describe, it, expect } from 'vitest'
import { getPhaseStyle } from '../status-colors'
import type { SessionPhase } from '@/domain/types'

describe('getPhaseStyle', () => {
  it('returns success with pulse for Running', () => {
    const style = getPhaseStyle('Running')
    expect(style.variant).toBe('success')
    expect(style.pulse).toBe(true)
  })

  it('returns info for Completed', () => {
    const style = getPhaseStyle('Completed')
    expect(style.variant).toBe('info')
    expect(style.pulse).toBe(false)
  })

  it('returns error for Failed', () => {
    const style = getPhaseStyle('Failed')
    expect(style.variant).toBe('error')
    expect(style.pulse).toBe(false)
  })

  it('returns default for Stopped', () => {
    const style = getPhaseStyle('Stopped')
    expect(style.variant).toBe('default')
    expect(style.pulse).toBe(false)
  })

  it('returns warning for transitioning phases', () => {
    const transitioning: SessionPhase[] = ['Pending', 'Creating', 'Stopping']
    for (const phase of transitioning) {
      const style = getPhaseStyle(phase)
      expect(style.variant).toBe('warning')
      expect(style.pulse).toBe(false)
    }
  })

  it('preserves phase name as label', () => {
    const phases: SessionPhase[] = ['Pending', 'Creating', 'Running', 'Stopping', 'Completed', 'Failed', 'Stopped']
    for (const phase of phases) {
      expect(getPhaseStyle(phase).label).toBe(phase)
    }
  })
})
