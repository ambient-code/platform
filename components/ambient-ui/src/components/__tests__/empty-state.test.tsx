import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { EmptyState } from '../empty-state'
import { Inbox } from 'lucide-react'

describe('EmptyState', () => {
  it('renders title and description', () => {
    render(
      <EmptyState
        icon={Inbox}
        title="No sessions"
        description="Create a session to get started."
      />,
    )

    expect(screen.getByText('No sessions')).toBeInTheDocument()
    expect(
      screen.getByText('Create a session to get started.'),
    ).toBeInTheDocument()
  })

  it('renders action when provided', () => {
    render(
      <EmptyState
        icon={Inbox}
        title="Empty"
        description="Nothing here."
        action={<button type="button">Create</button>}
      />,
    )

    expect(screen.getByRole('button', { name: 'Create' })).toBeInTheDocument()
  })

  it('does not render action section when omitted', () => {
    const { container } = render(
      <EmptyState
        icon={Inbox}
        title="Empty"
        description="Nothing here."
      />,
    )

    // The action wrapper div should not be present
    const buttons = container.querySelectorAll('button')
    expect(buttons).toHaveLength(0)
  })
})
