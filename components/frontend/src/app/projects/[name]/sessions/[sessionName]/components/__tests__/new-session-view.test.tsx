import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { NewSessionView } from '../new-session-view';

vi.mock('../runner-model-selector', () => ({
  RunnerModelSelector: ({ onSelect }: { onSelect: (r: string, m: string) => void }) => (
    <button data-testid="runner-model-selector" onClick={() => onSelect('claude-agent-sdk', 'claude-sonnet-4-5')}>
      claude-agent-sdk · Claude Sonnet 4.5
    </button>
  ),
  getDefaultModel: () => 'claude-sonnet-4-5',
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

vi.mock('../workflow-selector', () => ({
  WorkflowSelector: () => <button data-testid="workflow-selector">No workflow</button>,
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
        model: 'claude-sonnet-4-5',
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
});
