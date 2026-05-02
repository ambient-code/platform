import { describe, it, expect, vi, beforeEach } from 'vitest'
import { createSessionEventsAdapter } from '../session-events'

const fakeEventSource = {} as EventSource

const fakeApi = {
  createEventSource: vi.fn().mockReturnValue(fakeEventSource),
  sendMessage: vi.fn().mockResolvedValue({ runId: 'run-abc' }),
  interrupt: vi.fn().mockResolvedValue(undefined),
}

describe('sessionEventsAdapter', () => {
  const adapter = createSessionEventsAdapter(fakeApi)

  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('createEventSource', () => {
    it('delegates to api.createEventSource', () => {
      const result = adapter.createEventSource('my-project', 'my-session')
      expect(fakeApi.createEventSource).toHaveBeenCalledWith('my-project', 'my-session')
      expect(result).toBe(fakeEventSource)
    })

    it('forwards runId parameter', () => {
      adapter.createEventSource('proj', 'sess', 'run-123')
      expect(fakeApi.createEventSource).toHaveBeenCalledWith('proj', 'sess', 'run-123')
    })
  })

  describe('sendMessage', () => {
    it('delegates to api.sendMessage', async () => {
      const payload = {
        threadId: 'thread-1',
        messages: [{ id: '1', role: 'user' as const, content: 'Hello' }],
        tools: [],
      }

      const result = await adapter.sendMessage('proj', 'sess', payload)
      expect(fakeApi.sendMessage).toHaveBeenCalledWith('proj', 'sess', payload)
      expect(result).toEqual({ runId: 'run-abc' })
    })

    it('propagates errors from api', async () => {
      fakeApi.sendMessage.mockRejectedValueOnce(new Error('Server error'))

      const payload = {
        threadId: 'thread-1',
        messages: [{ id: '1', role: 'user' as const, content: 'Hello' }],
        tools: [],
      }

      await expect(adapter.sendMessage('proj', 'sess', payload)).rejects.toThrow('Server error')
    })
  })

  describe('interrupt', () => {
    it('delegates to api.interrupt', async () => {
      await adapter.interrupt('proj', 'sess', 'run-123')
      expect(fakeApi.interrupt).toHaveBeenCalledWith('proj', 'sess', 'run-123')
    })

    it('propagates errors from api', async () => {
      fakeApi.interrupt.mockRejectedValueOnce(new Error('Failed to interrupt'))

      await expect(adapter.interrupt('proj', 'sess', 'run-123')).rejects.toThrow('Failed to interrupt')
    })
  })
})
