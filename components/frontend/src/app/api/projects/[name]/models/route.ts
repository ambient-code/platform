import { BACKEND_URL } from "@/lib/config";
import { buildForwardHeadersAsync } from "@/lib/auth";

/**
 * GET /api/projects/:projectName/models?provider=...
 * Proxies to backend to list available models with workspace overrides.
 * Optional provider query param filters by model provider.
 */
export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name: projectName } = await params;
    const headers = await buildForwardHeadersAsync(request);

    const url = new URL(request.url);
    const provider = url.searchParams.get("provider");
    const backendParams = provider ? `?provider=${encodeURIComponent(provider)}` : "";

    const response = await fetch(
      `${BACKEND_URL}/projects/${encodeURIComponent(projectName)}/models${backendParams}`,
      { headers }
    );

    const data = await response.text();

    return new Response(data, {
      status: response.status,
      headers: { "Content-Type": "application/json" },
    });
  } catch (error) {
    console.error("Failed to fetch models:", error);
    return Response.json(
      { error: "Failed to fetch models" },
      { status: 500 }
    );
  }
}
