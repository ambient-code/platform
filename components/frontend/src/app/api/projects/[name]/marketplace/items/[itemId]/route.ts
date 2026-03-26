import { BACKEND_URL } from "@/lib/config";
import { buildForwardHeadersAsync } from "@/lib/auth";

export async function DELETE(
  request: Request,
  { params }: { params: Promise<{ name: string; itemId: string }> }
) {
  try {
    const { name: projectName, itemId } = await params;
    const url = new URL(request.url);
    const queryString = url.search;
    const headers = await buildForwardHeadersAsync(request);

    const response = await fetch(
      `${BACKEND_URL}/projects/${encodeURIComponent(projectName)}/marketplace/items/${encodeURIComponent(itemId)}${queryString}`,
      { method: "DELETE", headers }
    );
    const data = await response.text();
    return new Response(data, {
      status: response.status,
      headers: { "Content-Type": "application/json" },
    });
  } catch (error) {
    console.error("Failed to uninstall item:", error);
    return Response.json({ error: "Failed to uninstall item" }, { status: 500 });
  }
}
