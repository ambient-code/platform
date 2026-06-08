type SessionPhaseChangedEvent = {
  sessionId: string
  from: string
  to: string
  projectId: string
}

type ProjectSelectedEvent = {
  projectId: string
}

type ApiErrorEvent = {
  operation: string
  statusCode: number
  message: string
}

type CredentialRotatedEvent = {
  credentialId: string
  provider: string
}

type FeedbackSentEvent = {
  sessionId: string
  itemCount: number
  previewUrl: string
}

type FeedbackDeliveryFailedEvent = {
  sessionId: string
  error: string
}

export const domainProbe = {
  sessionPhaseChanged(event: SessionPhaseChangedEvent) {
    console.info('[domain-probe] session.phaseChanged', event)
  },

  projectSelected(event: ProjectSelectedEvent) {
    console.info('[domain-probe] project.selected', event)
  },

  apiError(event: ApiErrorEvent) {
    console.info('[domain-probe] api.error', event)
  },

  credentialRotated(event: CredentialRotatedEvent) {
    console.info('[domain-probe] credential.rotated', event)
  },

  feedbackSent(event: FeedbackSentEvent) {
    console.info('[domain-probe] feedback.sent', event)
  },

  feedbackDeliveryFailed(event: FeedbackDeliveryFailedEvent) {
    console.error('[domain-probe] feedback.deliveryFailed', event)
  },
}
