import { describe, it, expect, vi, afterEach } from 'vitest'
import { domainProbe } from '../observability'

describe('domainProbe', () => {
  const infoSpy = vi.spyOn(console, 'info').mockImplementation(() => {})

  afterEach(() => {
    infoSpy.mockClear()
  })

  describe('sessionPhaseChanged', () => {
    it('logs domain event name and payload', () => {
      const event = {
        sessionId: 'sess-001',
        from: 'Running',
        to: 'Failed',
        projectId: 'proj-123',
      }

      domainProbe.sessionPhaseChanged(event)

      expect(infoSpy).toHaveBeenCalledOnce()
      expect(infoSpy).toHaveBeenCalledWith(
        '[domain-probe] session.phaseChanged',
        event,
      )
    })
  })

  describe('projectSelected', () => {
    it('logs domain event name and payload', () => {
      const event = { projectId: 'proj-456' }

      domainProbe.projectSelected(event)

      expect(infoSpy).toHaveBeenCalledOnce()
      expect(infoSpy).toHaveBeenCalledWith(
        '[domain-probe] project.selected',
        event,
      )
    })
  })

  describe('apiError', () => {
    it('logs domain event name and payload', () => {
      const event = {
        operation: 'fetchSessions',
        statusCode: 503,
        message: 'Service unavailable',
      }

      domainProbe.apiError(event)

      expect(infoSpy).toHaveBeenCalledOnce()
      expect(infoSpy).toHaveBeenCalledWith(
        '[domain-probe] api.error',
        event,
      )
    })
  })

  describe('credentialRotated', () => {
    it('logs domain event name and payload', () => {
      const event = {
        credentialId: 'cred-789',
        provider: 'github',
      }

      domainProbe.credentialRotated(event)

      expect(infoSpy).toHaveBeenCalledOnce()
      expect(infoSpy).toHaveBeenCalledWith(
        '[domain-probe] credential.rotated',
        event,
      )
    })
  })

  describe('feedbackSent', () => {
    it('logs domain event name and payload', () => {
      const event = {
        sessionId: 'sess-001',
        itemCount: 3,
        previewUrl: 'https://app.example.com',
      }

      domainProbe.feedbackSent(event)

      expect(infoSpy).toHaveBeenCalledOnce()
      expect(infoSpy).toHaveBeenCalledWith(
        '[domain-probe] feedback.sent',
        event,
      )
    })
  })

  describe('feedbackDeliveryFailed', () => {
    it('logs domain event name and payload to console.error', () => {
      const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
      const event = {
        sessionId: 'sess-001',
        error: 'Network timeout',
      }

      domainProbe.feedbackDeliveryFailed(event)

      expect(errorSpy).toHaveBeenCalledOnce()
      expect(errorSpy).toHaveBeenCalledWith(
        '[domain-probe] feedback.deliveryFailed',
        event,
      )

      errorSpy.mockRestore()
    })
  })
})
