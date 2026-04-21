/**
 * Integration tests for the Ambient SDK.
 *
 * Requires environment variables:
 *   AMBIENT_API_URL   - e.g. http://localhost:8080
 *   AMBIENT_TOKEN     - Bearer token (≥20 chars)
 *   AMBIENT_PROJECT   - Kubernetes namespace / project name
 *
 * Skipped automatically when AMBIENT_TOKEN is absent (unit CI).
 * Run against a live cluster: AMBIENT_TOKEN=sha256~... AMBIENT_API_URL=... AMBIENT_PROJECT=... npm test
 */

import {
  AmbientClient,
  AgentBuilder,
  AgentPatchBuilder,
  ProjectBuilder,
  ProjectPatchBuilder,
} from '../src';

const SKIP = !process.env.AMBIENT_TOKEN;
const describeIntegration = SKIP ? describe.skip : describe;

const uid = () => Math.random().toString(36).slice(2, 8);

let client: AmbientClient;

beforeAll(() => {
  if (SKIP) return;
  client = AmbientClient.fromEnv();
});

// ── agents ──────────────────────────────────────────────────────────────────
describeIntegration('agents', () => {
  let agentId: string;
  const name = `sdk-test-agent-${uid()}`;

  it('create — agents.create()', async () => {
    const req = new AgentBuilder()
      .name(name)
      .ownerUserId('dev/user')
      .prompt('You are a test agent created by the SDK integration tests.')
      .build();
    const agent = await client.agents.create(req);
    expect(agent.id).toBeTruthy();
    expect(agent.name).toBe(name);
    agentId = agent.id;
  });

  it('get — agents.get()', async () => {
    const agent = await client.agents.get(agentId);
    expect(agent.id).toBe(agentId);
    expect(agent.name).toBe(name);
  });

  it('list — agents.list()', async () => {
    const result = await client.agents.list({ page: 1, size: 100 });
    expect(result.items).toBeInstanceOf(Array);
    expect(result.total).toBeGreaterThanOrEqual(1);
    const found = result.items.find(a => a.id === agentId);
    expect(found).toBeDefined();
  });

  it('listAll — agents.listAll() AsyncGenerator', async () => {
    const agents: typeof client.agents extends { listAll(): AsyncGenerator<infer T> } ? T[] : never[] = [];
    for await (const a of client.agents.listAll()) {
      (agents as any[]).push(a);
    }
    expect((agents as any[]).some((a: any) => a.id === agentId)).toBe(true);
  });

  it('update — agents.update()', async () => {
    const patch = new AgentPatchBuilder()
      .prompt('Updated prompt from SDK integration test.')
      .build();
    const updated = await client.agents.update(agentId, patch);
    expect(updated.id).toBe(agentId);
  });
});

// ── projects (workspaces) ───────────────────────────────────────────────────
describeIntegration('projects', () => {
  let projectId: string;
  const name = `sdk-test-ws-${uid()}`;

  it('create — projects.create()', async () => {
    const req = new ProjectBuilder()
      .name(name)
      .displayName('SDK Test Workspace')
      .description('Created by integration tests')
      .prompt('This workspace is used for automated SDK integration testing.')
      .build();
    const project = await client.projects.create(req);
    expect(project.id).toBeTruthy();
    expect(project.name).toBe(name);
    projectId = project.id;
  });

  it('get — projects.get()', async () => {
    const project = await client.projects.get(projectId);
    expect(project.id).toBe(projectId);
    expect(project.name).toBe(name);
  });

  it('list — projects.list()', async () => {
    const result = await client.projects.list({ page: 1, size: 50 });
    expect(result.items).toBeInstanceOf(Array);
    const found = result.items.find(p => p.id === projectId);
    expect(found).toBeDefined();
  });

  it('listAll — projects.listAll() AsyncGenerator', async () => {
    const projects: any[] = [];
    for await (const p of client.projects.listAll()) {
      projects.push(p);
    }
    expect(projects.some(p => p.id === projectId)).toBe(true);
  });

  it('update — projects.update()', async () => {
    const patch = new ProjectPatchBuilder()
      .displayName('SDK Test Workspace (updated)')
      .prompt('Updated workspace prompt.')
      .build();
    const updated = await client.projects.update(projectId, patch);
    expect(updated.id).toBe(projectId);
  });

  // agents within project sub-tests nested to reuse projectId
  describeIntegration('agents (within project)', () => {
    let paId: string;
    const agentName = `sdk-test-proj-agent-${uid()}`;

    it('create — agents.createInProject()', async () => {
      const pa = await client.agents.createInProject(projectId, new AgentBuilder()
        .name(agentName)
        .projectId(projectId)
        .build()
      );
      expect(pa.id).toBeTruthy();
      expect(pa.project_id).toBe(projectId);
      paId = pa.id;
    });

    it('get — agents.getByProject()', async () => {
      const pa = await client.agents.getByProject(projectId, paId);
      expect(pa.id).toBe(paId);
    });

    it('list — agents.listByProject()', async () => {
      const result = await client.agents.listByProject(projectId, { page: 1, size: 50 });
      expect(result.items).toBeInstanceOf(Array);
      expect(result.items.find(pa => pa.id === paId)).toBeDefined();
    });

    it('sessions — agents.sessions()', async () => {
      const result = await client.agents.sessions(projectId, paId, { page: 1, size: 10 });
      expect(result.items).toBeInstanceOf(Array);
    });

    it('inboxMessages.send() and inboxMessages.list()', async () => {
      const msg = await client.inboxMessages.send(projectId, paId, {
        body: 'Hello from SDK integration test',
        agent_id: paId,
        from_name: 'test-runner',
      });
      expect(msg.id).toBeTruthy();
      expect(msg.body).toBe('Hello from SDK integration test');

      const list = await client.inboxMessages.list(projectId, paId, { page: 1, size: 10 });
      expect(list.items.find(m => m.id === msg.id)).toBeDefined();
    });

    it('inboxMessages.listAll() AsyncGenerator', async () => {
      const msgs: any[] = [];
      for await (const m of client.inboxMessages.listAll(projectId, paId)) {
        msgs.push(m);
      }
      expect(msgs.length).toBeGreaterThanOrEqual(1);
    });

    it('start — agents.start()', async () => {
      const resp = await client.agents.start(projectId, paId, 'SDK integration test session poke');
      expect(resp).toBeDefined();
    });

    it('getIgnition — agents.getIgnition()', async () => {
      const resp = await client.agents.getIgnition(projectId, paId);
      expect(resp).toBeDefined();
    });

    it('delete — agents.deleteInProject()', async () => {
      await expect(client.agents.deleteInProject(projectId, paId)).resolves.toBeUndefined();
    });
  });

  it('delete — projects.delete()', async () => {
    await expect(client.projects.delete(projectId)).resolves.toBeUndefined();
  });
});

// ── sessions ────────────────────────────────────────────────────────────────
describeIntegration('sessions', () => {
  it('list — sessions.list()', async () => {
    const result = await client.sessions.list({ page: 1, size: 10 });
    expect(result.items).toBeInstanceOf(Array);
    expect(typeof result.total).toBe('number');
  });

  it('listAll — sessions.listAll() AsyncGenerator', async () => {
    const sessions: any[] = [];
    for await (const s of client.sessions.listAll()) {
      sessions.push(s);
      if (sessions.length >= 5) break;
    }
    expect(sessions.length).toBeGreaterThanOrEqual(0);
  });

  it('sessionMessages.list() on first available session', async () => {
    const result = await client.sessions.list({ page: 1, size: 5 });
    if (result.items.length === 0) return;
    const sessionId = result.items[0].id;
    const msgs = await client.sessionMessages.list(sessionId, { page: 1, size: 50 });
    expect(msgs.items).toBeInstanceOf(Array);
  });

  it('sessionMessages.listAll() AsyncGenerator', async () => {
    const result = await client.sessions.list({ page: 1, size: 5 });
    if (result.items.length === 0) return;
    const sessionId = result.items[0].id;
    const msgs: any[] = [];
    for await (const m of client.sessionMessages.listAll(sessionId)) {
      msgs.push(m);
      if (msgs.length >= 20) break;
    }
    expect(msgs.length).toBeGreaterThanOrEqual(0);
  });
});

// ── users ───────────────────────────────────────────────────────────────────
describeIntegration('users', () => {
  it('list — users.list()', async () => {
    const result = await client.users.list({ page: 1, size: 10 });
    expect(result.items).toBeInstanceOf(Array);
  });

  it('listAll — users.listAll() AsyncGenerator', async () => {
    const users: any[] = [];
    for await (const u of client.users.listAll()) {
      users.push(u);
    }
    expect(users.length).toBeGreaterThanOrEqual(0);
  });
});

// ── roles ───────────────────────────────────────────────────────────────────
describeIntegration('roles', () => {
  it('list — roles.list()', async () => {
    const result = await client.roles.list({ page: 1, size: 10 });
    expect(result.items).toBeInstanceOf(Array);
  });

  it('listAll — roles.listAll() AsyncGenerator', async () => {
    const roles: any[] = [];
    for await (const r of client.roles.listAll()) {
      roles.push(r);
    }
    expect(roles.length).toBeGreaterThanOrEqual(0);
  });
});

// ── roleBindings ─────────────────────────────────────────────────────────────
describeIntegration('roleBindings', () => {
  it('list — roleBindings.list()', async () => {
    const result = await client.roleBindings.list({ page: 1, size: 10 });
    expect(result.items).toBeInstanceOf(Array);
  });

  it('listAll — roleBindings.listAll() AsyncGenerator', async () => {
    const rbs: any[] = [];
    for await (const rb of client.roleBindings.listAll()) {
      rbs.push(rb);
    }
    expect(rbs.length).toBeGreaterThanOrEqual(0);
  });
});

// ── projectSettings ─────────────────────────────────────────────────────────
describeIntegration('projectSettings', () => {
  it('list — projectSettings.list()', async () => {
    const result = await client.projectSettings.list({ page: 1, size: 10 });
    expect(result.items).toBeInstanceOf(Array);
  });

  it('listAll — projectSettings.listAll() AsyncGenerator', async () => {
    const pss: any[] = [];
    for await (const ps of client.projectSettings.listAll()) {
      pss.push(ps);
    }
    expect(pss.length).toBeGreaterThanOrEqual(0);
  });
});
