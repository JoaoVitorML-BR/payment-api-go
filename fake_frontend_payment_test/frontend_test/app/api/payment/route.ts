import { NextResponse } from "next/server";

const backendBaseUrl = process.env.PAYMENT_API_BASE_URL ?? "http://localhost:8080";

export async function POST(request: Request) {
  try {
    const payload = await request.json();
    const upstreamResponse = await fetch(`${backendBaseUrl}/payment`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(payload),
      cache: "no-store",
    });

    const data = await upstreamResponse.json();
    return NextResponse.json(data, { status: upstreamResponse.status });
  } catch (error) {
    const message = error instanceof Error ? error.message : "Unexpected proxy error";
    return NextResponse.json({ error: message }, { status: 500 });
  }
}