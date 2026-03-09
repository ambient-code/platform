import { describe, it, expect } from 'vitest';
import {
  STATUS_COLORS,
  SESSION_PHASE_TO_STATUS,
  getSessionPhaseColor,
  getK8sResourceStatusColor,
} from '../status-colors';

describe('getSessionPhaseColor', () => {
  it.each([
    ['pending', 'warning'],
    ['creating', 'info'],
    ['running', 'running'],
    ['stopping', 'stopping'],
    ['completed', 'success'],
    ['failed', 'error'],
    ['error', 'error'],
    ['stopped', 'stopped'],
  ] as const)('maps phase "%s" to status key "%s"', (phase, expectedKey) => {
    expect(SESSION_PHASE_TO_STATUS[phase]).toBe(expectedKey);
    expect(getSessionPhaseColor(phase)).toBe(STATUS_COLORS[expectedKey]);
  });

  it('is case-insensitive', () => {
    expect(getSessionPhaseColor('Running')).toBe(STATUS_COLORS.running);
    expect(getSessionPhaseColor('COMPLETED')).toBe(STATUS_COLORS.success);
    expect(getSessionPhaseColor('Failed')).toBe(STATUS_COLORS.error);
  });

  it('returns default color for unknown phases', () => {
    expect(getSessionPhaseColor('unknown')).toBe(STATUS_COLORS.default);
    expect(getSessionPhaseColor('')).toBe(STATUS_COLORS.default);
  });
});

describe('getK8sResourceStatusColor', () => {
  it('maps running/active states', () => {
    expect(getK8sResourceStatusColor('Running')).toBe(STATUS_COLORS.running);
    expect(getK8sResourceStatusColor('Active')).toBe(STATUS_COLORS.running);
  });

  it('maps success states', () => {
    expect(getK8sResourceStatusColor('Succeeded')).toBe(STATUS_COLORS.success);
    expect(getK8sResourceStatusColor('Completed')).toBe(STATUS_COLORS.success);
  });

  it('maps error states', () => {
    expect(getK8sResourceStatusColor('Failed')).toBe(STATUS_COLORS.error);
    expect(getK8sResourceStatusColor('Error')).toBe(STATUS_COLORS.error);
  });

  it('maps waiting/pending states', () => {
    expect(getK8sResourceStatusColor('Waiting')).toBe(STATUS_COLORS.warning);
    expect(getK8sResourceStatusColor('Pending')).toBe(STATUS_COLORS.warning);
  });

  it('maps terminating states', () => {
    expect(getK8sResourceStatusColor('Terminating')).toBe(STATUS_COLORS.stopped);
    expect(getK8sResourceStatusColor('Terminated')).toBe(STATUS_COLORS.stopped);
  });

  it('maps not found states', () => {
    expect(getK8sResourceStatusColor('NotFound')).toBe(STATUS_COLORS.warning);
    expect(getK8sResourceStatusColor('Not Found')).toBe(STATUS_COLORS.warning);
  });

  it('returns default for unrecognized status', () => {
    expect(getK8sResourceStatusColor('SomethingElse')).toBe(STATUS_COLORS.default);
  });
});

describe('STATUS_COLORS', () => {
  it('has all expected keys', () => {
    const expectedKeys = ['success', 'error', 'warning', 'info', 'pending', 'running', 'stopping', 'stopped', 'default'];
    for (const key of expectedKeys) {
      expect(STATUS_COLORS).toHaveProperty(key);
      expect(typeof STATUS_COLORS[key as keyof typeof STATUS_COLORS]).toBe('string');
    }
  });
});
