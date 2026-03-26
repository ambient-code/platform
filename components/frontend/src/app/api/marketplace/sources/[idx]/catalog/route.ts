import { BACKEND_URL } from "@/lib/config";
import { NextRequest, NextResponse } from "next/server";

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ idx: string }> }
) {
  try {
    const { idx } = await params;
    const response = await fetch(
      `${BACKEND_URL}/marketplace/sources/${idx}/catalog`,
      {
        method: "GET",
        headers: {
          "Content-Type": "application/json",
          Authorization: request.headers.get("Authorization") || "",
        },
      }
    );

    const data = await response.text();
    return new NextResponse(data, {
      status: response.status,
      headers: { "Content-Type": "application/json" },
    });
  } catch (error) {
    console.error("Failed to fetch marketplace catalog:", error);
    return NextResponse.json(
      { error: "Failed to fetch marketplace catalog" },
      { status: 500 }
    );
  }
}
