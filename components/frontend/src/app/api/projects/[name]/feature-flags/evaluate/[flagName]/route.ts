import { BACKEND_URL } from "@/lib/config";
import { buildForwardHeadersAsync } from "@/lib/auth";

/**
 * GET /api/projects/:projectName/feature-flags/evaluate/:flagName
 * Evaluates a feature flag for a workspace (ConfigMap override > Unleash default)
 */
export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string; flagName: string }> }
) {
  try {
    const { name: projectName, flagName } = await params;
    const headers = await buildForwardHeadersAsync(request);

    const response = await fetch(
      `${BACKEND_URL}/projects/${encodeURIComponent(projectName)}/feature-flags/evaluate/${encodeURIComponent(flagName)}`,
      { headers }
    );

    const data = await response.text();

    return new Response(data, {
      status: response.status,
      headers: { "Content-Type": "application/json" },
    });
  } catch (error) {
    console.error("Failed to evaluate feature flag:", error);
    return Response.json(
      { error: "Failed to evaluate feature flag" },
      { status: 500 }
    );
  }
}
