// types/domain/payment.ts
// Contrato de domínio para pagamentos. Único lugar que define a "forma" dos dados
// trocados com o backend. Componentes de UI importam tipos daqui, nunca de lib/api.ts.

export type PaymentMethod = "pix" | "credit" | "debit" | "boleto";

export type PaymentStatus =
  | "pending"
  | "processing"
  | "approved"
  | "rejected"
  | "cancelled"
  | (string & {});

export type Customer = {
  name: string;
  email: string;
  phone: string;
  taxId: string;
  address?: string;
  city?: string;
  state?: string;
  postalCode?: string;
};

export type CreatePaymentInput = {
  idempotencyKey: string;
  merchantReference?: string;
  amountCents: number;
  currency: string;
  paymentMethod: PaymentMethod;
  installments?: number;
  stripePaymentMethodId?: string;
  customer?: Customer;
};

export type PaymentRecord = {
  paymentUuid: string;
  paymentMethod: PaymentMethod;
  status: PaymentStatus;
  createdAt: string;
  updatedAt: string;
};

export type CreatePaymentResult = {
  message: string;
  payment: PaymentRecord;
};

export type ClientSecret = {
  clientSecret: string;
  status: PaymentStatus;
};