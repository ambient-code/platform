import { NextRequest, NextResponse } from "next/server";
import { API_SERVER_URL } from "@/lib/config";
import { buildForwardHeadersAsync } from "@/lib/auth";

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    await params;
    const intelligenceId = request.nextUrl.searchParams.get("intelligence_id");

    if (!intelligenceId) {
      return NextResponse.json(
        { error: "intelligence_id query parameter is required" },
        { status: 400 }
      );
    }

    if (!/^[a-zA-Z0-9]+$/.test(intelligenceId)) {
      return NextResponse.json(
        { error: "Invalid intelligence_id format" },
        { status: 400 }
      );
    }

    const headers = await buildForwardHeadersAsync(request);
    const response = await fetch(
      `${API_SERVER_URL}/api/ambient/v1/repo_intelligences/${intelligenceId}/findings?search=status%20%3D%20'active'`,
      {
        method: "GET",
        headers,
      }
    );

    const data = await response.text();
    return new NextResponse(data, {
      status: response.status,
      headers: { "Content-Type": "application/json" },
    });
  } catch (error) {
    console.error("Failed to fetch repo findings:", error);
    return NextResponse.json(
      { error: "Failed to fetch repo findings" },
      { status: 500 }
    );
  }
}
