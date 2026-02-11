import {
  CopilotRuntime,
  ExperimentalEmptyAdapter,
  copilotRuntimeNextJSAppRouterEndpoint,
} from "@copilotkit/runtime";
import {
  AgentRunner,
  type AgentRunnerRunRequest,
  type AgentRunnerConnectRequest,
  type AgentRunnerIsRunningRequest,
  type AgentRunnerStopRequest,
} from "@copilotkitnext/runtime";
import { HttpAgent } from "@ag-ui/client";
import type { BaseEvent, RunAgentInput } from "@ag-ui/core";
import { NextRequest, NextResponse } from "next/server";
import { Observable } from "rxjs";
import { BACKEND_URL } from "@/lib/config";
import { buildForwardHeadersAsync } from "@/lib/auth";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

// ---------------------------------------------------------------------------
// BackendPersistedRunner — custom AgentRunner backed by our Go backend.
//
// The Go backend persists every AG-UI event to a JSONL log as events
// stream through it.  This runner uses the backend as the persistence
// layer instead of CopilotKit's InMemoryAgentRunner (which stalls on
// long-running agents due to lastValueFrom blocking in connectAgent).
//
// run():     Delegates to the HttpAgent (→ backend proxy → runner pod).
// connect(): POSTs to the backend with empty messages.  The backend
//            replays compacted events from the JSONL, completing
//            immediately so connectAgent() resolves and messages render.
// isRunning(): Checks session phase via the backend API.
// stop():      Sends interrupt signal via the backend API.
// ---------------------------------------------------------------------------

class BackendPersistedRunner extends AgentRunner {
  private _runUrl: string;
  private _interruptUrl: string;
  private _sessionUrl: string;
  private _headers: Record<string, string>;

  constructor(backendUrl: string, headers: Record<string, string>) {
    super();
    this._runUrl = backendUrl;
    const base = backendUrl.replace(/\/agui\/run$/, "");
    this._interruptUrl = `${base}/agui/interrupt`;
    this._sessionUrl = base;
    this._headers = headers;
  }

  run(request: AgentRunnerRunRequest): Observable<BaseEvent> {
    return request.agent.run(request.input);
  }

  connect(request: AgentRunnerConnectRequest): Observable<BaseEvent> {
    const { threadId } = request;

    const input: RunAgentInput = {
      threadId: threadId ?? "",
      runId: crypto.randomUUID(),
      messages: [],
      tools: [],
      context: [],
      forwardedProps: {},
      state: {},
    };

    return new Observable<BaseEvent>((subscriber) => {
      fetch(this._runUrl, {
        method: "POST",
        headers: {
          ...this._headers,
          "Content-Type": "application/json",
          Accept: "text/event-stream",
        },
        body: JSON.stringify(input),
      })
        .then(async (resp) => {
          if (!resp.ok || !resp.body) {
            const status = resp?.status ?? 0;
            console.error(`[BackendPersistedRunner] connect failed: HTTP ${status}`);
            subscriber.error(
              new Error(`Connect failed: HTTP ${status}`)
            );
            return;
          }

          const reader = resp.body.getReader();
          const decoder = new TextDecoder();
          let buffer = "";

          while (true) {
            const { done, value } = await reader.read();
            if (done) break;

            buffer += decoder.decode(value, { stream: true });
            const lines = buffer.split("\n");
            buffer = lines.pop() ?? "";

            for (const line of lines) {
              const trimmed = line.trim();
              if (trimmed.startsWith("data: ")) {
                try {
                  const event = JSON.parse(trimmed.slice(6)) as BaseEvent;
                  subscriber.next(event);
                } catch {
                  // skip unparseable
                }
              }
            }
          }
          subscriber.complete();
        })
        .catch((err) => {
          console.error("[BackendPersistedRunner] connect error:", err);
          subscriber.error(err);
        });
    });
  }

  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  async isRunning(_request: AgentRunnerIsRunningRequest): Promise<boolean> {
    try {
      const resp = await fetch(this._sessionUrl, {
        headers: this._headers,
      });
      if (!resp.ok) return false;
      const data = await resp.json();
      return data?.status?.phase === "Running";
    } catch {
      return false;
    }
  }

  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  async stop(_request: AgentRunnerStopRequest): Promise<boolean | undefined> {
    try {
      const resp = await fetch(this._interruptUrl, {
        method: "POST",
        headers: { ...this._headers, "Content-Type": "application/json" },
        body: JSON.stringify({}),
      });
      return resp.ok;
    } catch {
      return false;
    }
  }
}

// ---------------------------------------------------------------------------
// Route-level connect response cache.
//
// CopilotKit fires ~20 agent/connect HTTP requests on mount.  Each one
// goes through CopilotRuntime → runner.connect() → backend fetch.
// Since every connect replays the same JSONL, the result is identical.
//
// We cache the FIRST connect response (the raw body bytes + headers)
// per session for CONNECT_CACHE_TTL_MS.  Subsequent connects within
// the window get the cached response instantly — no backend round-trip,
// no CopilotRuntime overhead.  Non-connect requests (agent/run, etc.)
// bypass the cache entirely.
// ---------------------------------------------------------------------------

const CONNECT_CACHE_TTL_MS = 3_000;

type CachedConnect = {
  body: ArrayBuffer;
  headers: [string, string][];
  status: number;
  ts: number;
};

const connectCache = new Map<string, CachedConnect>();
const connectInflight = new Map<string, Promise<CachedConnect>>();

// ---------------------------------------------------------------------------
// Route handler
// ---------------------------------------------------------------------------

export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ project: string; session: string }> }
) {
  const { project, session } = await params;
  const cacheKey = `${project}:${session}`;

  // Read the body once — we need it both for method detection and
  // for forwarding to CopilotKit runtime.
  const bodyBytes = await request.arrayBuffer();

  // Detect agent/connect requests by peeking at the JSON body.
  let isConnect = false;
  try {
    const parsed = JSON.parse(new TextDecoder().decode(bodyBytes));
    isConnect = parsed?.method === "agent/connect";
  } catch {
    // Not JSON or malformed — fall through to normal handling.
  }

  if (isConnect) {
    // 1. Serve from cache if fresh.
    const cached = connectCache.get(cacheKey);
    if (cached && Date.now() - cached.ts < CONNECT_CACHE_TTL_MS) {
      return new NextResponse(cached.body, {
        status: cached.status,
        headers: cached.headers,
      });
    }

    // 2. If another connect is already in-flight, wait for it and
    //    return the same cached result (coalesces concurrent requests).
    const inflight = connectInflight.get(cacheKey);
    if (inflight) {
      const result = await inflight;
      return new NextResponse(result.body, {
        status: result.status,
        headers: result.headers,
      });
    }

    // 3. First request — go through CopilotKit, capture the response.
    //    CopilotKit returns a streaming SSE Response whose ReadableStream
    //    may contain string chunks (not Uint8Array), so resp.arrayBuffer()
    //    throws "Received non-Uint8Array chunk".  We manually drain the
    //    stream and normalise each chunk to Uint8Array.
    const promise = handleCopilotRequest(project, session, bodyBytes, request)
      .then(async (resp) => {
        const encoder = new TextEncoder();
        const chunks: Uint8Array[] = [];
        let totalLen = 0;

        if (resp.body) {
          const reader = resp.body.getReader();
          while (true) {
            const { done, value } = await reader.read();
            if (done) break;
            const chunk = typeof value === "string"
              ? encoder.encode(value)
              : (value as Uint8Array);
            chunks.push(chunk);
            totalLen += chunk.length;
          }
        }

        const body = new Uint8Array(totalLen);
        let offset = 0;
        for (const c of chunks) {
          body.set(c, offset);
          offset += c.length;
        }

        const headers: [string, string][] = [];
        resp.headers.forEach((v, k) => headers.push([k, v]));
        const entry: CachedConnect = {
          body: body.buffer as ArrayBuffer,
          headers,
          status: resp.status,
          ts: Date.now(),
        };
        connectCache.set(cacheKey, entry);
        connectInflight.delete(cacheKey);
        return entry;
      })
      .catch((err) => {
        connectInflight.delete(cacheKey);
        throw err;
      });

    connectInflight.set(cacheKey, promise);
    const result = await promise;
    return new NextResponse(result.body, {
      status: result.status,
      headers: result.headers,
    });
  }

  // Non-connect requests (agent/run, etc.) — pass through normally.
  return handleCopilotRequest(project, session, bodyBytes, request);
}

async function handleCopilotRequest(
  project: string,
  session: string,
  bodyBytes: ArrayBuffer,
  request: NextRequest,
): Promise<Response> {
  const runnerProxyUrl = `${BACKEND_URL}/projects/${encodeURIComponent(project)}/agentic-sessions/${encodeURIComponent(session)}/agui/run`;
  const forwardHeaders = await buildForwardHeadersAsync(request);

  const agent = new HttpAgent({
    url: runnerProxyUrl,
    headers: forwardHeaders,
  });

  const copilotRuntime = new CopilotRuntime({
    // eslint-disable-next-line @typescript-eslint/no-explicit-any -- AbstractAgent version mismatch
    agents: { [session]: agent as any },
    runner: new BackendPersistedRunner(runnerProxyUrl, forwardHeaders),
  });

  const { handleRequest } = copilotRuntimeNextJSAppRouterEndpoint({
    runtime: copilotRuntime,
    serviceAdapter: new ExperimentalEmptyAdapter(),
    endpoint: `/api/copilotkit/${project}/${session}`,
  });

  const cleanHeaders = new Headers(request.headers);
  cleanHeaders.delete("authorization");
  cleanHeaders.delete("x-forwarded-access-token");
  cleanHeaders.delete("x-forwarded-user");
  cleanHeaders.delete("x-forwarded-email");
  cleanHeaders.delete("x-forwarded-preferred-username");
  cleanHeaders.delete("x-forwarded-groups");

  const cleanRequest = new NextRequest(request.url, {
    method: request.method,
    headers: cleanHeaders,
    body: bodyBytes,
  });

  return handleRequest(cleanRequest);
}
