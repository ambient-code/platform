import { describe, it, expect, vi } from 'vitest'
import { createSessionTasksAdapter } from '../session-tasks'

describe('sessionTasksAdapter', () => {
  it('delegates stopTask to API', async () => {
    const fakeApi = {
      stopBackgroundTask: vi.fn().mockResolvedValue({ message: 'stopped' }),
      getTaskOutput: vi.fn(),
    }
    const adapter = createSessionTasksAdapter(fakeApi)

    const result = await adapter.stopTask('p', 's', 'task-1')

    expect(result.message).toBe('stopped')
    expect(fakeApi.stopBackgroundTask).toHaveBeenCalledWith('p', 's', 'task-1')
  })

  it('delegates getTaskOutput to API', async () => {
    const fakeApi = {
      stopBackgroundTask: vi.fn(),
      getTaskOutput: vi.fn().mockResolvedValue({ task_id: 'task-1', output: [{ line: 'hello' }] }),
    }
    const adapter = createSessionTasksAdapter(fakeApi)

    const result = await adapter.getTaskOutput('p', 's', 'task-1')

    expect(result.task_id).toBe('task-1')
    expect(result.output).toHaveLength(1)
  })
})
