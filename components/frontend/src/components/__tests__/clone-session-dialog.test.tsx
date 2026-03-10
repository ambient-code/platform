import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { CloneSessionDialog } from '../clone-session-dialog';
import type { AgenticSession } from '@/types/agentic-session';

const mockCloneMutate = vi.fn();

vi.mock('@/services/queries', () => ({
  useProjects: vi.fn(() => ({
    data: [
      { name: 'project-a', displayName: 'Project A' },
      { name: 'project-b', displayName: 'Project B' },
    ],
    isLoading: false,
  })),
  useCloneSession: vi.fn(() => ({
    mutate: mockCloneMutate,
    isPending: false,
  })),
}));

function makeSession(): AgenticSession {
  return {
    metadata: { name: 'my-session', namespace: 'default', uid: '456', creationTimestamp: '' },
    spec: {
      displayName: 'My Session',
      initialPrompt: 'test',
      project: 'project-a',
      llmSettings: { model: 'test', temperature: 0, maxTokens: 100 },
      timeout: 3600,
    },
    status: { phase: 'Completed' },
  } as AgenticSession;
}

describe('CloneSessionDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('opens dialog when trigger is clicked', () => {
    render(
      <CloneSessionDialog
        session={makeSession()}
        trigger={<button>Open Clone</button>}
      />
    );

    fireEvent.click(screen.getByText('Open Clone'));
    expect(screen.getByText(/Clone "My Session" to a target project/)).toBeDefined();
  });

  it('shows project selector when projectName is not provided', () => {
    render(
      <CloneSessionDialog
        session={makeSession()}
        trigger={<button>Open Clone</button>}
      />
    );

    fireEvent.click(screen.getByText('Open Clone'));
    expect(screen.getByText('Target Project')).toBeDefined();
  });

  it('hides project selector when projectName is provided', () => {
    render(
      <CloneSessionDialog
        session={makeSession()}
        trigger={<button>Open Clone</button>}
        projectName="project-a"
      />
    );

    fireEvent.click(screen.getByText('Open Clone'));
    expect(screen.queryByText('Target Project')).toBeNull();
    expect(
      screen.getByText(/Clone "My Session" into this project/)
    ).toBeDefined();
  });

  it('submits clone with provided projectName', async () => {
    render(
      <CloneSessionDialog
        session={makeSession()}
        trigger={<button>Open Clone</button>}
        projectName="project-a"
      />
    );

    fireEvent.click(screen.getByText('Open Clone'));

    const submitBtn = screen.getByRole('button', { name: 'Clone Session' });
    fireEvent.click(submitBtn);

    await waitFor(() => {
      expect(mockCloneMutate).toHaveBeenCalledWith(
        expect.objectContaining({
          projectName: 'project-a',
          sessionName: 'my-session',
          data: expect.objectContaining({
            targetProject: 'project-a',
          }),
        }),
        expect.any(Object)
      );
    });
  });

  it('closes dialog on cancel', async () => {
    render(
      <CloneSessionDialog
        session={makeSession()}
        trigger={<button>Open Clone</button>}
        projectName="project-a"
      />
    );

    fireEvent.click(screen.getByText('Open Clone'));
    expect(screen.getByText(/Clone "My Session" into this project/)).toBeDefined();

    fireEvent.click(screen.getByText('Cancel'));
    await waitFor(() => {
      expect(screen.queryByText(/Clone "My Session"/)).toBeNull();
    });
  });
});
