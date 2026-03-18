import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { WorkflowSelector } from '../workflow-selector';
import type { WorkflowConfig } from '@/types/workflow';

vi.mock('../../hooks/use-workflow-selection', () => ({
  useWorkflowSelection: vi.fn(),
}));

import { useWorkflowSelection } from '../../hooks/use-workflow-selection';

const mockUseWorkflowSelection = vi.mocked(useWorkflowSelection);

const sampleWorkflows: WorkflowConfig[] = [
  {
    id: 'wf-1',
    name: 'Code Review',
    description: 'Review pull requests',
    gitUrl: 'https://github.com/example/repo',
    branch: 'main',
    enabled: true,
  },
  {
    id: 'wf-2',
    name: 'Bug Fix',
    description: 'Fix bugs in the codebase',
    gitUrl: 'https://github.com/example/repo',
    branch: 'main',
    enabled: true,
  },
];

function setupMock(overrides: Partial<ReturnType<typeof useWorkflowSelection>> = {}) {
  mockUseWorkflowSelection.mockReturnValue({
    search: '',
    setSearch: vi.fn(),
    popoverOpen: false,
    searchInputRef: { current: null },
    filteredWorkflows: sampleWorkflows,
    showGeneralChat: true,
    showCustomWorkflow: true,
    selectedLabel: 'No workflow',
    isActivating: false,
    handleSelect: vi.fn(),
    handleOpenChange: vi.fn(),
    ...overrides,
  });
}

const defaultProps = {
  sessionPhase: 'Running',
  activeWorkflow: null,
  selectedWorkflow: 'none',
  workflowActivating: false,
  ootbWorkflows: sampleWorkflows,
  onWorkflowChange: vi.fn(),
};

describe('WorkflowSelector', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders button with workflow label', () => {
    setupMock({ selectedLabel: 'No workflow' });
    render(<WorkflowSelector {...defaultProps} />);

    expect(screen.getByText('No workflow')).toBeDefined();
  });

  it('button is disabled when session is Stopped', () => {
    setupMock();
    render(<WorkflowSelector {...defaultProps} sessionPhase="Stopped" />);

    const button = screen.getByRole('button');
    expect(button.hasAttribute('disabled')).toBe(true);
  });

  it('shows "Switching..." when workflowActivating is true', () => {
    setupMock({ isActivating: true });
    render(
      <WorkflowSelector
        {...defaultProps}
        workflowActivating={true}
      />
    );

    expect(screen.getByText('Switching...')).toBeDefined();
  });

  it('renders correct display label from activeWorkflow', () => {
    setupMock({ selectedLabel: 'Code Review' });
    render(
      <WorkflowSelector
        {...defaultProps}
        activeWorkflow="wf-1"
      />
    );

    // When activeWorkflow is set, the component looks up the name from ootbWorkflows
    expect(screen.getByText('Code Review')).toBeDefined();
  });
});
