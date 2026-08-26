"use client";
// lib/api.ts
// Camada de I/O: a única parte do frontend que conhece o formato "on the wire" do
// backend (snake_case). Traduz para/de types/domain e não vaza esse detalhe pra cima.
import type {
  ClientSecret,
  CreatePaymentInput,
  CreatePaymentResult,
  PaymentMethod,
  PaymentStatus,
} from "@/types/domain/payment";

const backendUrl = process.env.NEXT_PUBLIC_PAYMENT_API_URL ?? "http://localhost:8080";

// ---- Formato "wire" (o que o backend realmente fala) ----
type CreatePaymentWireRequest = {
  idempotency_key: string;
  merchant_reference?: string;
  amount_cents: number;
  currency: string;
  payment_method: PaymentMethod;
  stripe_payment_method_id?: string;
  installments?: number;
  customer?: {
    name: string;
    email: string;
    phone: string;
    tax_id: string;
    address?: string;
    city?: string;
    state?: string;
    postal_code?: string;
  };
};

type CreatePaymentWireResponse = {
  message: string;
  data: {
    payment_uuid: string;
    payment_method: PaymentMethod;
    status: PaymentStatus;
    created_at: string;
    updated_at: string;
  };
};

type ClientSecretWireResponse = {
  client_secret: string;
  status: PaymentStatus;
};

function toWireRequest(input: CreatePaymentInput): CreatePaymentWireRequest {
  return {
    idempotency_key: input.idempotencyKey,
    merchant_reference: input.merchantReference,
    amount_cents: input.amountCents,
    currency: input.currency,
    payment_method: input.paymentMethod,
    stripe_payment_method_id: input.stripePaymentMethodId,
    installments: input.installments,
    customer: input.customer
      ? {
        name: input.customer.name,
        email: input.customer.email,
        phone: input.customer.phone,
        tax_id: input.customer.taxId,
        address: input.customer.address,
        city: input.customer.city,
        state: input.customer.state,
        postal_code: input.customer.postalCode,
      }
      : undefined,
  };
}

function fromWireCreateResponse(wire: CreatePaymentWireResponse): CreatePaymentResult {
  return {
    message: wire.message,
    payment: {
      paymentUuid: wire.data.payment_uuid,
      paymentMethod: wire.data.payment_method,
      status: wire.data.status,
      createdAt: wire.data.created_at,
      updatedAt: wire.data.updated_at,
    },
  };
}

function fromWireClientSecret(wire: ClientSecretWireResponse): ClientSecret {
  return { clientSecret: wire.client_secret, status: wire.status };
}

export async function createPayment(input: CreatePaymentInput): Promise<CreatePaymentResult> {
  const response = await fetch(`${backendUrl}/payment`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(toWireRequest(input)),
    cache: "no-store",
  });

  if (!response.ok) {
    const errorData = (await response.json().catch(() => null)) as { error?: string } | null;
    throw new Error(errorData?.error ?? `Falha ao criar payment request (${response.status})`);
  }

  return fromWireCreateResponse((await response.json()) as CreatePaymentWireResponse);
}

export async function getClientSecret(paymentUuid: string): Promise<ClientSecret> {
  const response = await fetch(`${backendUrl}/payment/client-secret/${paymentUuid}`, {
    cache: "no-store",
  });

  if (response.status === 404) {
    throw new Error("pending");
  }

  if (!response.ok) {
    const errorData = (await response.json().catch(() => null)) as { error?: string } | null;
    throw new Error(errorData?.error ?? `Falha ao buscar client secret (${response.status})`);
  }

  return fromWireClientSecret((await response.json()) as ClientSecretWireResponse);
}

export async function pollClientSecret(
  paymentUuid: string,
  maxAttempts = 10,
  intervalMs = 1000,
): Promise<ClientSecret | null> {
  for (let attempt = 1; attempt <= maxAttempts; attempt += 1) {
    try {
      const data = await getClientSecret(paymentUuid);
      if (data.clientSecret) return data;
    } catch (error) {
      const message = error instanceof Error ? error.message : "";
      if (message !== "pending") throw error;
    }
    await new Promise((resolve) => setTimeout(resolve, intervalMs));
  }
  return null;
}

export function generateIdempotencyKey(): string {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `key-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}