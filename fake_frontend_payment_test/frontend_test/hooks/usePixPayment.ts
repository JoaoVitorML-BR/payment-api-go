"use client";
// hooks/usePixPayment.ts
// Orquestra o teste de pagamento Pix: cria o payment request e faz o polling do
// client secret. Componentes de UI só leem o estado devolvido, nunca chamam lib/api.ts.
import { useState } from "react";
import { createPayment, generateIdempotencyKey, pollClientSecret } from "@/lib/api";
import type { ClientSecret, CreatePaymentResult, Customer } from "@/types/domain/payment";

export type PixPaymentFormInput = {
    merchantReference: string;
    amountCents: number;
    currency: string;
    customer: Customer;
};

type PixPaymentState = {
    status: "idle" | "submitting" | "polling" | "done" | "error";
    message: string;
    error: string;
    createResult: CreatePaymentResult | null;
    clientSecret: ClientSecret | null;
};

const initialState: PixPaymentState = {
    status: "idle",
    message: "Pronto para criar um pagamento de teste via Pix.",
    error: "",
    createResult: null,
    clientSecret: null,
};

export function usePixPayment() {
    const [state, setState] = useState<PixPaymentState>(initialState);

    async function submit(input: PixPaymentFormInput) {
        setState({ ...initialState, status: "submitting", message: "Enviando para o backend..." });

        try {
            const createResult = await createPayment({
                idempotencyKey: generateIdempotencyKey(),
                merchantReference: input.merchantReference,
                amountCents: input.amountCents,
                currency: input.currency,
                paymentMethod: "pix",
                customer: input.customer,
            });

            setState((current) => ({
                ...current,
                status: "polling",
                createResult,
                message: `Payment request criado com id ${createResult.payment.paymentUuid}. Consultando status processado...`,
            }));

            const clientSecret = await pollClientSecret(createResult.payment.paymentUuid);

            setState((current) => ({
                ...current,
                status: "done",
                clientSecret,
                message: clientSecret
                    ? `Pagamento processado com status final ${clientSecret.status}.`
                    : "Payment request criado, mas o status processado ainda não ficou disponível.",
            }));
        } catch (submitError) {
            const message =
                submitError instanceof Error ? submitError.message : "Erro inesperado ao testar o pagamento.";
            setState((current) => ({ ...current, status: "error", error: message, message: "Falha ao executar o teste." }));
        }
    }

    function reset() {
        setState(initialState);
    }

    return { ...state, submit, reset };
}