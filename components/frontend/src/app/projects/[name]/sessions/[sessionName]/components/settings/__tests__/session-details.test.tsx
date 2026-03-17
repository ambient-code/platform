import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { SessionDetails } from '../session-details';
import type { AgenticSession } from '@/types/agentic-session';

vi.mock('@/components/status-badge', () => ({
  SessionPhaseBadge: ({ phase }: { phase: string }) => <span>{phase}</span>,
}));

vi.mock('date-fns', () => ({
  formatDistanceToNow: () => '3 hours ago',
}));

function makeSession(): AgenticSession {
  return {
    metadata: {
      name: 'test-session',
      namespace: 'default',
      uid: '123',
      creationTimestamp: '2026-01-01T00:00:00Z',
    },
    spec: {
      displayName: 'Test Session',
      initialPrompt: 'test',
      llmSettings: { model: 'claude-sonnet-4-20250514', temperature: 0, maxTokens: 100 },
      timeout: 3600,
    },
    status: { phase: 'Running' },
  };
}

describe('SessionDetails', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders Session Details heading', () => {
    render(<SessionDetails session={makeSession()} />);
    expect(screen.getByText('Session Details')).toBeDefined();
  });

  it('renders session name', () => {
    render(<SessionDetails session={makeSession()} />);
    expect(screen.getByText('Test Session')).toBeDefined();
  });

  it('renders session ID', () => {
    render(<SessionDetails session={makeSession()} />);
    expect(screen.getByText('test-session')).toBeDefined();
  });

  it('renders model', () => {
    render(<SessionDetails session={makeSession()} />);
    expect(screen.getByText('claude-sonnet-4-20250514')).toBeDefined();
  });

  it('renders edit button when onEditName provided', () => {
    const onEditName = vi.fn();
    render(<SessionDetails session={makeSession()} onEditName={onEditName} />);
    const buttons = screen.getAllByRole('button');
    expect(buttons.length).toBeGreaterThan(0);
  });

  it('does not render edit button when onEditName not provided', () => {
    render(<SessionDetails session={makeSession()} />);
    const buttons = screen.queryAllByRole('button');
    expect(buttons.length).toBe(0);
  });
});
