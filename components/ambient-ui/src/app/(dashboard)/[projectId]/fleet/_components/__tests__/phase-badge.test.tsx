import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { PhaseBadge } from '../phase-badge'
import type { SessionPhase } from '@/domain/types'

describe('PhaseBadge', () => {
  it('renders the phase label for each valid phase', () => {
    const phases: SessionPhase[] = [
      'Pending',
      'Creating',
      'Running',
      'Stopping',
      'Completed',
      'Failed',
      'Stopped',
    ]

    for (const phase of phases) {
      const { unmount } = render(<PhaseBadge phase={phase} />)
      expect(screen.getByText(phase)).toBeInTheDocument()
      unmount()
    }
  })

  it('renders a pulse indicator for Running phase', () => {
    const { container } = render(<PhaseBadge phase="Running" />)
    const pulseElement = container.querySelector('.animate-ping')
    expect(pulseElement).toBeInTheDocument()
  })

  it('does not render a pulse indicator for terminal phases', () => {
    const terminalPhases: SessionPhase[] = ['Completed', 'Failed', 'Stopped']

    for (const phase of terminalPhases) {
      const { container, unmount } = render(<PhaseBadge phase={phase} />)
      const pulseElement = container.querySelector('.animate-ping')
      expect(pulseElement).not.toBeInTheDocument()
      unmount()
    }
  })
})
