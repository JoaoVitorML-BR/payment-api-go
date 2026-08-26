"use client";
// components/checkout/MercadoPagoBricks.tsx
import { useEffect, useState } from "react";
import { initMercadoPago, Payment } from "@mercadopago/sdk-react";
import { Alert, Group, Paper, Text, ThemeIcon, Title } from "@mantine/core";
import { IconAlertCircle, IconQrcode } from "@tabler/icons-react";

type MercadoPagoBricksProps = {
  publicKey: string;
  amount: number;
  payerEmail: string;
  onSubmit: () => Promise<void>;
};

// Só Pix está habilitado por enquanto. Quando crédito/débito forem implementados,
// isso vira um Brick próprio (CardPayment) — não reabrir os métodos aqui sem
// também tratar o formData do Brick no backend.
const customization = {
  paymentMethods: {
    bankTransfer: "all" as const,
  },
};

export default function MercadoPagoBricks({ publicKey, amount, payerEmail, onSubmit }: MercadoPagoBricksProps) {
  const [error, setError] = useState("");

  useEffect(() => {
    if (!publicKey) {
      setError("Public key não configurada. Defina NEXT_PUBLIC_MERCADO_PAGO_PUBLIC_KEY em .env.local");
      return;
    }
    initMercadoPago(publicKey);
  }, [publicKey]);

  if (error) {
    return (
      <Paper withBorder p="md" radius="md">
        <Alert icon={<IconAlertCircle size={16} />} title="Erro" color="red" variant="light">
          {error}
        </Alert>
      </Paper>
    );
  }

  return (
    <Paper withBorder p="md" radius="md">
      <Group mb="md">
        <ThemeIcon variant="light" color="blue" size="lg" radius="md">
          <IconQrcode size={20} />
        </ThemeIcon>
        <div>
          <Title order={5}>Mercado Pago Pix</Title>
          <Text size="sm" c="dimmed">Pagamento via Pix com QR Code</Text>
        </div>
      </Group>

      <Payment
        initialization={{ amount, payer: { email: payerEmail } }}
        customization={customization}
        onSubmit={async () => {
          // O formData do Brick não é usado hoje: quem cria a cobrança Pix é o
          // nosso próprio backend. O Brick aqui serve como gatilho de submit + UI.
          await onSubmit();
        }}
        onReady={() => setError("")}
        onError={(brickError) => setError(brickError?.message ?? "Erro ao renderizar o Brick de pagamento")}
      />
    </Paper>
  );
}