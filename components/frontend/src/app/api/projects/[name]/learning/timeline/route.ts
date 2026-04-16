import { BACKEND_URL } from "@/lib/config";
import { buildForwardHeadersAsync } from "@/lib/auth";

export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name } = await params;
    const headers = await buildForwardHeadersAsync(request);
    const url = new URL(request.url);
    const page = url.searchParams.get("page") || "1";
    const pageSize = url.searchParams.get("pageSize") || "20";

    const response = await fetch(
      `${BACKEND_URL}/projects/${encodeURIComponent(name)}/learning/timeline?page=${page}&pageSize=${pageSize}`,
      { headers }
    );

    if (!response.ok) {
      const errorData = await response
        .json()
        .catch(() => ({ error: "Unknown error" }));
      return Response.json(errorData, { status: response.status });
    }

    const data = await response.json();
    return Response.json(data);
  } catch (error) {
    console.error("Error fetching learning timeline:", error);
    return Response.json(
      { error: "Failed to fetch learning timeline" },
      { status: 500 }
    );
  }
}
