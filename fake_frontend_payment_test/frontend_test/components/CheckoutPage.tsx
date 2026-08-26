"use client";
// components/CheckoutPage.tsx
import { Container, Text, Title } from "@mantine/core";
import PixCheckoutForm from "./checkout/PixCheckoutForm";

export default function CheckoutPage() {
  return (
    <Container size="md" py="xl">
      <Title order={2} mb="xs">Criar payment request</Title>
      <Text c="dimmed" mb="lg">
        Ambiente de testes de pagamento. Por enquanto só Pix está disponível; crédito e
        débito entram depois como um novo componente em components/checkout.
      </Text>
      <PixCheckoutForm />
    </Container>
  );
}