import { NextRequest, NextResponse } from "next/server";
import { API_SERVER_URL } from "@/lib/config";
import { buildForwardHeadersAsync } from "@/lib/auth";

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name: projectName } = await params;
    const repoUrl = request.nextUrl.searchParams.get("repo_url");

    if (!repoUrl) {
      return NextResponse.json(
        { error: "repo_url query parameter is required" },
        { status: 400 }
      );
    }

    const searchParams = new URLSearchParams({
      project_id: projectName,
      repo_url: repoUrl,
    });

    const headers = await buildForwardHeadersAsync(request);
    const response = await fetch(
      `${API_SERVER_URL}/api/ambient/v1/repo_intelligences/lookup?${searchParams}`,
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
    console.error("Failed to fetch repo intelligence:", error);
    return NextResponse.json(
      { error: "Failed to fetch repo intelligence" },
      { status: 500 }
    );
  }
}
