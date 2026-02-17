import { BACKEND_URL } from "@/lib/config";
import { buildForwardHeadersAsync } from "@/lib/auth";

/**
 * PUT /api/projects/:projectName/feature-flags/:flagName/override
 * Sets a workspace-scoped override for a feature flag
 */
export async function PUT(
  request: Request,
  { params }: { params: Promise<{ name: string; flagName: string }> }
) {
  try {
    const { name: projectName, flagName } = await params;
    const headers = await buildForwardHeadersAsync(request);
    const body = await request.text();

    const response = await fetch(
      `${BACKEND_URL}/projects/${encodeURIComponent(projectName)}/feature-flags/${encodeURIComponent(flagName)}/override`,
      {
        method: "PUT",
        headers,
        body,
      }
    );

    const data = await response.text();

    return new Response(data, {
      status: response.status,
      headers: { "Content-Type": "application/json" },
    });
  } catch (error) {
    console.error("Failed to set feature flag override:", error);
    return Response.json(
      { error: "Failed to set feature flag override" },
      { status: 500 }
    );
  }
}

/**
 * DELETE /api/projects/:projectName/feature-flags/:flagName/override
 * Removes a workspace-scoped override, reverting to Unleash default
 */
export async function DELETE(
  request: Request,
  { params }: { params: Promise<{ name: string; flagName: string }> }
) {
  try {
    const { name: projectName, flagName } = await params;
    const headers = await buildForwardHeadersAsync(request);

    const response = await fetch(
      `${BACKEND_URL}/projects/${encodeURIComponent(projectName)}/feature-flags/${encodeURIComponent(flagName)}/override`,
      {
        method: "DELETE",
        headers,
      }
    );

    const data = await response.text();

    return new Response(data, {
      status: response.status,
      headers: { "Content-Type": "application/json" },
    });
  } catch (error) {
    console.error("Failed to remove feature flag override:", error);
    return Response.json(
      { error: "Failed to remove feature flag override" },
      { status: 500 }
    );
  }
}
