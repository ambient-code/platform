import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { NewSessionView } from '../new-session-view';

vi.mock('../runner-model-selector', () => ({
  RunnerModelSelector: ({ onSelect }: { onSelect: (r: string, m: string) => void }) => (
    <button data-testid="runner-model-selector" onClick={() => onSelect('claude-agent-sdk', 'claude-sonnet-4-6')}>
      claude-agent-sdk · Claude Sonnet 4.6
    </button>
  ),
  getDefaultModel: () => 'claude-sonnet-4-6',
}));

vi.mock('@/services/queries/use-runner-types', () => ({
  useRunnerTypes: () => ({
    data: [
      { id: 'claude-agent-sdk', displayName: 'Claude Agent SDK', description: '', framework: '', provider: 'anthropic', auth: { requiredSecretKeys: [], secretKeyLogic: 'any', vertexSupported: false } },
    ],
  }),
}));

vi.mock('@/services/api/runner-types', () => ({
  DEFAULT_RUNNER_TYPE_ID: 'claude-agent-sdk',
}));

vi.mock('@/services/queries/use-models', () => ({
  useModels: () => ({
    data: {
      models: [
        { id: 'claude-sonnet-4-5', label: 'Claude Sonnet 4.5', provider: 'anthropic', isDefault: false },
        { id: 'claude-sonnet-4-6', label: 'Claude Sonnet 4.6', provider: 'anthropic', isDefault: true },
      ],
      defaultModel: 'claude-sonnet-4-6',
    },
    isLoading: false,
  }),
}));

vi.mock('../workflow-selector', () => ({
  WorkflowSelector: () => <button data-testid="workflow-selector">No workflow</button>,
}));

vi.mock('../modals/add-context-modal', () => ({
  AddContextModal: ({ onAddRepository }: { open: boolean; onAddRepository: (url: string, branch: string, autoPush?: boolean) => Promise<void> }) => (
    <>
      <span data-testid="add-repo-btn" role="none" onClick={() => onAddRepository('https://github.com/org/platform.git', '')}>
        Add repo
      </span>
      <span data-testid="add-repo-with-branch-btn" role="none" onClick={() => onAddRepository('https://github.com/org/other.git', 'develop', true)}>
        Add repo with branch
      </span>
    </>
  ),
}));

describe('NewSessionView', () => {
  const defaultProps = {
    projectName: 'test-project',
    onCreateSession: vi.fn(),
    ootbWorkflows: [],
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders heading and subtitle', () => {
    render(<NewSessionView {...defaultProps} />);
    expect(screen.getByText('What are you working on?')).toBeDefined();
    expect(screen.getByText(/Start a new session/)).toBeDefined();
  });

  it('renders textarea with placeholder', () => {
    render(<NewSessionView {...defaultProps} />);
    const textarea = screen.getByPlaceholderText("Describe what you'd like to work on...");
    expect(textarea).toBeDefined();
  });

  it('renders runner/model selector and workflow selector', () => {
    render(<NewSessionView {...defaultProps} />);
    expect(screen.getByTestId('runner-model-selector')).toBeDefined();
    expect(screen.getByTestId('workflow-selector')).toBeDefined();
  });

  it('send button is disabled when textarea is empty', () => {
    render(<NewSessionView {...defaultProps} />);
    const allButtons = screen.getAllByRole('button');
    const lastButton = allButtons[allButtons.length - 1];
    expect(lastButton.hasAttribute('disabled')).toBe(true);
  });

  it('calls onCreateSession with prompt when submitted', () => {
    render(<NewSessionView {...defaultProps} />);
    const textarea = screen.getByPlaceholderText("Describe what you'd like to work on...");
    fireEvent.change(textarea, { target: { value: 'Build a REST API' } });
    fireEvent.keyDown(textarea, { key: 'Enter', shiftKey: false });
    expect(defaultProps.onCreateSession).toHaveBeenCalledWith(
      expect.objectContaining({
        prompt: 'Build a REST API',
        runner: 'claude-agent-sdk',
        model: 'claude-sonnet-4-6',
      })
    );
  });

  it('does not submit when prompt is empty', () => {
    render(<NewSessionView {...defaultProps} />);
    const textarea = screen.getByPlaceholderText("Describe what you'd like to work on...");
    fireEvent.keyDown(textarea, { key: 'Enter', shiftKey: false });
    expect(defaultProps.onCreateSession).not.toHaveBeenCalled();
  });

  it('Shift+Enter does not submit (allows newline)', () => {
    render(<NewSessionView {...defaultProps} />);
    const textarea = screen.getByPlaceholderText("Describe what you'd like to work on...");
    fireEvent.change(textarea, { target: { value: 'some text' } });
    fireEvent.keyDown(textarea, { key: 'Enter', shiftKey: true });
    expect(defaultProps.onCreateSession).not.toHaveBeenCalled();
  });

  it('includes branch and autoPush in onCreateSession when repo is added with branch', () => {
    render(<NewSessionView {...defaultProps} />);

    // Add a repo with branch via the mock AddContextModal
    fireEvent.click(screen.getByTestId('add-repo-with-branch-btn'));

    // Type a prompt and submit
    const textarea = screen.getByPlaceholderText("Describe what you'd like to work on...");
    fireEvent.change(textarea, { target: { value: 'Fix a bug' } });
    fireEvent.keyDown(textarea, { key: 'Enter', shiftKey: false });

    expect(defaultProps.onCreateSession).toHaveBeenCalledWith(
      expect.objectContaining({
        prompt: 'Fix a bug',
        repos: [
          { url: 'https://github.com/org/other.git', branch: 'develop', autoPush: true },
        ],
      })
    );
  });

  it('omits branch from repos when no branch is specified', () => {
    render(<NewSessionView {...defaultProps} />);

    // Add a repo without branch
    fireEvent.click(screen.getByTestId('add-repo-btn'));

    const textarea = screen.getByPlaceholderText("Describe what you'd like to work on...");
    fireEvent.change(textarea, { target: { value: 'Fix a bug' } });
    fireEvent.keyDown(textarea, { key: 'Enter', shiftKey: false });

    const call = defaultProps.onCreateSession.mock.calls[0][0];
    expect(call.repos).toHaveLength(1);
    expect(call.repos[0].url).toBe('https://github.com/org/platform.git');
    expect(call.repos[0].branch).toBeUndefined();
  });

  it('removes a pending repo badge when the X button is clicked', () => {
    render(<NewSessionView {...defaultProps} />);

    // Add a repo via the always-rendered mock AddContextModal
    fireEvent.click(screen.getByTestId('add-repo-btn'));

    // Badge should appear with the repo name derived from URL
    expect(screen.getByText('platform')).toBeDefined();

    // Click the remove button
    const removeBtn = screen.getByRole('button', { name: /Remove platform/i });
    fireEvent.click(removeBtn);

    // Badge should be gone
    expect(screen.queryByText('platform')).toBeNull();
  });
});
