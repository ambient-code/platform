import type { SessionPhase } from '@/domain/types'

export type PhaseStyle = {
  variant: 'success' | 'error' | 'warning' | 'info' | 'default'
  label: string
  pulse: boolean
}

const PHASE_STYLES: Record<SessionPhase, PhaseStyle> = {
  Running: { variant: 'success', label: 'Running', pulse: true },
  Completed: { variant: 'info', label: 'Completed', pulse: false },
  Failed: { variant: 'error', label: 'Failed', pulse: false },
  Stopped: { variant: 'default', label: 'Stopped', pulse: false },
  Pending: { variant: 'warning', label: 'Pending', pulse: false },
  Creating: { variant: 'warning', label: 'Creating', pulse: false },
  Stopping: { variant: 'warning', label: 'Stopping', pulse: false },
}

export function getPhaseStyle(phase: SessionPhase): PhaseStyle {
  return PHASE_STYLES[phase] ?? PHASE_STYLES.Pending
}
