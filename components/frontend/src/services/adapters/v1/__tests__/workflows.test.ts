import { describe, it, expect, vi } from 'vitest'
import { createWorkflowsAdapter } from '../workflows'

describe('workflowsAdapter', () => {
  it('delegates listOOTBWorkflows to API', async () => {
    const fakeApi = {
      listOOTBWorkflows: vi.fn().mockResolvedValue([{ id: 'wf-1', name: 'Review', description: 'Code review', gitUrl: 'https://git', branch: 'main', enabled: true }]),
      getWorkflowMetadata: vi.fn(),
    }
    const adapter = createWorkflowsAdapter(fakeApi)

    const result = await adapter.listOOTBWorkflows('p')

    expect(result).toHaveLength(1)
    expect(result[0].name).toBe('Review')
  })

  it('delegates getWorkflowMetadata to API', async () => {
    const fakeApi = {
      listOOTBWorkflows: vi.fn(),
      getWorkflowMetadata: vi.fn().mockResolvedValue({ commands: [], agents: [] }),
    }
    const adapter = createWorkflowsAdapter(fakeApi)

    const result = await adapter.getWorkflowMetadata('p', 's')

    expect(result.commands).toEqual([])
  })
})
