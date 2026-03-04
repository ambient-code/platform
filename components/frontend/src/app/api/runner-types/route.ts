import { BACKEND_URL } from "@/lib/config";
import { buildForwardHeadersAsync } from "@/lib/auth";

export async function GET(request: Request) {
  try {
    const headers = await buildForwardHeadersAsync(request);

    const response = await fetch(`${BACKEND_URL}/runner-types`, {
      method: "GET",
      headers,
    });

    const data = await response.text();

    return new Response(data, {
      status: response.status,
      headers: {
        "Content-Type": "application/json",
      },
    });
  } catch (error) {
    console.error("Failed to fetch runner types:", error);
    return new Response(
      JSON.stringify({ error: "Failed to fetch runner types" }),
      {
        status: 500,
        headers: { "Content-Type": "application/json" },
      }
    );
  }
}
