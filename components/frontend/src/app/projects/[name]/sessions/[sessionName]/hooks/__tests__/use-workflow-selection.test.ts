import { describe, it, expect, vi } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useWorkflowSelection } from '../use-workflow-selection';
import type { WorkflowConfig } from '../../lib/types';

const mockWorkflows: WorkflowConfig[] = [
  { id: 'wf-1', name: 'Code Review', description: 'Automated code review workflow', enabled: true },
  { id: 'wf-2', name: 'Bug Fix', description: 'Fix bugs automatically', enabled: true },
  { id: 'wf-3', name: 'Documentation', description: 'Generate docs from code', enabled: true },
] as WorkflowConfig[];

function renderWorkflowHook(overrides: Partial<Parameters<typeof useWorkflowSelection>[0]> = {}) {
  const defaultProps = {
    selectedWorkflow: 'none',
    ootbWorkflows: mockWorkflows,
    workflowActivating: false,
    onWorkflowChange: vi.fn(),
    ...overrides,
  };
  return { ...renderHook(() => useWorkflowSelection(defaultProps)), props: defaultProps };
}

describe('useWorkflowSelection', () => {
  it('returns all workflows when no search is active', () => {
    const { result } = renderWorkflowHook();
    expect(result.current.filteredWorkflows).toHaveLength(3);
  });

  it('sorts workflows alphabetically by name', () => {
    const { result } = renderWorkflowHook();
    const names = result.current.filteredWorkflows.map((w) => w.name);
    expect(names).toEqual(['Bug Fix', 'Code Review', 'Documentation']);
  });

  it('filters workflows by name', () => {
    const { result } = renderWorkflowHook();
    act(() => {
      result.current.setSearch('bug fix');
    });
    expect(result.current.filteredWorkflows).toHaveLength(1);
    expect(result.current.filteredWorkflows[0].id).toBe('wf-2');
  });

  it('filters workflows by description', () => {
    const { result } = renderWorkflowHook();
    act(() => {
      result.current.setSearch('bugs');
    });
    expect(result.current.filteredWorkflows).toHaveLength(1);
    expect(result.current.filteredWorkflows[0].id).toBe('wf-2');
  });

  it('filter is case-insensitive', () => {
    const { result } = renderWorkflowHook();
    act(() => {
      result.current.setSearch('CODE REVIEW');
    });
    expect(result.current.filteredWorkflows).toHaveLength(1);
  });

  it('returns empty list when nothing matches', () => {
    const { result } = renderWorkflowHook();
    act(() => {
      result.current.setSearch('zzzzz');
    });
    expect(result.current.filteredWorkflows).toHaveLength(0);
  });

  it('selectedLabel returns "No workflow" when selectedWorkflow is "none"', () => {
    const { result } = renderWorkflowHook({ selectedWorkflow: 'none' });
    expect(result.current.selectedLabel).toBe('No workflow');
  });

  it('selectedLabel returns "Custom workflow" when selectedWorkflow is "custom"', () => {
    const { result } = renderWorkflowHook({ selectedWorkflow: 'custom' });
    expect(result.current.selectedLabel).toBe('Custom workflow');
  });

  it('selectedLabel returns workflow name when a workflow id is selected', () => {
    const { result } = renderWorkflowHook({ selectedWorkflow: 'wf-2' });
    expect(result.current.selectedLabel).toBe('Bug Fix');
  });

  it('selectedLabel falls back to "No workflow" for unknown id', () => {
    const { result } = renderWorkflowHook({ selectedWorkflow: 'unknown-id' });
    expect(result.current.selectedLabel).toBe('No workflow');
  });

  it('handleSelect calls onWorkflowChange and closes popover', () => {
    const onWorkflowChange = vi.fn();
    const { result } = renderWorkflowHook({ onWorkflowChange });

    act(() => {
      result.current.handleOpenChange(true);
    });
    expect(result.current.popoverOpen).toBe(true);

    act(() => {
      result.current.handleSelect('wf-1');
    });
    expect(onWorkflowChange).toHaveBeenCalledWith('wf-1');
    expect(result.current.popoverOpen).toBe(false);
  });

  it('isActivating reflects workflowActivating prop', () => {
    const { result: resultFalse } = renderWorkflowHook({ workflowActivating: false });
    expect(resultFalse.current.isActivating).toBe(false);

    const { result: resultTrue } = renderWorkflowHook({ workflowActivating: true });
    expect(resultTrue.current.isActivating).toBe(true);
  });

  it('handleOpenChange resets search and opens popover', () => {
    const { result } = renderWorkflowHook();

    act(() => {
      result.current.setSearch('something');
    });
    expect(result.current.search).toBe('something');

    act(() => {
      result.current.handleOpenChange(true);
    });
    expect(result.current.popoverOpen).toBe(true);
    expect(result.current.search).toBe('');
  });

  it('showGeneralChat is true when search is empty', () => {
    const { result } = renderWorkflowHook();
    expect(result.current.showGeneralChat).toBe(true);
  });

  it('showGeneralChat matches on "general" search', () => {
    const { result } = renderWorkflowHook();
    act(() => {
      result.current.setSearch('general');
    });
    expect(result.current.showGeneralChat).toBe(true);
  });

  it('showGeneralChat is false for unrelated search', () => {
    const { result } = renderWorkflowHook();
    act(() => {
      result.current.setSearch('zzzzz');
    });
    expect(result.current.showGeneralChat).toBe(false);
  });

  it('showCustomWorkflow is true when search is empty', () => {
    const { result } = renderWorkflowHook();
    expect(result.current.showCustomWorkflow).toBe(true);
  });

  it('showCustomWorkflow matches on "custom" search', () => {
    const { result } = renderWorkflowHook();
    act(() => {
      result.current.setSearch('custom');
    });
    expect(result.current.showCustomWorkflow).toBe(true);
  });
});
