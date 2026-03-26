import { BACKEND_URL } from "@/lib/config";
import { NextRequest, NextResponse } from "next/server";

export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name: projectName } = await params;
    const body = await request.text();

    const response = await fetch(
      `${BACKEND_URL}/projects/${projectName}/marketplace/scan`,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-User-ID": request.headers.get("X-User-ID") || "",
          "X-User-Groups": request.headers.get("X-User-Groups") || "",
        },
        body,
      }
    );

    const data = await response.text();
    return new NextResponse(data, {
      status: response.status,
      headers: { "Content-Type": "application/json" },
    });
  } catch (error) {
    console.error("Failed to scan git source:", error);
    return NextResponse.json(
      { error: "Failed to scan git source" },
      { status: 500 }
    );
  }
}
