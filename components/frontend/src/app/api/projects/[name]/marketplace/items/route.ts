import { BACKEND_URL } from "@/lib/config";
import { NextRequest, NextResponse } from "next/server";

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name: projectName } = await params;

    const response = await fetch(
      `${BACKEND_URL}/projects/${projectName}/marketplace/items`,
      {
        method: "GET",
        headers: {
          "Content-Type": "application/json",
          "X-User-ID": request.headers.get("X-User-ID") || "",
          "X-User-Groups": request.headers.get("X-User-Groups") || "",
        },
      }
    );

    const data = await response.text();
    return new NextResponse(data, {
      status: response.status,
      headers: { "Content-Type": "application/json" },
    });
  } catch (error) {
    console.error("Failed to fetch installed items:", error);
    return NextResponse.json(
      { error: "Failed to fetch installed items" },
      { status: 500 }
    );
  }
}

export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name: projectName } = await params;
    const body = await request.text();

    const response = await fetch(
      `${BACKEND_URL}/projects/${projectName}/marketplace/items`,
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
    console.error("Failed to install items:", error);
    return NextResponse.json(
      { error: "Failed to install items" },
      { status: 500 }
    );
  }
}
