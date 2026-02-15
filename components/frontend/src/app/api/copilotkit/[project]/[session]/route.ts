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
import { NextRequest } from "next/server";
import { Observable } from "rxjs";
import { BACKEND_URL } from "@/lib/config";
import { buildForwardHeadersAsync } from "@/lib/auth";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

// ---------------------------------------------------------------------------
// HttpConnectRunner — replaces InMemoryAgentRunner
//
// Delegates run() to the agent.  Implements connect() by POSTing to the
// backend with empty messages — the backend compacts its event log and
// returns a MESSAGES_SNAPSHOT with the conversation history.
// ---------------------------------------------------------------------------

class HttpConnectRunner extends AgentRunner {
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
      threadId,
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
            subscriber.complete();
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
        .catch(() => {
          subscriber.complete();
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
// Route handler
// ---------------------------------------------------------------------------

export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ project: string; session: string }> }
) {
  const { project, session } = await params;

  const runnerProxyUrl = `${BACKEND_URL}/projects/${encodeURIComponent(project)}/agentic-sessions/${encodeURIComponent(session)}/agui/run`;

  const bodyBytes = await request.arrayBuffer();
  const forwardHeaders = await buildForwardHeadersAsync(request);

  const agent = new HttpAgent({
    url: runnerProxyUrl,
    headers: forwardHeaders,
  });

  const copilotRuntime = new CopilotRuntime({
    // eslint-disable-next-line @typescript-eslint/no-explicit-any -- AbstractAgent version mismatch
    agents: { session: agent as any },
    runner: new HttpConnectRunner(runnerProxyUrl, forwardHeaders),
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
