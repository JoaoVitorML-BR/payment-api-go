import { NextResponse } from "next/server";

const backendBaseUrl = process.env.PAYMENT_API_BASE_URL ?? "http://localhost:8080";

type RouteContext = {
  params: Promise<{
    paymentId: string;
  }>;
};

export async function GET(_: Request, context: RouteContext) {
  try {
    const { paymentId } = await context.params;
    const upstreamResponse = await fetch(`${backendBaseUrl}/payment/client-secret/${paymentId}`, {
      cache: "no-store",
    });

    const data = await upstreamResponse.json();
    return NextResponse.json(data, { status: upstreamResponse.status });
  } catch (error) {
    const message = error instanceof Error ? error.message : "Unexpected proxy error";
    return NextResponse.json({ error: message }, { status: 500 });
  }
}