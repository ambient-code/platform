import { BACKEND_URL } from "@/lib/config";
import { NextRequest, NextResponse } from "next/server";

export async function GET(request: NextRequest) {
  try {
    const response = await fetch(`${BACKEND_URL}/marketplace/sources`, {
      method: "GET",
      headers: {
        "Content-Type": "application/json",
        Authorization: request.headers.get("Authorization") || "",
      },
    });

    const data = await response.text();
    return new NextResponse(data, {
      status: response.status,
      headers: { "Content-Type": "application/json" },
    });
  } catch (error) {
    console.error("Failed to fetch marketplace sources:", error);
    return NextResponse.json(
      { error: "Failed to fetch marketplace sources" },
      { status: 500 }
    );
  }
}
