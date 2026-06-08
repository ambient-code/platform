import { describe, it, expect } from 'vitest'
import { queryKeys } from '../query-keys'

describe('queryKeys', () => {
  describe('sessions', () => {
    it('sessions.all is a stable base key', () => {
      expect(queryKeys.sessions.all).toEqual(['sessions'])
    })

    it('sessions.lists() nests under sessions.all', () => {
      const key = queryKeys.sessions.lists()
      expect(key).toEqual(['sessions', 'list'])
      expect(key.slice(0, 1)).toEqual(queryKeys.sessions.all)
    })

    it('sessions.list() includes projectId', () => {
      const key = queryKeys.sessions.list('proj-123')
      expect(key).toEqual(['sessions', 'list', 'proj-123', undefined])
    })

    it('sessions.list() includes params when provided', () => {
      const key = queryKeys.sessions.list('proj-123', { page: 2, size: 10 })
      expect(key).toEqual([
        'sessions',
        'list',
        'proj-123',
        { page: 2, size: 10 },
      ])
    })

    it('sessions.detail() includes sessionId', () => {
      const key = queryKeys.sessions.detail('sess-001')
      expect(key).toEqual(['sessions', 'detail', 'sess-001'])
    })

    it('different projectIds produce different keys', () => {
      const key1 = queryKeys.sessions.list('proj-a')
      const key2 = queryKeys.sessions.list('proj-b')
      expect(key1).not.toEqual(key2)
    })
  })

  describe('projects', () => {
    it('projects.all is a stable base key', () => {
      expect(queryKeys.projects.all).toEqual(['projects'])
    })

    it('projects.list() includes params', () => {
      const key = queryKeys.projects.list({ page: 1, size: 20 })
      expect(key).toEqual(['projects', 'list', { page: 1, size: 20 }])
    })

    it('projects.detail() includes projectId', () => {
      const key = queryKeys.projects.detail('proj-xyz')
      expect(key).toEqual(['projects', 'detail', 'proj-xyz'])
    })
  })

  describe('messages', () => {
    it('messages.all is a stable base key', () => {
      expect(queryKeys.messages.all).toEqual(['messages'])
    })

    it('messages.list() includes sessionId', () => {
      const key = queryKeys.messages.list('sess-001')
      expect(key).toEqual(['messages', 'list', 'sess-001'])
    })
  })
})
