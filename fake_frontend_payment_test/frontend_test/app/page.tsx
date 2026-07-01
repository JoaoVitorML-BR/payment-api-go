"use client";

import { useMemo, useState, type SyntheticEvent } from "react";
import { CardElement, Elements, useElements, useStripe } from "@stripe/react-stripe-js";
import { loadStripe } from "@stripe/stripe-js";

type CreatePaymentResponse = {
  message: string;
  data: {
    payment_uuid: string;
    payment_method: string;
    status: string;
    created_at: string;
    updated_at: string;
  };
};

type ClientSecretResponse = {
  client_secret: string;
  status: string;
};

type FormState = {
  idempotencyKey: string;
  merchantReference: string;
  amountCents: string;
  currency: string;
  paymentMethod: string;
  installments: string;
  cardHolderName: string;
  cardHolderEmail: string;
};

const publishableKey = process.env.NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY ?? "";
const hasValidStripePublishableKey = publishableKey.startsWith("pk_");
const stripePromise = hasValidStripePublishableKey ? loadStripe(publishableKey) : null;
const paymentMethodOptions = ["credit", "debit", "pix", "boleto"];

function generateKey() {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }

  return `key-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

function createInitialFormState(): FormState {
  return {
    idempotencyKey: "",
    merchantReference: "order-987",
    amountCents: "1500",
    currency: "BRL",
    paymentMethod: "credit",
    installments: "8",
    cardHolderName: "Teste Stripe",
    cardHolderEmail: "teste@example.com",
  };
}

function emptyResponseMessage() {
  return "Nenhuma resposta ainda.";
}

function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function fetchClientSecret(paymentId: string) {
  for (let attempt = 1; attempt <= 10; attempt += 1) {
    const response = await fetch(`/api/payment/client-secret/${paymentId}`, {
      cache: "no-store",
    });

    if (response.ok) {
      const data = (await response.json()) as ClientSecretResponse;
      if (data.client_secret) {
        return data;
      }
    }

    await sleep(1000);
  }

  return null;
}

function StripeCheckoutForm() {
  const stripe = useStripe();
  const elements = useElements();

  const [form, setForm] = useState<FormState>(() => createInitialFormState());
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [status, setStatus] = useState("Pronto para criar um pagamento de teste.");
  const [error, setError] = useState("");
  const [createResponse, setCreateResponse] = useState<CreatePaymentResponse | null>(null);
  const [clientSecret, setClientSecret] = useState("");
  const [paymentDetails, setPaymentDetails] = useState<ClientSecretResponse | null>(null);
  const [generatedPaymentMethodId, setGeneratedPaymentMethodId] = useState("");

  const payloadPreview = useMemo(
    () =>
      JSON.stringify(
        {
          idempotency_key: form.idempotencyKey,
          merchant_reference: form.merchantReference,
          amount_cents: Number(form.amountCents),
          currency: form.currency,
          payment_method: form.paymentMethod,
          stripe_payment_method_id: generatedPaymentMethodId || form.idempotencyKey || undefined,
          installments: form.installments ? Number(form.installments) : undefined,
        },
        null,
        2,
      ),
    [form, generatedPaymentMethodId],
  );

  async function handleSubmit(event: SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    setIsSubmitting(true);
    setError("");
    setCreateResponse(null);
    setClientSecret("");
    setPaymentDetails(null);
    setGeneratedPaymentMethodId("");

    try {
      if (!stripe || !elements) {
        throw new Error("Stripe ainda não carregou. Verifique NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY.");
      }

      const cardElement = elements.getElement(CardElement);
      if (!cardElement) {
        throw new Error("CardElement não está disponível.");
      }

      const { error: paymentMethodError, paymentMethod } = await stripe.createPaymentMethod({
        type: "card",
        card: cardElement,
        billing_details: {
          name: form.cardHolderName || undefined,
          email: form.cardHolderEmail || undefined,
        },
      });

      if (paymentMethodError || !paymentMethod) {
        throw new Error(paymentMethodError?.message || "Não foi possível criar o payment method na Stripe.");
      }

      setGeneratedPaymentMethodId(paymentMethod.id);
      setStatus(`Payment method ${paymentMethod.id} criado. Enviando para o backend...`);

      const idempotencyKey = form.idempotencyKey || generateKey();
      const response = await fetch("/api/payment", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          idempotency_key: idempotencyKey,
          merchant_reference: form.merchantReference,
          amount_cents: Number(form.amountCents),
          currency: form.currency,
          payment_method: form.paymentMethod,
          stripe_payment_method_id: paymentMethod.id,
          installments: form.installments ? Number(form.installments) : undefined,
        }),
      });

      const result = (await response.json()) as CreatePaymentResponse | { error?: string };

      if (!response.ok) {
        throw new Error("error" in result && result.error ? result.error : "Falha ao criar payment request.");
      }

      const typedResult = result as CreatePaymentResponse;
      setCreateResponse(typedResult);
      setStatus(`Payment request criado com id ${typedResult.data.payment_uuid}. Consultando status processado...`);

      const paymentDetails = await fetchClientSecret(typedResult.data.payment_uuid);
      if (paymentDetails) {
        setClientSecret(paymentDetails.client_secret);
        setPaymentDetails(paymentDetails);
        setStatus(`Pagamento processado com status final ${paymentDetails.status}.`);
      } else {
        setStatus("Payment request criado, mas o status processado ainda não ficou disponível.");
      }

      setForm((current) => ({
        ...current,
        idempotencyKey: generateKey(),
      }));
    } catch (submitError) {
      const message = submitError instanceof Error ? submitError.message : "Erro inesperado ao testar o pagamento.";
      setError(message);
      setStatus("Falha ao executar o teste.");
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <main className="min-h-screen px-6 py-8 text-slate-100 sm:px-10 lg:px-12">
      <div className="mx-auto flex min-h-[calc(100vh-4rem)] w-full max-w-7xl flex-col gap-8">
        <section className="overflow-hidden rounded-4xl border border-white/10 bg-white/5 shadow-[0_30px_100px_rgba(0,0,0,0.35)] backdrop-blur-xl">
          <div className="grid gap-0 lg:grid-cols-[1.1fr_0.9fr]">
            <div className="relative p-6 sm:p-8 lg:p-10">
              <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(99,102,241,0.22),transparent_35%),radial-gradient(circle_at_bottom_right,rgba(16,185,129,0.14),transparent_35%)]" />
              <div className="relative z-10 flex flex-col gap-6">
                <div className="flex flex-wrap items-center gap-3 text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">
                  <span className="rounded-full border border-emerald-400/30 bg-emerald-400/10 px-3 py-1 text-emerald-300">
                    Stripe Elements test
                  </span>
                  <span>Card entered in the browser, backend confirms the PaymentIntent</span>
                </div>

                <div className="space-y-4">
                  <h1 className="max-w-2xl text-4xl font-semibold tracking-tight text-white sm:text-5xl">
                    Teste o fluxo real sem mandar dados do cartão para o backend.
                  </h1>
                  <p className="max-w-2xl text-base leading-7 text-slate-300 sm:text-lg">
                    A Stripe coleta o cartão no frontend, cria um payment method no browser e seu backend só recebe o
                    ID para confirmar a cobrança com a resposta da Stripe.
                  </p>
                </div>

                <div className="grid gap-3 sm:grid-cols-3">
                  {[
                    ["1", "Cartão digitado no Stripe Elements"],
                    ["2", "Payment method criado no browser"],
                    ["3", "Backend confirma com a Stripe"],
                  ].map(([step, label]) => (
                    <div key={step} className="rounded-2xl border border-white/10 bg-slate-950/40 p-4">
                      <div className="text-sm font-semibold text-indigo-300">Step {step}</div>
                      <div className="mt-2 text-sm leading-6 text-slate-200">{label}</div>
                    </div>
                  ))}
                </div>
              </div>
            </div>

            <aside className="border-t border-white/10 bg-slate-950/50 p-6 sm:p-8 lg:border-l lg:border-t-0 lg:p-10">
              <div className="space-y-3">
                <p className="text-sm font-medium text-slate-400">Status atual</p>
                <p className="text-lg font-medium text-white">{status}</p>
                {error ? <p className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">{error}</p> : null}
              </div>

              <div className="mt-6 space-y-4 rounded-3xl border border-white/10 bg-white/5 p-4">
                <div className="flex items-center justify-between gap-3">
                  <div>
                    <p className="text-sm text-slate-400">Payload enviado</p>
                    <p className="text-xs text-slate-500">O frontend gera o payment method e o Next faz proxy para o backend.</p>
                  </div>
                  <button
                    type="button"
                    onClick={() => setForm((current) => ({ ...current, idempotencyKey: generateKey() }))}
                    className="rounded-full border border-white/10 px-4 py-2 text-sm text-slate-200 transition hover:border-white/20 hover:bg-white/5"
                  >
                    Gerar nova chave
                  </button>
                </div>
                <pre className="max-h-72 overflow-auto rounded-2xl bg-slate-950/90 p-4 text-xs leading-6 text-slate-300">
                  {payloadPreview}
                </pre>
              </div>
            </aside>
          </div>
        </section>

        <section className="grid gap-8 lg:grid-cols-[1.05fr_0.95fr]">
          <form onSubmit={handleSubmit} className="rounded-[28px] border border-white/10 bg-white/5 p-6 shadow-xl backdrop-blur-xl sm:p-8">
            <div className="mb-6 flex items-center justify-between gap-4">
              <div>
                <h2 className="text-2xl font-semibold text-white">Criar payment request</h2>
                <p className="mt-1 text-sm text-slate-400">Preencha os dados da cobrança e o cartão de teste.</p>
              </div>
              <div className="rounded-full border border-sky-400/20 bg-sky-400/10 px-3 py-1 text-xs font-medium text-sky-200">
                API mode
              </div>
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              <label className="space-y-2 sm:col-span-2">
                <span className="text-sm font-medium text-slate-300">Idempotency key</span>
                <input
                  value={form.idempotencyKey}
                  onChange={(event) => setForm((current) => ({ ...current, idempotencyKey: event.target.value }))}
                  className="w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-sm text-white outline-none transition placeholder:text-slate-600 focus:border-indigo-400/60"
                  placeholder="Clique em Gerar nova chave ou deixe vazio e envie"
                />
              </label>

              <label className="space-y-2 sm:col-span-2">
                <span className="text-sm font-medium text-slate-300">Merchant reference</span>
                <input
                  value={form.merchantReference}
                  onChange={(event) => setForm((current) => ({ ...current, merchantReference: event.target.value }))}
                  className="w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-sm text-white outline-none transition placeholder:text-slate-600 focus:border-indigo-400/60"
                  placeholder="order-987"
                />
              </label>

              <label className="space-y-2">
                <span className="text-sm font-medium text-slate-300">Amount in cents</span>
                <input
                  type="number"
                  min={1}
                  value={form.amountCents}
                  onChange={(event) => setForm((current) => ({ ...current, amountCents: event.target.value }))}
                  className="w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-sm text-white outline-none transition placeholder:text-slate-600 focus:border-indigo-400/60"
                />
              </label>

              <label className="space-y-2">
                <span className="text-sm font-medium text-slate-300">Currency</span>
                <input
                  value={form.currency}
                  onChange={(event) => setForm((current) => ({ ...current, currency: event.target.value.toUpperCase() }))}
                  className="w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-sm text-white outline-none transition placeholder:text-slate-600 focus:border-indigo-400/60"
                  placeholder="BRL"
                />
              </label>

              <label className="space-y-2">
                <span className="text-sm font-medium text-slate-300">Payment method</span>
                <select
                  value={form.paymentMethod}
                  onChange={(event) => setForm((current) => ({ ...current, paymentMethod: event.target.value }))}
                  className="w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-sm text-white outline-none transition focus:border-indigo-400/60"
                >
                  {paymentMethodOptions.map((option) => (
                    <option key={option} value={option} className="bg-slate-950">
                      {option}
                    </option>
                  ))}
                </select>
              </label>

              <label className="space-y-2">
                <span className="text-sm font-medium text-slate-300">Installments</span>
                <input
                  type="number"
                  min={1}
                  max={12}
                  value={form.installments}
                  onChange={(event) => setForm((current) => ({ ...current, installments: event.target.value }))}
                  className="w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-sm text-white outline-none transition placeholder:text-slate-600 focus:border-indigo-400/60"
                />
              </label>

              <label className="space-y-2 sm:col-span-2">
                <span className="text-sm font-medium text-slate-300">Cardholder name</span>
                <input
                  value={form.cardHolderName}
                  onChange={(event) => setForm((current) => ({ ...current, cardHolderName: event.target.value }))}
                  className="w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-sm text-white outline-none transition placeholder:text-slate-600 focus:border-indigo-400/60"
                  placeholder="Teste Stripe"
                />
              </label>

              <label className="space-y-2 sm:col-span-2">
                <span className="text-sm font-medium text-slate-300">Cardholder email</span>
                <input
                  type="email"
                  value={form.cardHolderEmail}
                  onChange={(event) => setForm((current) => ({ ...current, cardHolderEmail: event.target.value }))}
                  className="w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-sm text-white outline-none transition placeholder:text-slate-600 focus:border-indigo-400/60"
                  placeholder="teste@example.com"
                />
              </label>

              <div className="space-y-2 sm:col-span-2">
                <span className="text-sm font-medium text-slate-300">Stripe card input</span>
                <div className="rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-4">
                  <CardElement
                    options={{
                      hidePostalCode: true,
                      iconStyle: "solid",
                      style: {
                        base: {
                          color: "#e2e8f0",
                          fontSize: "16px",
                          fontFamily: "var(--font-geist-sans), Arial, sans-serif",
                          "::placeholder": {
                            color: "#64748b",
                          },
                        },
                        invalid: {
                          color: "#fda4af",
                        },
                      },
                    }}
                  />
                </div>
                <p className="text-xs leading-5 text-slate-500">
                  Use o cartão de teste <span className="text-slate-300">4242 4242 4242 4242</span>, validade futura e CVC qualquer.
                </p>
              </div>
            </div>

            <div className="mt-6 flex flex-col gap-3 sm:flex-row">
              <button
                type="submit"
                disabled={isSubmitting || !stripe || !elements || !hasValidStripePublishableKey}
                className="inline-flex items-center justify-center rounded-2xl bg-linear-to-r from-indigo-500 via-violet-500 to-cyan-500 px-5 py-3 text-sm font-semibold text-white shadow-lg shadow-indigo-500/20 transition hover:opacity-95 disabled:cursor-not-allowed disabled:opacity-60"
              >
                {isSubmitting ? "Processando..." : "Testar pagamento"}
              </button>
              <button
                type="button"
                onClick={() => setForm(createInitialFormState())}
                className="inline-flex items-center justify-center rounded-2xl border border-white/10 px-5 py-3 text-sm font-medium text-slate-200 transition hover:border-white/20 hover:bg-white/5"
              >
                Resetar formulário
              </button>
            </div>
          </form>

          <div className="space-y-6">
            <section className="rounded-[28px] border border-white/10 bg-white/5 p-6 shadow-xl backdrop-blur-xl sm:p-8">
              <div className="flex items-center justify-between gap-4">
                <div>
                  <h2 className="text-2xl font-semibold text-white">Resposta do backend</h2>
                  <p className="mt-1 text-sm text-slate-400">A resposta inicial do POST e o status final processado aparecem aqui.</p>
                </div>
                <div className="flex flex-wrap items-center gap-2">
                  {createResponse ? (
                    <span className="rounded-full border border-sky-400/20 bg-sky-400/10 px-3 py-1 text-xs font-medium text-sky-200">
                      pedido aceito
                    </span>
                  ) : null}
                  {paymentDetails ? (
                    <span className="rounded-full border border-emerald-400/20 bg-emerald-400/10 px-3 py-1 text-xs font-medium text-emerald-200">
                      pagamento {paymentDetails.status}
                    </span>
                  ) : null}
                </div>
              </div>

              <div className="mt-5 grid gap-4 sm:grid-cols-3">
                <div className="rounded-2xl border border-white/10 bg-slate-950/60 p-4">
                  <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Payment UUID</p>
                  <p className="mt-2 break-all text-sm font-medium text-white">
                    {createResponse?.data.payment_uuid ?? "Aguardando envio"}
                  </p>
                </div>
                <div className="rounded-2xl border border-white/10 bg-slate-950/60 p-4">
                  <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Client secret</p>
                  <p className="mt-2 break-all text-sm font-medium text-white">
                    {clientSecret || "Será exibido após o processamento pelo backend"}
                  </p>
                </div>
                <div className="rounded-2xl border border-white/10 bg-slate-950/60 p-4">
                  <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Stripe status</p>
                  <p className="mt-2 break-all text-sm font-medium text-white">
                    {paymentDetails?.status ?? createResponse?.data.status ?? "Aguardando envio"}
                  </p>
                </div>
              </div>

              <div className="mt-4 rounded-2xl border border-white/10 bg-slate-950/60 p-4">
                <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Resposta inicial do POST</p>
                <pre className="mt-3 overflow-auto text-xs leading-6 text-slate-300">
                  {createResponse ? JSON.stringify(createResponse, null, 2) : emptyResponseMessage()}
                </pre>
              </div>

              <div className="mt-4 rounded-2xl border border-white/10 bg-slate-950/60 p-4">
                <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Status final processado</p>
                <pre className="mt-3 overflow-auto text-xs leading-6 text-slate-300">
                  {paymentDetails ? JSON.stringify(paymentDetails, null, 2) : "Aguardando a confirmação da rota /payment/client-secret/[paymentId]."}
                </pre>
              </div>
            </section>

            <section className="rounded-3xl border border-white/10 bg-linear-to-br from-slate-950/80 to-slate-900/60 p-6 shadow-xl backdrop-blur-xl sm:p-8">
              <h2 className="text-2xl font-semibold text-white">Como isso funciona</h2>
              <ol className="mt-4 space-y-3 text-sm leading-6 text-slate-300">
                <li className="rounded-2xl border border-white/10 bg-white/5 p-4">1. Você digita o cartão dentro do Stripe Elements.</li>
                <li className="rounded-2xl border border-white/10 bg-white/5 p-4">2. O browser cria um payment method na Stripe e devolve apenas o ID.</li>
                <li className="rounded-2xl border border-white/10 bg-white/5 p-4">3. O frontend envia esse ID para o backend, que confirma a cobrança com a Stripe.</li>
              </ol>
            </section>
          </div>
        </section>
      </div>
    </main>
  );
}

export default function Home() {
  if (!hasValidStripePublishableKey || !stripePromise) {
    return (
      <main className="min-h-screen px-6 py-8 text-slate-100 sm:px-10 lg:px-12">
        <section className="mx-auto max-w-3xl rounded-[28px] border border-white/10 bg-white/5 p-8 shadow-xl backdrop-blur-xl">
          <h1 className="text-3xl font-semibold text-white">Payment test harness</h1>
          <p className="mt-4 text-slate-300">
            Defina <span className="font-semibold text-white">NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY</span> em <span className="font-semibold text-white">.env.local</span> com uma chave que comece com <span className="font-semibold text-white">pk_</span> para carregar o Stripe Elements.
          </p>
          <p className="mt-3 text-sm text-slate-400">
            Exemplo: <span className="font-mono text-slate-200">NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY=pk_test_...</span>
          </p>
        </section>
      </main>
    );
  }

  return (
    <Elements stripe={stripePromise}>
      <StripeCheckoutForm />
    </Elements>
  );
}
