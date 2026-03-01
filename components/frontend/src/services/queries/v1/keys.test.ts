import { describe, it, expect } from 'vitest';
import {
  agentKeys,
  sessionKeys,
  projectKeys,
  projectKeyKeys,
  permissionKeys,
  repositoryRefKeys,
  skillKeys,
  taskKeys,
  userKeys,
  workflowKeys,
  workflowSkillKeys,
  workflowTaskKeys,
  projectSettingsKeys,
} from './keys';

describe('v1 query key factories', () => {
  it('generates unique base keys for all 13 resources', () => {
    const allKeys = [
      agentKeys.all,
      sessionKeys.all,
      projectKeys.all,
      projectKeyKeys.all,
      permissionKeys.all,
      repositoryRefKeys.all,
      skillKeys.all,
      taskKeys.all,
      userKeys.all,
      workflowKeys.all,
      workflowSkillKeys.all,
      workflowTaskKeys.all,
      projectSettingsKeys.all,
    ];

    const serialized = allKeys.map((k) => JSON.stringify(k));
    const unique = new Set(serialized);
    expect(unique.size).toBe(13);
  });

  it('all keys start with v1 prefix', () => {
    expect(agentKeys.all[0]).toBe('v1');
    expect(sessionKeys.all[0]).toBe('v1');
    expect(projectKeys.all[0]).toBe('v1');
    expect(projectKeyKeys.all[0]).toBe('v1');
  });

  it('list keys include options', () => {
    const opts = { page: 2, size: 10 };
    const listKey = sessionKeys.list(opts);
    expect(listKey).toEqual(['v1', 'sessions', 'list', opts]);
  });

  it('list keys default to empty object when no options', () => {
    const listKey = sessionKeys.list();
    expect(listKey).toEqual(['v1', 'sessions', 'list', {}]);
  });

  it('detail keys include id', () => {
    const detailKey = projectKeys.detail('abc-123');
    expect(detailKey).toEqual(['v1', 'projects', 'detail', 'abc-123']);
  });

  it('projectKeyKeys does not collide with projectKeys', () => {
    expect(projectKeys.all[1]).toBe('projects');
    expect(projectKeyKeys.all[1]).toBe('projectKeys');
    expect(JSON.stringify(projectKeys.all)).not.toBe(JSON.stringify(projectKeyKeys.all));
  });
});
