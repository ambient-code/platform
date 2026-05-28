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
}
