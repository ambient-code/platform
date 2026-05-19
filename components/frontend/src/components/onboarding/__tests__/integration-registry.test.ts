import { describe, it, expect } from 'vitest';
import { INTEGRATION_REGISTRY } from '../integration-registry';
import type { IntegrationsStatus } from '@/services/api/integrations';

function makeStatus(overrides: Partial<IntegrationsStatus> = {}): IntegrationsStatus {
  return {
    github: {
      installed: false,
      pat: { configured: false },
    },
    gitlab: { connected: false },
    google: { connected: false },
    jira: { connected: false },
    coderabbit: { connected: false },
    gerrit: { connected: false },
    ...overrides,
  };
}

describe('INTEGRATION_REGISTRY', () => {
  it('has an entry for every non-mcpServers key of IntegrationsStatus', () => {
    const registryIds = INTEGRATION_REGISTRY.map((e) => e.id);
    const expectedKeys: string[] = ['github', 'gitlab', 'google', 'jira', 'coderabbit', 'gerrit'];
    expect(registryIds.sort()).toEqual(expectedKeys.sort());
  });

  it('github: detects App install', () => {
    const entry = INTEGRATION_REGISTRY.find((e) => e.id === 'github')!;
    expect(entry.isConnected(makeStatus({ github: { installed: true, pat: { configured: false } } }))).toBe(true);
    expect(entry.isConnected(makeStatus())).toBe(false);
  });

  it('github: detects PAT', () => {
    const entry = INTEGRATION_REGISTRY.find((e) => e.id === 'github')!;
    expect(entry.isConnected(makeStatus({ github: { installed: false, pat: { configured: true } } }))).toBe(true);
  });

  it('gitlab: detects connected', () => {
    const entry = INTEGRATION_REGISTRY.find((e) => e.id === 'gitlab')!;
    expect(entry.isConnected(makeStatus({ gitlab: { connected: true } }))).toBe(true);
    expect(entry.isConnected(makeStatus())).toBe(false);
  });

  it('google: detects connected', () => {
    const entry = INTEGRATION_REGISTRY.find((e) => e.id === 'google')!;
    expect(entry.isConnected(makeStatus({ google: { connected: true } }))).toBe(true);
    expect(entry.isConnected(makeStatus())).toBe(false);
  });

  it('jira: detects connected', () => {
    const entry = INTEGRATION_REGISTRY.find((e) => e.id === 'jira')!;
    expect(entry.isConnected(makeStatus({ jira: { connected: true } }))).toBe(true);
    expect(entry.isConnected(makeStatus())).toBe(false);
  });

  it('coderabbit: detects connected', () => {
    const entry = INTEGRATION_REGISTRY.find((e) => e.id === 'coderabbit')!;
    expect(entry.isConnected(makeStatus({ coderabbit: { connected: true } }))).toBe(true);
    expect(entry.isConnected(makeStatus())).toBe(false);
  });

  it('gerrit: detects connected instance', () => {
    const entry = INTEGRATION_REGISTRY.find((e) => e.id === 'gerrit')!;
    expect(
      entry.isConnected(
        makeStatus({
          gerrit: {
            connected: false,
            instances: [
              { instanceName: 'g1', url: 'https://g.example.com', authMethod: 'http_basic', connected: true },
            ],
          },
        })
      )
    ).toBe(true);
    expect(entry.isConnected(makeStatus())).toBe(false);
  });

  it('gerrit: false when no connected instances', () => {
    const entry = INTEGRATION_REGISTRY.find((e) => e.id === 'gerrit')!;
    expect(
      entry.isConnected(
        makeStatus({
          gerrit: {
            connected: false,
            instances: [
              { instanceName: 'g1', url: 'https://g.example.com', authMethod: 'http_basic', connected: false },
            ],
          },
        })
      )
    ).toBe(false);
  });
});
