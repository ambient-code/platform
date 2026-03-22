"use strict";
var AmbientSDK = (() => {
  var __defProp = Object.defineProperty;
  var __getOwnPropDesc = Object.getOwnPropertyDescriptor;
  var __getOwnPropNames = Object.getOwnPropertyNames;
  var __hasOwnProp = Object.prototype.hasOwnProperty;
  var __export = (target, all) => {
    for (var name in all)
      __defProp(target, name, { get: all[name], enumerable: true });
  };
  var __copyProps = (to, from, except, desc) => {
    if (from && typeof from === "object" || typeof from === "function") {
      for (let key of __getOwnPropNames(from))
        if (!__hasOwnProp.call(to, key) && key !== except)
          __defProp(to, key, { get: () => from[key], enumerable: !(desc = __getOwnPropDesc(from, key)) || desc.enumerable });
    }
    return to;
  };
  var __toCommonJS = (mod) => __copyProps(__defProp({}, "__esModule", { value: true }), mod);

  // ../../ambient/platform/platform-api-server/components/ambient-sdk/ts-sdk/src/index.ts
  var index_exports = {};
  __export(index_exports, {
    AmbientAPIError: () => AmbientAPIError,
    AmbientClient: () => AmbientClient,
    InboxMessageAPI: () => InboxMessageAPI,
    InboxMessageBuilder: () => InboxMessageBuilder,
    InboxMessagePatchBuilder: () => InboxMessagePatchBuilder,
    ProjectAPI: () => ProjectAPI,
    ProjectAgentAPI: () => ProjectAgentAPI,
    ProjectAgentBuilder: () => ProjectAgentBuilder,
    ProjectAgentPatchBuilder: () => ProjectAgentPatchBuilder,
    ProjectBuilder: () => ProjectBuilder,
    ProjectPatchBuilder: () => ProjectPatchBuilder,
    ProjectSettingsAPI: () => ProjectSettingsAPI,
    ProjectSettingsBuilder: () => ProjectSettingsBuilder,
    ProjectSettingsPatchBuilder: () => ProjectSettingsPatchBuilder,
    RoleAPI: () => RoleAPI,
    RoleBindingAPI: () => RoleBindingAPI,
    RoleBindingBuilder: () => RoleBindingBuilder,
    RoleBindingPatchBuilder: () => RoleBindingPatchBuilder,
    RoleBuilder: () => RoleBuilder,
    RolePatchBuilder: () => RolePatchBuilder,
    SessionAPI: () => SessionAPI,
    SessionBuilder: () => SessionBuilder,
    SessionMessageAPI: () => SessionMessageAPI,
    SessionMessageBuilder: () => SessionMessageBuilder,
    SessionMessagePatchBuilder: () => SessionMessagePatchBuilder,
    SessionPatchBuilder: () => SessionPatchBuilder,
    SessionStatusPatchBuilder: () => SessionStatusPatchBuilder,
    UserAPI: () => UserAPI,
    UserBuilder: () => UserBuilder,
    UserPatchBuilder: () => UserPatchBuilder,
    buildQueryString: () => buildQueryString
  });

  // ../../ambient/platform/platform-api-server/components/ambient-sdk/ts-sdk/src/base.ts
  var AmbientAPIError = class extends Error {
    constructor(error) {
      super(`ambient API error ${error.status_code}: ${error.code} \u2014 ${error.reason}`);
      this.name = "AmbientAPIError";
      this.statusCode = error.status_code;
      this.code = error.code;
      this.reason = error.reason;
      this.operationId = error.operation_id;
    }
  };
  function buildQueryString(opts) {
    if (!opts) return "";
    const params = new URLSearchParams();
    if (opts.page !== void 0) params.set("page", String(opts.page));
    if (opts.size !== void 0) params.set("size", String(Math.min(opts.size, 65500)));
    if (opts.search) params.set("search", opts.search);
    if (opts.orderBy) params.set("orderBy", opts.orderBy);
    if (opts.fields) params.set("fields", opts.fields);
    const qs = params.toString();
    return qs ? `?${qs}` : "";
  }
  async function ambientFetch(config, method, path, body, requestOpts) {
    const url = `${config.baseUrl}/api/ambient/v1${path}`;
    const headers = {
      "Authorization": `Bearer ${config.token}`,
      "X-Ambient-Project": config.project
    };
    if (body !== void 0) {
      headers["Content-Type"] = "application/json";
    }
    const resp = await fetch(url, {
      method,
      headers,
      body: body !== void 0 ? JSON.stringify(body) : void 0,
      signal: requestOpts?.signal
    });
    if (!resp.ok) {
      let errorData;
      try {
        const jsonData2 = await resp.json();
        if (typeof jsonData2 === "object" && jsonData2 !== null) {
          errorData = {
            id: typeof jsonData2.id === "string" ? jsonData2.id : "",
            kind: typeof jsonData2.kind === "string" ? jsonData2.kind : "Error",
            href: typeof jsonData2.href === "string" ? jsonData2.href : "",
            code: typeof jsonData2.code === "string" ? jsonData2.code : "unknown_error",
            reason: typeof jsonData2.reason === "string" ? jsonData2.reason : `HTTP ${resp.status}: ${resp.statusText}`,
            operation_id: typeof jsonData2.operation_id === "string" ? jsonData2.operation_id : "",
            status_code: resp.status
          };
        } else {
          throw new Error("Invalid error response format");
        }
      } catch {
        errorData = {
          id: "",
          kind: "Error",
          href: "",
          code: "unknown_error",
          reason: `HTTP ${resp.status}: ${resp.statusText}`,
          operation_id: "",
          status_code: resp.status
        };
      }
      throw new AmbientAPIError(errorData);
    }
    if (resp.status === 204) {
      return void 0;
    }
    const jsonData = await resp.json();
    return jsonData;
  }

  // ../../ambient/platform/platform-api-server/components/ambient-sdk/ts-sdk/src/inbox_message_api.ts
  var InboxMessageAPI = class {
    constructor(config) {
      this.config = config;
    }
    async send(projectId, agentId, data, opts) {
      return ambientFetch(this.config, "POST", `/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agentId)}/inbox`, data, opts);
    }
    async listByAgent(projectId, agentId, listOpts, opts) {
      const qs = buildQueryString(listOpts);
      return ambientFetch(this.config, "GET", `/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agentId)}/inbox${qs}`, void 0, opts);
    }
    async markRead(projectId, agentId, msgId, opts) {
      return ambientFetch(this.config, "PATCH", `/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agentId)}/inbox/${encodeURIComponent(msgId)}`, { read: true }, opts);
    }
    async deleteMessage(projectId, agentId, msgId, opts) {
      return ambientFetch(this.config, "DELETE", `/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agentId)}/inbox/${encodeURIComponent(msgId)}`, void 0, opts);
    }
  };

  // ../../ambient/platform/platform-api-server/components/ambient-sdk/ts-sdk/src/project_api.ts
  var ProjectAPI = class {
    constructor(config) {
      this.config = config;
    }
    async create(data, opts) {
      return ambientFetch(this.config, "POST", "/projects", data, opts);
    }
    async get(id, opts) {
      return ambientFetch(this.config, "GET", `/projects/${id}`, void 0, opts);
    }
    async list(listOpts, opts) {
      const qs = buildQueryString(listOpts);
      return ambientFetch(this.config, "GET", `/projects${qs}`, void 0, opts);
    }
    async update(id, patch, opts) {
      return ambientFetch(this.config, "PATCH", `/projects/${id}`, patch, opts);
    }
    async delete(id, opts) {
      return ambientFetch(this.config, "DELETE", `/projects/${id}`, void 0, opts);
    }
    async *listAll(size = 100, opts) {
      let page = 1;
      while (true) {
        const result = await this.list({ page, size }, opts);
        for (const item of result.items) {
          yield item;
        }
        if (page * size >= result.total) {
          break;
        }
        page++;
      }
    }
  };

  // ../../ambient/platform/platform-api-server/components/ambient-sdk/ts-sdk/src/project_agent_api.ts
  var ProjectAgentAPI = class {
    constructor(config) {
      this.config = config;
    }
    async listByProject(projectId, listOpts, opts) {
      const qs = buildQueryString(listOpts);
      return ambientFetch(this.config, "GET", `/projects/${encodeURIComponent(projectId)}/agents${qs}`, void 0, opts);
    }
    async getByProject(projectId, agentId, opts) {
      return ambientFetch(this.config, "GET", `/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agentId)}`, void 0, opts);
    }
    async createInProject(projectId, data, opts) {
      return ambientFetch(this.config, "POST", `/projects/${encodeURIComponent(projectId)}/agents`, data, opts);
    }
    async updateInProject(projectId, agentId, patch, opts) {
      return ambientFetch(this.config, "PATCH", `/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agentId)}`, patch, opts);
    }
    async deleteInProject(projectId, agentId, opts) {
      return ambientFetch(this.config, "DELETE", `/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agentId)}`, void 0, opts);
    }
    async ignite(projectId, agentId, prompt, opts) {
      return ambientFetch(this.config, "POST", `/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agentId)}/ignite`, { prompt }, opts);
    }
    async getIgnition(projectId, agentId, opts) {
      return ambientFetch(this.config, "GET", `/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agentId)}/ignition`, void 0, opts);
    }
    async sessions(projectId, agentId, listOpts, opts) {
      const qs = buildQueryString(listOpts);
      return ambientFetch(this.config, "GET", `/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agentId)}/sessions${qs}`, void 0, opts);
    }
  };

  // ../../ambient/platform/platform-api-server/components/ambient-sdk/ts-sdk/src/project_settings_api.ts
  var ProjectSettingsAPI = class {
    constructor(config) {
      this.config = config;
    }
    async create(data, opts) {
      return ambientFetch(this.config, "POST", "/project_settings", data, opts);
    }
    async get(id, opts) {
      return ambientFetch(this.config, "GET", `/project_settings/${id}`, void 0, opts);
    }
    async list(listOpts, opts) {
      const qs = buildQueryString(listOpts);
      return ambientFetch(this.config, "GET", `/project_settings${qs}`, void 0, opts);
    }
    async update(id, patch, opts) {
      return ambientFetch(this.config, "PATCH", `/project_settings/${id}`, patch, opts);
    }
    async delete(id, opts) {
      return ambientFetch(this.config, "DELETE", `/project_settings/${id}`, void 0, opts);
    }
    async *listAll(size = 100, opts) {
      let page = 1;
      while (true) {
        const result = await this.list({ page, size }, opts);
        for (const item of result.items) {
          yield item;
        }
        if (page * size >= result.total) {
          break;
        }
        page++;
      }
    }
  };

  // ../../ambient/platform/platform-api-server/components/ambient-sdk/ts-sdk/src/role_api.ts
  var RoleAPI = class {
    constructor(config) {
      this.config = config;
    }
    async create(data, opts) {
      return ambientFetch(this.config, "POST", "/roles", data, opts);
    }
    async get(id, opts) {
      return ambientFetch(this.config, "GET", `/roles/${id}`, void 0, opts);
    }
    async list(listOpts, opts) {
      const qs = buildQueryString(listOpts);
      return ambientFetch(this.config, "GET", `/roles${qs}`, void 0, opts);
    }
    async update(id, patch, opts) {
      return ambientFetch(this.config, "PATCH", `/roles/${id}`, patch, opts);
    }
    async *listAll(size = 100, opts) {
      let page = 1;
      while (true) {
        const result = await this.list({ page, size }, opts);
        for (const item of result.items) {
          yield item;
        }
        if (page * size >= result.total) {
          break;
        }
        page++;
      }
    }
  };

  // ../../ambient/platform/platform-api-server/components/ambient-sdk/ts-sdk/src/role_binding_api.ts
  var RoleBindingAPI = class {
    constructor(config) {
      this.config = config;
    }
    async create(data, opts) {
      return ambientFetch(this.config, "POST", "/role_bindings", data, opts);
    }
    async get(id, opts) {
      return ambientFetch(this.config, "GET", `/role_bindings/${id}`, void 0, opts);
    }
    async list(listOpts, opts) {
      const qs = buildQueryString(listOpts);
      return ambientFetch(this.config, "GET", `/role_bindings${qs}`, void 0, opts);
    }
    async update(id, patch, opts) {
      return ambientFetch(this.config, "PATCH", `/role_bindings/${id}`, patch, opts);
    }
    async *listAll(size = 100, opts) {
      let page = 1;
      while (true) {
        const result = await this.list({ page, size }, opts);
        for (const item of result.items) {
          yield item;
        }
        if (page * size >= result.total) {
          break;
        }
        page++;
      }
    }
  };

  // ../../ambient/platform/platform-api-server/components/ambient-sdk/ts-sdk/src/session_api.ts
  var SessionAPI = class {
    constructor(config) {
      this.config = config;
    }
    async create(data, opts) {
      return ambientFetch(this.config, "POST", "/sessions", data, opts);
    }
    async get(id, opts) {
      return ambientFetch(this.config, "GET", `/sessions/${id}`, void 0, opts);
    }
    async list(listOpts, opts) {
      const qs = buildQueryString(listOpts);
      return ambientFetch(this.config, "GET", `/sessions${qs}`, void 0, opts);
    }
    async update(id, patch, opts) {
      return ambientFetch(this.config, "PATCH", `/sessions/${id}`, patch, opts);
    }
    async delete(id, opts) {
      return ambientFetch(this.config, "DELETE", `/sessions/${id}`, void 0, opts);
    }
    async updateStatus(id, patch, opts) {
      return ambientFetch(this.config, "PATCH", `/sessions/${id}/status`, patch, opts);
    }
    async start(id, opts) {
      return ambientFetch(this.config, "POST", `/sessions/${id}/start`, void 0, opts);
    }
    async stop(id, opts) {
      return ambientFetch(this.config, "POST", `/sessions/${id}/stop`, void 0, opts);
    }
    async *listAll(size = 100, opts) {
      let page = 1;
      while (true) {
        const result = await this.list({ page, size }, opts);
        for (const item of result.items) {
          yield item;
        }
        if (page * size >= result.total) {
          break;
        }
        page++;
      }
    }
  };

  // ../../ambient/platform/platform-api-server/components/ambient-sdk/ts-sdk/src/session_message_api.ts
  var SessionMessageAPI = class {
    constructor(config) {
      this.config = config;
    }
    async create(data, opts) {
      return ambientFetch(this.config, "POST", "/sessions", data, opts);
    }
    async get(id, opts) {
      return ambientFetch(this.config, "GET", `/sessions/${id}`, void 0, opts);
    }
    async list(listOpts, opts) {
      const qs = buildQueryString(listOpts);
      return ambientFetch(this.config, "GET", `/sessions${qs}`, void 0, opts);
    }
    async *listAll(size = 100, opts) {
      let page = 1;
      while (true) {
        const result = await this.list({ page, size }, opts);
        for (const item of result.items) {
          yield item;
        }
        if (page * size >= result.total) {
          break;
        }
        page++;
      }
    }
  };

  // ../../ambient/platform/platform-api-server/components/ambient-sdk/ts-sdk/src/user_api.ts
  var UserAPI = class {
    constructor(config) {
      this.config = config;
    }
    async create(data, opts) {
      return ambientFetch(this.config, "POST", "/users", data, opts);
    }
    async get(id, opts) {
      return ambientFetch(this.config, "GET", `/users/${id}`, void 0, opts);
    }
    async list(listOpts, opts) {
      const qs = buildQueryString(listOpts);
      return ambientFetch(this.config, "GET", `/users${qs}`, void 0, opts);
    }
    async update(id, patch, opts) {
      return ambientFetch(this.config, "PATCH", `/users/${id}`, patch, opts);
    }
    async *listAll(size = 100, opts) {
      let page = 1;
      while (true) {
        const result = await this.list({ page, size }, opts);
        for (const item of result.items) {
          yield item;
        }
        if (page * size >= result.total) {
          break;
        }
        page++;
      }
    }
  };

  // ../../ambient/platform/platform-api-server/components/ambient-sdk/ts-sdk/src/client.ts
  var AmbientClient = class _AmbientClient {
    constructor(config) {
      if (!config.baseUrl) {
        throw new Error("baseUrl is required");
      }
      if (!config.token) {
        throw new Error("token is required");
      }
      if (config.token.length < 20) {
        throw new Error("token is too short (minimum 20 characters)");
      }
      if (config.token === "YOUR_TOKEN_HERE" || config.token === "PLACEHOLDER_TOKEN") {
        throw new Error("placeholder token is not allowed");
      }
      if (config.project && config.project.length > 63) {
        throw new Error("project name cannot exceed 63 characters");
      }
      const url = new URL(config.baseUrl);
      if (url.protocol !== "http:" && url.protocol !== "https:") {
        throw new Error("only HTTP and HTTPS schemes are supported");
      }
      this.config = {
        ...config,
        baseUrl: config.baseUrl.replace(/\/+$/, "")
      };
      this.inboxMessages = new InboxMessageAPI(this.config);
      this.projects = new ProjectAPI(this.config);
      this.projectAgents = new ProjectAgentAPI(this.config);
      this.projectSettings = new ProjectSettingsAPI(this.config);
      this.roles = new RoleAPI(this.config);
      this.roleBindings = new RoleBindingAPI(this.config);
      this.sessions = new SessionAPI(this.config);
      this.sessionMessages = new SessionMessageAPI(this.config);
      this.users = new UserAPI(this.config);
    }
    static fromEnv() {
      const baseUrl = process.env.AMBIENT_API_URL || "http://localhost:8080";
      const token = process.env.AMBIENT_TOKEN;
      const project = process.env.AMBIENT_PROJECT;
      if (!token) {
        throw new Error("AMBIENT_TOKEN environment variable is required");
      }
      if (!project) {
        throw new Error("AMBIENT_PROJECT environment variable is required");
      }
      return new _AmbientClient({ baseUrl, token, project });
    }
  };

  // ../../ambient/platform/platform-api-server/components/ambient-sdk/ts-sdk/src/inbox_message.ts
  var InboxMessageBuilder = class {
    constructor() {
      this.data = {};
    }
    agentId(value) {
      this.data["agent_id"] = value;
      return this;
    }
    body(value) {
      this.data["body"] = value;
      return this;
    }
    fromAgentId(value) {
      this.data["from_agent_id"] = value;
      return this;
    }
    fromName(value) {
      this.data["from_name"] = value;
      return this;
    }
    build() {
      if (!this.data["agent_id"]) {
        throw new Error("agent_id is required");
      }
      if (!this.data["body"]) {
        throw new Error("body is required");
      }
      return this.data;
    }
  };
  var InboxMessagePatchBuilder = class {
    constructor() {
      this.data = {};
    }
    read(value) {
      this.data["read"] = value;
      return this;
    }
    build() {
      return this.data;
    }
  };

  // ../../ambient/platform/platform-api-server/components/ambient-sdk/ts-sdk/src/project.ts
  var ProjectBuilder = class {
    constructor() {
      this.data = {};
    }
    annotations(value) {
      this.data["annotations"] = value;
      return this;
    }
    description(value) {
      this.data["description"] = value;
      return this;
    }
    labels(value) {
      this.data["labels"] = value;
      return this;
    }
    name(value) {
      this.data["name"] = value;
      return this;
    }
    prompt(value) {
      this.data["prompt"] = value;
      return this;
    }
    status(value) {
      this.data["status"] = value;
      return this;
    }
    build() {
      if (!this.data["name"]) {
        throw new Error("name is required");
      }
      return this.data;
    }
  };
  var ProjectPatchBuilder = class {
    constructor() {
      this.data = {};
    }
    annotations(value) {
      this.data["annotations"] = value;
      return this;
    }
    description(value) {
      this.data["description"] = value;
      return this;
    }
    labels(value) {
      this.data["labels"] = value;
      return this;
    }
    name(value) {
      this.data["name"] = value;
      return this;
    }
    prompt(value) {
      this.data["prompt"] = value;
      return this;
    }
    status(value) {
      this.data["status"] = value;
      return this;
    }
    build() {
      return this.data;
    }
  };

  // ../../ambient/platform/platform-api-server/components/ambient-sdk/ts-sdk/src/project_agent.ts
  var ProjectAgentBuilder = class {
    constructor() {
      this.data = {};
    }
    annotations(value) {
      this.data["annotations"] = value;
      return this;
    }
    labels(value) {
      this.data["labels"] = value;
      return this;
    }
    name(value) {
      this.data["name"] = value;
      return this;
    }
    projectId(value) {
      this.data["project_id"] = value;
      return this;
    }
    prompt(value) {
      this.data["prompt"] = value;
      return this;
    }
    build() {
      if (!this.data["name"]) {
        throw new Error("name is required");
      }
      if (!this.data["project_id"]) {
        throw new Error("project_id is required");
      }
      return this.data;
    }
  };
  var ProjectAgentPatchBuilder = class {
    constructor() {
      this.data = {};
    }
    annotations(value) {
      this.data["annotations"] = value;
      return this;
    }
    labels(value) {
      this.data["labels"] = value;
      return this;
    }
    name(value) {
      this.data["name"] = value;
      return this;
    }
    prompt(value) {
      this.data["prompt"] = value;
      return this;
    }
    build() {
      return this.data;
    }
  };

  // ../../ambient/platform/platform-api-server/components/ambient-sdk/ts-sdk/src/project_settings.ts
  var ProjectSettingsBuilder = class {
    constructor() {
      this.data = {};
    }
    groupAccess(value) {
      this.data["group_access"] = value;
      return this;
    }
    projectId(value) {
      this.data["project_id"] = value;
      return this;
    }
    repositories(value) {
      this.data["repositories"] = value;
      return this;
    }
    build() {
      if (!this.data["project_id"]) {
        throw new Error("project_id is required");
      }
      return this.data;
    }
  };
  var ProjectSettingsPatchBuilder = class {
    constructor() {
      this.data = {};
    }
    groupAccess(value) {
      this.data["group_access"] = value;
      return this;
    }
    projectId(value) {
      this.data["project_id"] = value;
      return this;
    }
    repositories(value) {
      this.data["repositories"] = value;
      return this;
    }
    build() {
      return this.data;
    }
  };

  // ../../ambient/platform/platform-api-server/components/ambient-sdk/ts-sdk/src/role.ts
  var RoleBuilder = class {
    constructor() {
      this.data = {};
    }
    builtIn(value) {
      this.data["built_in"] = value;
      return this;
    }
    description(value) {
      this.data["description"] = value;
      return this;
    }
    displayName(value) {
      this.data["display_name"] = value;
      return this;
    }
    name(value) {
      this.data["name"] = value;
      return this;
    }
    permissions(value) {
      this.data["permissions"] = value;
      return this;
    }
    build() {
      if (!this.data["name"]) {
        throw new Error("name is required");
      }
      return this.data;
    }
  };
  var RolePatchBuilder = class {
    constructor() {
      this.data = {};
    }
    builtIn(value) {
      this.data["built_in"] = value;
      return this;
    }
    description(value) {
      this.data["description"] = value;
      return this;
    }
    displayName(value) {
      this.data["display_name"] = value;
      return this;
    }
    name(value) {
      this.data["name"] = value;
      return this;
    }
    permissions(value) {
      this.data["permissions"] = value;
      return this;
    }
    build() {
      return this.data;
    }
  };

  // ../../ambient/platform/platform-api-server/components/ambient-sdk/ts-sdk/src/role_binding.ts
  var RoleBindingBuilder = class {
    constructor() {
      this.data = {};
    }
    roleId(value) {
      this.data["role_id"] = value;
      return this;
    }
    scope(value) {
      this.data["scope"] = value;
      return this;
    }
    scopeId(value) {
      this.data["scope_id"] = value;
      return this;
    }
    userId(value) {
      this.data["user_id"] = value;
      return this;
    }
    build() {
      if (!this.data["role_id"]) {
        throw new Error("role_id is required");
      }
      if (!this.data["scope"]) {
        throw new Error("scope is required");
      }
      if (!this.data["user_id"]) {
        throw new Error("user_id is required");
      }
      return this.data;
    }
  };
  var RoleBindingPatchBuilder = class {
    constructor() {
      this.data = {};
    }
    roleId(value) {
      this.data["role_id"] = value;
      return this;
    }
    scope(value) {
      this.data["scope"] = value;
      return this;
    }
    scopeId(value) {
      this.data["scope_id"] = value;
      return this;
    }
    userId(value) {
      this.data["user_id"] = value;
      return this;
    }
    build() {
      return this.data;
    }
  };

  // ../../ambient/platform/platform-api-server/components/ambient-sdk/ts-sdk/src/session.ts
  var SessionBuilder = class {
    constructor() {
      this.data = {};
    }
    agentId(value) {
      this.data["agent_id"] = value;
      return this;
    }
    annotations(value) {
      this.data["annotations"] = value;
      return this;
    }
    assignedUserId(value) {
      this.data["assigned_user_id"] = value;
      return this;
    }
    botAccountName(value) {
      this.data["bot_account_name"] = value;
      return this;
    }
    environmentVariables(value) {
      this.data["environment_variables"] = value;
      return this;
    }
    labels(value) {
      this.data["labels"] = value;
      return this;
    }
    llmMaxTokens(value) {
      this.data["llm_max_tokens"] = value;
      return this;
    }
    llmModel(value) {
      this.data["llm_model"] = value;
      return this;
    }
    llmTemperature(value) {
      this.data["llm_temperature"] = value;
      return this;
    }
    name(value) {
      this.data["name"] = value;
      return this;
    }
    parentSessionId(value) {
      this.data["parent_session_id"] = value;
      return this;
    }
    projectId(value) {
      this.data["project_id"] = value;
      return this;
    }
    prompt(value) {
      this.data["prompt"] = value;
      return this;
    }
    repoUrl(value) {
      this.data["repo_url"] = value;
      return this;
    }
    repos(value) {
      this.data["repos"] = value;
      return this;
    }
    resourceOverrides(value) {
      this.data["resource_overrides"] = value;
      return this;
    }
    timeout(value) {
      this.data["timeout"] = value;
      return this;
    }
    workflowId(value) {
      this.data["workflow_id"] = value;
      return this;
    }
    build() {
      if (!this.data["name"]) {
        throw new Error("name is required");
      }
      return this.data;
    }
  };
  var SessionPatchBuilder = class {
    constructor() {
      this.data = {};
    }
    annotations(value) {
      this.data["annotations"] = value;
      return this;
    }
    assignedUserId(value) {
      this.data["assigned_user_id"] = value;
      return this;
    }
    botAccountName(value) {
      this.data["bot_account_name"] = value;
      return this;
    }
    environmentVariables(value) {
      this.data["environment_variables"] = value;
      return this;
    }
    labels(value) {
      this.data["labels"] = value;
      return this;
    }
    llmMaxTokens(value) {
      this.data["llm_max_tokens"] = value;
      return this;
    }
    llmModel(value) {
      this.data["llm_model"] = value;
      return this;
    }
    llmTemperature(value) {
      this.data["llm_temperature"] = value;
      return this;
    }
    name(value) {
      this.data["name"] = value;
      return this;
    }
    parentSessionId(value) {
      this.data["parent_session_id"] = value;
      return this;
    }
    prompt(value) {
      this.data["prompt"] = value;
      return this;
    }
    repoUrl(value) {
      this.data["repo_url"] = value;
      return this;
    }
    repos(value) {
      this.data["repos"] = value;
      return this;
    }
    resourceOverrides(value) {
      this.data["resource_overrides"] = value;
      return this;
    }
    timeout(value) {
      this.data["timeout"] = value;
      return this;
    }
    workflowId(value) {
      this.data["workflow_id"] = value;
      return this;
    }
    build() {
      return this.data;
    }
  };
  var SessionStatusPatchBuilder = class {
    constructor() {
      this.data = {};
    }
    completionTime(value) {
      this.data["completion_time"] = value;
      return this;
    }
    conditions(value) {
      this.data["conditions"] = value;
      return this;
    }
    kubeCrUid(value) {
      this.data["kube_cr_uid"] = value;
      return this;
    }
    kubeNamespace(value) {
      this.data["kube_namespace"] = value;
      return this;
    }
    phase(value) {
      this.data["phase"] = value;
      return this;
    }
    reconciledRepos(value) {
      this.data["reconciled_repos"] = value;
      return this;
    }
    reconciledWorkflow(value) {
      this.data["reconciled_workflow"] = value;
      return this;
    }
    sdkRestartCount(value) {
      this.data["sdk_restart_count"] = value;
      return this;
    }
    sdkSessionId(value) {
      this.data["sdk_session_id"] = value;
      return this;
    }
    startTime(value) {
      this.data["start_time"] = value;
      return this;
    }
    build() {
      return this.data;
    }
  };

  // ../../ambient/platform/platform-api-server/components/ambient-sdk/ts-sdk/src/session_message.ts
  var SessionMessageBuilder = class {
    constructor() {
      this.data = {};
    }
    eventType(value) {
      this.data["event_type"] = value;
      return this;
    }
    payload(value) {
      this.data["payload"] = value;
      return this;
    }
    build() {
      return this.data;
    }
  };
  var SessionMessagePatchBuilder = class {
    constructor() {
      this.data = {};
    }
    build() {
      return this.data;
    }
  };

  // ../../ambient/platform/platform-api-server/components/ambient-sdk/ts-sdk/src/user.ts
  var UserBuilder = class {
    constructor() {
      this.data = {};
    }
    email(value) {
      this.data["email"] = value;
      return this;
    }
    name(value) {
      this.data["name"] = value;
      return this;
    }
    username(value) {
      this.data["username"] = value;
      return this;
    }
    build() {
      if (!this.data["name"]) {
        throw new Error("name is required");
      }
      if (!this.data["username"]) {
        throw new Error("username is required");
      }
      return this.data;
    }
  };
  var UserPatchBuilder = class {
    constructor() {
      this.data = {};
    }
    email(value) {
      this.data["email"] = value;
      return this;
    }
    name(value) {
      this.data["name"] = value;
      return this;
    }
    username(value) {
      this.data["username"] = value;
      return this;
    }
    build() {
      return this.data;
    }
  };
  return __toCommonJS(index_exports);
})();
