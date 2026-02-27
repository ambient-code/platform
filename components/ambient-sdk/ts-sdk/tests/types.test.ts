import {
  SessionBuilder,
  SessionPatchBuilder,
  SessionStatusPatchBuilder,
  ProjectBuilder,
  ProjectPatchBuilder,
  ProjectSettingsBuilder,
  ProjectSettingsPatchBuilder,
  UserBuilder,
} from '../src';
import type {
  Session,
  SessionList,
  SessionCreateRequest,
  SessionPatchRequest,
  SessionStatusPatchRequest,
  Project,
  ProjectList,
  ProjectSettings,
  ProjectSettingsList,
  User,
  UserList,
  ObjectReference,
  ListMeta,
  ListOptions,
} from '../src';

describe('ObjectReference base type', () => {
  it('should have correct shape', () => {
    const ref: ObjectReference = {
      id: 'abc123',
      kind: 'Session',
      href: '/sessions/abc123',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: null,
    };
    expect(ref.id).toBe('abc123');
    expect(ref.kind).toBe('Session');
    expect(ref.updated_at).toBeNull();
  });
});

describe('ListMeta base type', () => {
  it('should have correct shape', () => {
    const meta: ListMeta = { kind: 'SessionList', page: 1, size: 100, total: 250 };
    expect(meta.page).toBe(1);
    expect(meta.total).toBe(250);
  });
});

describe('ListOptions type', () => {
  it('should accept partial options', () => {
    const opts: ListOptions = { page: 2, size: 50 };
    expect(opts.page).toBe(2);
    expect(opts.search).toBeUndefined();
  });
});

describe('Session types', () => {
  it('Session type extends ObjectReference', () => {
    const session: Session = {
      id: 'sess-1',
      kind: 'Session',
      href: '/sessions/sess-1',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
      name: 'test-session',
      prompt: 'do something',
      phase: 'pending',
      timeout: 3600,
      llm_model: 'claude-sonnet-4-20250514',
      llm_temperature: 0.7,
      llm_max_tokens: 4096,
      annotations: '',
      assigned_user_id: '',
      bot_account_name: '',
      completion_time: '',
      conditions: '',
      created_by_user_id: '',
      environment_variables: '',
      kube_cr_name: '',
      kube_cr_uid: '',
      kube_namespace: '',
      labels: '',
      parent_session_id: '',
      project_id: '',
      reconciled_repos: '',
      reconciled_workflow: '',
      repo_url: '',
      repos: '',
      resource_overrides: '',
      sdk_restart_count: 0,
      sdk_session_id: '',
      start_time: '',
      workflow_id: '',
    };
    expect(session.id).toBe('sess-1');
    expect(session.name).toBe('test-session');
    expect(session.phase).toBe('pending');
  });

  it('SessionList has items array', () => {
    const list: SessionList = {
      kind: 'SessionList',
      page: 1,
      size: 10,
      total: 1,
      items: [],
    };
    expect(list.items).toHaveLength(0);
    expect(list.total).toBe(1);
  });
});

describe('SessionBuilder', () => {
  it('builds a valid create request with fluent API', () => {
    const req = new SessionBuilder()
      .name('my-session')
      .prompt('analyze code')
      .llmModel('claude-sonnet-4-20250514')
      .timeout(3600)
      .build();
    expect(req.name).toBe('my-session');
    expect(req.prompt).toBe('analyze code');
    expect(req.llm_model).toBe('claude-sonnet-4-20250514');
    expect(req.timeout).toBe(3600);
  });

  it('throws when name is missing', () => {
    expect(() => new SessionBuilder().prompt('test').build()).toThrow('name is required');
  });
});

describe('SessionPatchBuilder', () => {
  it('builds a partial patch', () => {
    const patch = new SessionPatchBuilder()
      .prompt('updated prompt')
      .timeout(7200)
      .build();
    expect(patch.prompt).toBe('updated prompt');
    expect(patch.timeout).toBe(7200);
    expect(patch.name).toBeUndefined();
  });
});

describe('SessionStatusPatchBuilder', () => {
  it('builds a status patch', () => {
    const patch = new SessionStatusPatchBuilder()
      .phase('running')
      .startTime('2026-01-01T00:00:00Z')
      .build();
    expect(patch.phase).toBe('running');
    expect(patch.start_time).toBe('2026-01-01T00:00:00Z');
    expect(patch.completion_time).toBeUndefined();
  });
});

describe('All 4 resource builders exist and build', () => {
  it('SessionBuilder', () => {
    const req = new SessionBuilder().name('session-1').build();
    expect(req.name).toBe('session-1');
  });

  it('ProjectBuilder', () => {
    const req = new ProjectBuilder().name('project-1').build();
    expect(req.name).toBe('project-1');
  });

  it('ProjectSettingsBuilder', () => {
    const req = new ProjectSettingsBuilder().projectId('proj-1').build();
    expect(req.project_id).toBe('proj-1');
  });

  it('UserBuilder', () => {
    const req = new UserBuilder().name('user-1').username('user1').build();
    expect(req.name).toBe('user-1');
    expect(req.username).toBe('user1');
  });

  it('UserBuilder throws when name is missing', () => {
    expect(() => new UserBuilder().username('user1').build()).toThrow('name is required');
  });

  it('UserBuilder throws when username is missing', () => {
    expect(() => new UserBuilder().name('user-1').build()).toThrow('username is required');
  });
});

describe('PatchBuilder for each resource', () => {
  it('SessionPatchBuilder exists', () => {
    const patch = new SessionPatchBuilder().name('updated').build();
    expect(patch.name).toBe('updated');
  });

  it('ProjectPatchBuilder exists', () => {
    const patch = new ProjectPatchBuilder().name('updated').build();
    expect(patch.name).toBe('updated');
  });

  it('ProjectSettingsPatchBuilder exists', () => {
    const patch = new ProjectSettingsPatchBuilder().projectId('proj-2').build();
    expect(patch.project_id).toBe('proj-2');
  });
});