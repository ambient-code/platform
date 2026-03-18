import { NextRequest, NextResponse } from "next/server";
import { BACKEND_URL } from "@/lib/config";
import { buildForwardHeadersAsync } from "@/lib/auth";

export async function GET(request: NextRequest) {
  try {
    const headers = await buildForwardHeadersAsync(request);
    const queryString = request.nextUrl.search;
    const response = await fetch(`${BACKEND_URL}/ldap/users${queryString}`, {
      method: 'GET',
      headers,
    });

    const data = await response.text();

    return new NextResponse(data, {
      status: response.status,
      headers: {
        "Content-Type": "application/json",
      },
    });
  } catch (error) {
    console.error("Failed to search LDAP users:", error);
    return NextResponse.json(
      { error: "LDAP search unavailable" },
      { status: 503 }
    );
  }
}
