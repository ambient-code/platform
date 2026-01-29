import { BACKEND_URL } from "@/lib/config";
import { NextRequest } from "next/server";

export async function GET(request: NextRequest) {
  try {
    // Forward query parameters to backend (e.g., project param for GitHub token lookup)
    const searchParams = request.nextUrl.searchParams;
    const queryString = searchParams.toString();
    const url = queryString
      ? `${BACKEND_URL}/workflows/ootb?${queryString}`
      : `${BACKEND_URL}/workflows/ootb`;

    // Forward authorization header if present (enables GitHub token lookup for better rate limits)
    const headers: HeadersInit = {
      "Content-Type": "application/json",
    };
    const authHeader = request.headers.get("Authorization");
    if (authHeader) {
      headers["Authorization"] = authHeader;
    }

    const response = await fetch(url, {
      method: 'GET',
      headers,
    });

    // Forward the response from backend
    const data = await response.text();

    return new Response(data, {
      status: response.status,
      headers: {
        "Content-Type": "application/json",
      },
    });
  } catch (error) {
    console.error("Failed to fetch OOTB workflows:", error);
    return new Response(
      JSON.stringify({ error: "Failed to fetch OOTB workflows" }),
      { 
        status: 500,
        headers: { "Content-Type": "application/json" }
      }
    );
  }
}

