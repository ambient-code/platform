import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { EventTypeBadge } from '../event-type-badge'

describe('EventTypeBadge', () => {
  it('renders the correct label for each known event type', () => {
    const expectedLabels: Record<string, string> = {
      user: 'User',
      assistant: 'Assistant',
      text: 'Text',
      tool_use: 'Tool Call',
      tool_result: 'Tool Result',
      error: 'Error',
      lifecycle: 'Lifecycle',
      user_feedback: 'Feedback',
      system: 'System',
    }

    for (const [eventType, label] of Object.entries(expectedLabels)) {
      const { unmount } = render(<EventTypeBadge eventType={eventType} />)
      expect(screen.getByText(label)).toBeInTheDocument()
      unmount()
    }
  })

  it('falls back to "System" label for unknown event types', () => {
    render(<EventTypeBadge eventType="some_unknown_event" />)
    expect(screen.getByText('System')).toBeInTheDocument()
  })

  it('falls back to "System" for empty string event type', () => {
    render(<EventTypeBadge eventType="" />)
    expect(screen.getByText('System')).toBeInTheDocument()
  })
})
