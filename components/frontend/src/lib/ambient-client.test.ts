import { describe, it, expect, vi, beforeEach } from 'vitest';
import { createAmbientClient } from './ambient-client';

describe('createAmbientClient', () => {
  beforeEach(() => {
    vi.unstubAllEnvs();
  });

  it('creates client with default no-auth token when no token sources available', () => {
    const client = createAmbientClient('test-project');
    expect(client).toBeDefined();
    expect(client.projects).toBeDefined();
    expect(client.sessions).toBeDefined();
  });

  it('creates client with empty project using default fallback', () => {
    const client = createAmbientClient('');
    expect(client).toBeDefined();
  });

  it('uses provided token over defaults', () => {
    const client = createAmbientClient('proj', 'my-token');
    expect(client).toBeDefined();
  });

  it('uses OC_TOKEN env var when available', () => {
    vi.stubEnv('OC_TOKEN', 'oc-token-value');
    const client = createAmbientClient('proj');
    expect(client).toBeDefined();
  });

  it('exposes all 13 resource APIs', () => {
    const client = createAmbientClient('proj');
    expect(client.agents).toBeDefined();
    expect(client.permissions).toBeDefined();
    expect(client.projects).toBeDefined();
    expect(client.projectKeys).toBeDefined();
    expect(client.projectSettings).toBeDefined();
    expect(client.repositoryRefs).toBeDefined();
    expect(client.sessions).toBeDefined();
    expect(client.skills).toBeDefined();
    expect(client.tasks).toBeDefined();
    expect(client.users).toBeDefined();
    expect(client.workflows).toBeDefined();
    expect(client.workflowSkills).toBeDefined();
    expect(client.workflowTasks).toBeDefined();
  });
});
