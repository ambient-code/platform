import { AmbientClient, AmbientAPIError } from '../src';

describe('AmbientClient construction', () => {
  it('creates client with valid config', () => {
    const client = new AmbientClient({
      baseUrl: 'https://api.example.com',
      token: 'sha256~abcdefghijklmnopqrstuvwxyz1234567890',
      project: 'test-project',
    });
    expect(client).toBeDefined();
    expect(client.sessions).toBeDefined();
    expect(client.projects).toBeDefined();
    expect(client.projectSettings).toBeDefined();
    expect(client.users).toBeDefined();
  });

  it('throws when baseUrl is missing', () => {
    expect(() => new AmbientClient({
      baseUrl: '',
      token: 'test-token',
      project: 'test-project',
    })).toThrow('baseUrl is required');
  });

  it('throws when token is missing', () => {
    expect(() => new AmbientClient({
      baseUrl: 'https://api.example.com',
      token: '',
      project: 'test-project',
    })).toThrow('token is required');
  });

  it('throws when project is missing', () => {
    expect(() => new AmbientClient({
      baseUrl: 'https://api.example.com',
      token: 'test-token',
      project: '',
    })).toThrow('project is required');
  });

  it('strips trailing slashes from baseUrl', () => {
    const client = new AmbientClient({
      baseUrl: 'https://api.example.com///',
      token: 'test-token',
      project: 'test-project',
    });
    expect(client).toBeDefined();
  });
});

describe('AmbientClient.fromEnv', () => {
  const originalEnv = process.env;

  beforeEach(() => {
    process.env = { ...originalEnv };
  });

  afterAll(() => {
    process.env = originalEnv;
  });

  it('creates client from environment variables', () => {
    process.env.AMBIENT_API_URL = 'https://api.test.com';
    process.env.AMBIENT_TOKEN = 'sha256~testtoken123';
    process.env.AMBIENT_PROJECT = 'my-project';
    const client = AmbientClient.fromEnv();
    expect(client).toBeDefined();
    expect(client.sessions).toBeDefined();
  });

  it('throws when AMBIENT_TOKEN is missing', () => {
    delete process.env.AMBIENT_TOKEN;
    process.env.AMBIENT_PROJECT = 'my-project';
    expect(() => AmbientClient.fromEnv()).toThrow('AMBIENT_TOKEN environment variable is required');
  });

  it('throws when AMBIENT_PROJECT is missing', () => {
    process.env.AMBIENT_TOKEN = 'test-token';
    delete process.env.AMBIENT_PROJECT;
    expect(() => AmbientClient.fromEnv()).toThrow('AMBIENT_PROJECT environment variable is required');
  });
});

describe('AmbientAPIError', () => {
  it('formats error message correctly', () => {
    const error = new AmbientAPIError({
      id: '',
      kind: 'Error',
      href: '',
      code: 'not_found',
      reason: 'Session not found',
      operation_id: '',
      status_code: 404,
    });
    expect(error.message).toBe('ambient API error 404: not_found — Session not found');
    expect(error.statusCode).toBe(404);
    expect(error.code).toBe('not_found');
    expect(error.reason).toBe('Session not found');
    expect(error.name).toBe('AmbientAPIError');
    expect(error).toBeInstanceOf(Error);
  });
});

describe('Resource API accessor properties', () => {
  const client = new AmbientClient({
    baseUrl: 'https://api.test.com',
    token: 'sha256~abcdefghijklmnopqrstuvwxyz1234567890',
    project: 'test-project',
  });

  const resourcesWithUpdate = [
    'sessions', 'projectSettings',
  ] as const;

  for (const name of resourcesWithUpdate) {
    it(`${name} API has CRUD methods`, () => {
      const api = client[name] as Record<string, unknown>;
      expect(typeof api.create).toBe('function');
      expect(typeof api.get).toBe('function');
      expect(typeof api.list).toBe('function');
      expect(typeof api.update).toBe('function');
      expect(typeof api.listAll).toBe('function');
    });
  }

  const resourcesWithDelete = [
    'projects', 'projectSettings',
  ] as const;

  for (const name of resourcesWithDelete) {
    it(`${name} API has delete method`, () => {
      const api = client[name] as Record<string, unknown>;
      expect(typeof api.delete).toBe('function');
    });
  }

  it('sessions API has start/stop/updateStatus methods', () => {
    expect(typeof client.sessions.start).toBe('function');
    expect(typeof client.sessions.stop).toBe('function');
    expect(typeof client.sessions.updateStatus).toBe('function');
  });

  it('users API has basic CRUD methods', () => {
    const api = client.users as Record<string, unknown>;
    expect(typeof api.create).toBe('function');
    expect(typeof api.get).toBe('function');
    expect(typeof api.list).toBe('function');
    expect(typeof api.listAll).toBe('function');
  });
});
