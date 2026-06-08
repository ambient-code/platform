import { describe, it, expect } from 'vitest'
import {
  getRegisteredAnnotation,
  isRegisteredAnnotation,
  getAnnotationsByCategory,
  getRegisteredAnnotations,
  getPreviewAnnotations,
} from '../annotations'
import type { AnnotationCategory } from '../annotations'

describe('annotation registry', () => {
  describe('getRegisteredAnnotation', () => {
    it('returns the registration for a known key', () => {
      const result = getRegisteredAnnotation('ambient-code.io/ui/pinned')
      expect(result).not.toBeNull()
      expect(result!.key).toBe('ambient-code.io/ui/pinned')
      expect(result!.category).toBe('ui')
      expect(result!.label).toBeDefined()
    })

    it('returns null for an unknown key', () => {
      expect(getRegisteredAnnotation('some.random/key')).toBeNull()
    })

    it('returns correct category for integration keys', () => {
      const result = getRegisteredAnnotation('ambient-code.io/jira/issue')
      expect(result).not.toBeNull()
      expect(result!.category).toBe('integration')
    })

    it('returns correct category for review keys', () => {
      const result = getRegisteredAnnotation('ambient-code.io/review/status')
      expect(result).not.toBeNull()
      expect(result!.category).toBe('review')
    })

    it('returns correct category for provenance keys', () => {
      const result = getRegisteredAnnotation('ambient-code.io/triggered-by')
      expect(result).not.toBeNull()
      expect(result!.category).toBe('provenance')
    })

    it('returns correct category for cost keys', () => {
      const result = getRegisteredAnnotation('ambient-code.io/cost/estimate')
      expect(result).not.toBeNull()
      expect(result!.category).toBe('cost')
    })

    it('returns correct category for oncall keys', () => {
      const result = getRegisteredAnnotation('ambient-code.io/oncall/incident')
      expect(result).not.toBeNull()
      expect(result!.category).toBe('oncall')
    })

    it('returns correct category for agent keys', () => {
      const result = getRegisteredAnnotation('ambient-code.io/parent-agent')
      expect(result).not.toBeNull()
      expect(result!.category).toBe('agent')
    })

    it('includes a human-readable label', () => {
      const result = getRegisteredAnnotation('ambient-code.io/github/pr')
      expect(result).not.toBeNull()
      expect(result!.label).toBeTruthy()
      expect(typeof result!.label).toBe('string')
    })

    it('returns all registered keys from the spec', () => {
      const expectedKeys = [
        'ambient-code.io/ui/path',
        'ambient-code.io/ui/pinned',
        'ambient-code.io/ui/priority',
        'ambient-code.io/ui/tag',
        'ambient-code.io/ui/preview-url',
        'ambient-code.io/ui/preview-title',
        'ambient-code.io/jira/issue',
        'ambient-code.io/jira/epic',
        'ambient-code.io/github/pr',
        'ambient-code.io/github/repo',
        'ambient-code.io/github/branch',
        'ambient-code.io/gitlab/mr',
        'ambient-code.io/gerrit/change',
        'ambient-code.io/review/status',
        'ambient-code.io/review/reviewer',
        'ambient-code.io/triggered-by',
        'ambient-code.io/cost/estimate',
        'ambient-code.io/oncall/incident',
        'ambient-code.io/parent-agent',
      ]

      for (const key of expectedKeys) {
        expect(getRegisteredAnnotation(key)).not.toBeNull()
      }
    })
  })

  describe('isRegisteredAnnotation', () => {
    it('returns true for a registered key', () => {
      expect(isRegisteredAnnotation('ambient-code.io/ui/pinned')).toBe(true)
    })

    it('returns false for an unregistered key', () => {
      expect(isRegisteredAnnotation('custom.io/something')).toBe(false)
    })

    it('returns false for empty string', () => {
      expect(isRegisteredAnnotation('')).toBe(false)
    })
  })

  describe('getAnnotationsByCategory', () => {
    it('returns all ui annotations', () => {
      const results = getAnnotationsByCategory('ui')
      expect(results.length).toBe(6)
      for (const r of results) {
        expect(r.category).toBe('ui')
      }
    })

    it('returns all integration annotations', () => {
      const results = getAnnotationsByCategory('integration')
      expect(results.length).toBe(7)
      for (const r of results) {
        expect(r.category).toBe('integration')
      }
    })

    it('returns all review annotations', () => {
      const results = getAnnotationsByCategory('review')
      expect(results.length).toBe(2)
    })

    it('returns single provenance annotation', () => {
      const results = getAnnotationsByCategory('provenance')
      expect(results.length).toBe(1)
    })

    it('returns single cost annotation', () => {
      const results = getAnnotationsByCategory('cost')
      expect(results.length).toBe(1)
    })

    it('returns single oncall annotation', () => {
      const results = getAnnotationsByCategory('oncall')
      expect(results.length).toBe(1)
    })

    it('returns single agent annotation', () => {
      const results = getAnnotationsByCategory('agent')
      expect(results.length).toBe(1)
    })

    it('returns empty array for unknown category', () => {
      // Cast to bypass type check for robustness test
      const results = getAnnotationsByCategory('nonexistent' as AnnotationCategory)
      expect(results).toEqual([])
    })
  })

  describe('getRegisteredAnnotations', () => {
    it('returns registered annotations paired with their values', () => {
      const annotations: Record<string, string> = {
        'ambient-code.io/ui/pinned': 'true',
        'ambient-code.io/github/pr': 'org/repo#42',
        'custom.io/unregistered': 'ignored',
      }

      const results = getRegisteredAnnotations(annotations)
      expect(results.length).toBe(2)

      const pinned = results.find((r) => r.annotation.key === 'ambient-code.io/ui/pinned')
      expect(pinned).toBeDefined()
      expect(pinned!.value).toBe('true')

      const pr = results.find((r) => r.annotation.key === 'ambient-code.io/github/pr')
      expect(pr).toBeDefined()
      expect(pr!.value).toBe('org/repo#42')
    })

    it('returns empty array when no annotations match', () => {
      const annotations: Record<string, string> = {
        'custom.io/a': '1',
        'custom.io/b': '2',
      }

      expect(getRegisteredAnnotations(annotations)).toEqual([])
    })

    it('returns empty array for empty input', () => {
      expect(getRegisteredAnnotations({})).toEqual([])
    })
  })

  describe('getPreviewAnnotations', () => {
    it('returns url and title when both are present', () => {
      const annotations: Record<string, string> = {
        'ambient-code.io/ui/preview-url': 'https://app.example.com',
        'ambient-code.io/ui/preview-title': 'SSO Login v2',
      }

      const result = getPreviewAnnotations(annotations)
      expect(result).not.toBeNull()
      expect(result!.url).toBe('https://app.example.com')
      expect(result!.title).toBe('SSO Login v2')
    })

    it('returns url without title when only url is present', () => {
      const annotations: Record<string, string> = {
        'ambient-code.io/ui/preview-url': 'https://app.example.com',
      }

      const result = getPreviewAnnotations(annotations)
      expect(result).not.toBeNull()
      expect(result!.url).toBe('https://app.example.com')
      expect(result!.title).toBeUndefined()
    })

    it('returns null when preview-url is missing', () => {
      const annotations: Record<string, string> = {
        'ambient-code.io/ui/preview-title': 'Some Title',
      }

      expect(getPreviewAnnotations(annotations)).toBeNull()
    })

    it('returns null for empty annotations', () => {
      expect(getPreviewAnnotations({})).toBeNull()
    })
  })
})
