import {
  CopilotRuntime,
  ExperimentalEmptyAdapter,
  copilotRuntimeNextJSAppRouterEndpoint,
} from "@copilotkit/runtime";
import { HttpAgent } from "@ag-ui/client";
import { NextRequest } from "next/server";

/**
 * CopilotKit API route for session chat.
 *
 * Creates an HttpAgent that points at the backend's AG-UI runner proxy.
 * The backend handles auth, RBAC, and proxying to the actual runner pod.
 */
export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ project: string; session: string }> }
) {
  const { project, session } = await params;

  // The backend proxy URL — the backend handles auth forwarding
  const backendUrl =
    process.env.NEXT_PUBLIC_API_URL || process.env.BACKEND_URL || "";
  const runnerProxyUrl = `${backendUrl}/api/projects/${project}/agentic-sessions/${session}/agui/run`;

  const agent = new HttpAgent({
    url: runnerProxyUrl,
  });

  const runtime = new CopilotRuntime({
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    agents: { session: agent as any },
  });

  const { handleRequest } = copilotRuntimeNextJSAppRouterEndpoint({
    runtime,
    serviceAdapter: new ExperimentalEmptyAdapter(),
    endpoint: `/api/copilotkit/${project}/${session}`,
  });

  return handleRequest(request);
}
