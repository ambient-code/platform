import { BACKEND_URL } from "@/lib/config";
import { NextRequest, NextResponse } from "next/server";

export async function DELETE(
  request: NextRequest,
  { params }: { params: Promise<{ name: string; itemId: string }> }
) {
  try {
    const { name: projectName, itemId } = await params;
    const sourceUrl = request.nextUrl.searchParams.get("sourceUrl") || "";
    const queryString = sourceUrl ? `?sourceUrl=${encodeURIComponent(sourceUrl)}` : "";

    const response = await fetch(
      `${BACKEND_URL}/projects/${projectName}/marketplace/items/${itemId}${queryString}`,
      {
        method: "DELETE",
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
    console.error("Failed to uninstall item:", error);
    return NextResponse.json(
      { error: "Failed to uninstall item" },
      { status: 500 }
    );
  }
}
