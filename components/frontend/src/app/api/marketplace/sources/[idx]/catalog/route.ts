import { BACKEND_URL } from "@/lib/config";
import { buildForwardHeadersAsync } from "@/lib/auth";

export async function GET(
  request: Request,
  { params }: { params: Promise<{ idx: string }> }
) {
  try {
    const { idx } = await params;
    const headers = await buildForwardHeadersAsync(request);
    const response = await fetch(`${BACKEND_URL}/marketplace/sources/${idx}/catalog`, { headers });
    const data = await response.text();
    return new Response(data, {
      status: response.status,
      headers: { "Content-Type": "application/json" },
    });
  } catch (error) {
    console.error("Failed to fetch marketplace catalog:", error);
    return Response.json({ error: "Failed to fetch marketplace catalog" }, { status: 500 });
  }
}
