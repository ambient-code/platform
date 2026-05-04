import { describe, it, expect, vi } from 'vitest'
import { createJiraAdapter } from '../jira'

describe('jiraAdapter', () => {
  it('delegates getJiraStatus to API', async () => {
    const fakeApi = {
      getJiraStatus: vi.fn().mockResolvedValue({ connected: true, url: 'https://jira.example.com', email: 'u@test.com' }),
      connectJira: vi.fn(),
      disconnectJira: vi.fn(),
    }
    const adapter = createJiraAdapter(fakeApi)

    const result = await adapter.getJiraStatus()

    expect(result.connected).toBe(true)
    expect(result.url).toBe('https://jira.example.com')
  })
})
