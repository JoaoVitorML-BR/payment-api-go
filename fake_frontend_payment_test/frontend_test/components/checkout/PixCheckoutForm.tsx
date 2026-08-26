"use client";
// components/checkout/PixCheckoutForm.tsx
import { useMemo, useState } from "react";
import { Divider, Paper, SimpleGrid, Text, TextInput, Title } from "@mantine/core";
import MercadoPagoBricks from "./MercadoPagoBricks";
import { usePixPayment } from "@/hooks/usePixPayment";
import type { Customer } from "@/types/domain/payment";

function createInitialCustomer(): Customer {
  return {
    name: "Joao Teste",
    email: "teste@example.com",
    phone: "11999999999",
    taxId: "12345678909",
    address: "Rua Exemplo, 123",
    city: "São Paulo",
    state: "SP",
    postalCode: "01000000",
  };
}

const mpPublicKey = process.env.NEXT_PUBLIC_MERCADO_PAGO_PUBLIC_KEY ?? "";

export default function PixCheckoutForm() {
  const [merchantReference, setMerchantReference] = useState("order-987");
  const [amountCents, setAmountCents] = useState("1500");
  const [currency, setCurrency] = useState("BRL");
  const [customer, setCustomer] = useState<Customer>(() => createInitialCustomer());

  const { message, error, createResult, clientSecret, submit } = usePixPayment();

  const updateCustomer = (field: keyof Customer) => (value: string) =>
    setCustomer((current) => ({ ...current, [field]: value }));

  const amountPreview = useMemo(() => Number(amountCents) / 100, [amountCents]);

  return (
    <Paper withBorder p="lg" radius="md">
      <Title order={3} mb="xs">Testar pagamento via Pix</Title>
      <Text c="dimmed" mb="lg" size="sm">
        Fluxo dedicado ao Pix usando o Brick de pagamento do Mercado Pago.
      </Text>

      <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="md">
        <TextInput
          label="Merchant reference"
          value={merchantReference}
          onChange={(event) => setMerchantReference(event.currentTarget.value)}
        />
        <TextInput
          label="Amount (centavos)"
          type="number"
          min={1}
          value={amountCents}
          onChange={(event) => setAmountCents(event.currentTarget.value)}
        />
        <TextInput
          label="Currency"
          value={currency}
          onChange={(event) => setCurrency(event.currentTarget.value.toUpperCase())}
        />
      </SimpleGrid>

      <Divider my="md" label="Dados do pagador" labelPosition="center" />
      <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="md">
        <TextInput label="Nome" value={customer.name} onChange={(e) => updateCustomer("name")(e.currentTarget.value)} />
        <TextInput label="Email" type="email" value={customer.email} onChange={(e) => updateCustomer("email")(e.currentTarget.value)} />
        <TextInput label="Telefone" value={customer.phone} onChange={(e) => updateCustomer("phone")(e.currentTarget.value)} />
        <TextInput label="CPF/CNPJ" value={customer.taxId} onChange={(e) => updateCustomer("taxId")(e.currentTarget.value)} />
        <TextInput label="Endereço" value={customer.address} onChange={(e) => updateCustomer("address")(e.currentTarget.value)} />
        <TextInput label="Cidade" value={customer.city} onChange={(e) => updateCustomer("city")(e.currentTarget.value)} />
        <TextInput label="Estado" value={customer.state} onChange={(e) => updateCustomer("state")(e.currentTarget.value)} />
        <TextInput label="CEP" value={customer.postalCode} onChange={(e) => updateCustomer("postalCode")(e.currentTarget.value)} />
      </SimpleGrid>

      <Divider my="md" label="Pagamento" labelPosition="center" />

      <MercadoPagoBricks
        publicKey={mpPublicKey}
        amount={amountPreview}
        payerEmail={customer.email}
        onSubmit={() =>
          submit({
            merchantReference,
            amountCents: Number(amountCents),
            currency,
            customer,
          })
        }
      />

      {createResult && (
        <Paper withBorder p="md" radius="md" mt="lg">
          <Text fw={600} size="sm" mb="xs">Resposta do backend</Text>
          <pre style={{ fontSize: 12, whiteSpace: "pre-wrap" }}>{JSON.stringify(createResult, null, 2)}</pre>
        </Paper>
      )}

      <Paper withBorder p="md" radius="md" mt="md">
        <Text fw={600} size="sm" mb="xs">Status atual</Text>
        <Text size="sm">{message}</Text>
        {error && <Text size="sm" c="red" mt="xs">{error}</Text>}
      </Paper>

      {clientSecret && (
        <Text size="xs" c="dimmed" mt="sm">
          client_secret: {clientSecret.clientSecret} · status: {clientSecret.status}
        </Text>
      )}
    </Paper>
  );
}