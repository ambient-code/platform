import { BACKEND_URL } from "@/lib/config";
import { buildForwardHeadersAsync } from "@/lib/auth";

/**
 * POST /api/projects/:projectName/feature-flags/:flagName/enable
 * Proxies to backend to enable a feature flag in Unleash
 */
export async function POST(
  request: Request,
  { params }: { params: Promise<{ name: string; flagName: string }> }
) {
  try {
    const { name: projectName, flagName } = await params;
    const headers = await buildForwardHeadersAsync(request);

    const response = await fetch(
      `${BACKEND_URL}/projects/${encodeURIComponent(projectName)}/feature-flags/${encodeURIComponent(flagName)}/enable`,
      {
        method: "POST",
        headers,
      }
    );

    const data = await response.text();

    return new Response(data, {
      status: response.status,
      headers: { "Content-Type": "application/json" },
    });
  } catch (error) {
    console.error("Failed to enable feature flag:", error);
    return Response.json(
      { error: "Failed to enable feature flag" },
      { status: 500 }
    );
  }
}
