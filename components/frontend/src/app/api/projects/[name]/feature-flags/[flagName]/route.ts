import { BACKEND_URL } from "@/lib/config";
import { buildForwardHeadersAsync } from "@/lib/auth";

/**
 * GET /api/projects/:projectName/feature-flags/:flagName
 * Proxies to backend to get a specific feature flag from Unleash
 */
export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string; flagName: string }> }
) {
  try {
    const { name: projectName, flagName } = await params;
    const headers = await buildForwardHeadersAsync(request);

    const response = await fetch(
      `${BACKEND_URL}/projects/${encodeURIComponent(projectName)}/feature-flags/${encodeURIComponent(flagName)}`,
      { headers }
    );

    const data = await response.text();

    return new Response(data, {
      status: response.status,
      headers: { "Content-Type": "application/json" },
    });
  } catch (error) {
    console.error("Failed to fetch feature flag:", error);
    return Response.json(
      { error: "Failed to fetch feature flag" },
      { status: 500 }
    );
  }
}
