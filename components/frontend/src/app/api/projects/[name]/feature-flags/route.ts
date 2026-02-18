import { BACKEND_URL } from "@/lib/config";
import { buildForwardHeadersAsync } from "@/lib/auth";

/**
 * GET /api/projects/:projectName/feature-flags
 * Proxies to backend to list all feature flags from Unleash
 */
export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name: projectName } = await params;
    const headers = await buildForwardHeadersAsync(request);

    const response = await fetch(
      `${BACKEND_URL}/projects/${encodeURIComponent(projectName)}/feature-flags`,
      { headers }
    );

    const data = await response.text();

    return new Response(data, {
      status: response.status,
      headers: { "Content-Type": "application/json" },
    });
  } catch (error) {
    console.error("Failed to fetch feature flags:", error);
    return Response.json(
      { error: "Failed to fetch feature flags" },
      { status: 500 }
    );
  }
}
