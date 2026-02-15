import {
  CopilotRuntime,
  ExperimentalEmptyAdapter,
  copilotRuntimeNextJSAppRouterEndpoint,
} from "@copilotkit/runtime";
import { InMemoryAgentRunner } from "@copilotkitnext/runtime";
import { HttpAgent } from "@ag-ui/client";
import { NextRequest } from "next/server";
import { BACKEND_URL } from "@/lib/config";
import { buildForwardHeadersAsync } from "@/lib/auth";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

// ---------------------------------------------------------------------------
// Route handler
//
// Uses InMemoryAgentRunner (the AG-UI reference implementation) for
// persistence and reconnection.  Events are stored in a module-level
// global store keyed by threadId, surviving across requests in the
// same Node.js process.
//
// On run():     HttpAgent POSTs to backend proxy → runner.  Events
//               are collected in the global store automatically.
// On connect(): Historic events are replayed.  If a run is active,
//               the stream stays open and forwards live events.
//
// The backend proxy still persists events to JSONL for cross-restart
// recovery, but is no longer in the reconnect hot path.
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
    runner: new InMemoryAgentRunner(),
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
