"use client";
// fake_frontend_payment_test\frontend_test\components\StripeCheckoutForm.tsx
import { FormEvent } from "react";
import { CardElement, useElements, useStripe } from "@stripe/react-stripe-js";
import { Button, Paper, Text } from "@mantine/core";

type StripeCheckoutFormProps = {
  isSubmitting: boolean;
  onSubmit: () => void;
};

export default function StripeCheckoutForm({ isSubmitting, onSubmit }: StripeCheckoutFormProps) {
  const stripe = useStripe();
  const elements = useElements();

  const handleLocalSubmit = async (event: FormEvent) => {
    event.preventDefault();
    if (!stripe || !elements) return;
    onSubmit();
  };

  return (
    <form onSubmit={handleLocalSubmit}>
      <Paper withBorder p="md" radius="md">
        <Text size="sm" fw={500} mb="sm">
          Cartão de crédito
        </Text>
        <CardElement
          options={{
            hidePostalCode: true,
            iconStyle: "solid",
            style: {
              base: {
                fontSize: "16px",
                fontFamily: "var(--font-geist-sans), Arial, sans-serif",
              },
            },
          }}
        />
        <Text size="xs" c="dimmed" mt="xs">
          Use o cartão de teste <strong>4242 4242 4242 4242</strong>, validade futura e CVC qualquer.
        </Text>

        <Button
          type="submit"
          fullWidth
          mt="md"
          loading={isSubmitting}
          disabled={isSubmitting || !stripe || !elements}
        >
          Confirmar pagamento
        </Button>
      </Paper>
    </form>
  );
}