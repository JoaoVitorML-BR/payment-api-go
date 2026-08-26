// fake_frontend_payment_test\frontend_test\app\page.tsx
import { Code, Container, Stack, Text, Title } from "@mantine/core";

export default function HomePage() {
  return (
    <Container size="sm" py="xl">
      <Stack gap="md">
        <Title order={1}>Pagamentos Teste - Mercado Pago / Stripe</Title>

        <Text c="dimmed">
          Frontend de testes: Next.js App Router + TypeScript + Mantine v9. As rotas são organizadas
          por pastas em <Code>app/</Code>.
        </Text>

        <Text size="sm" c="dimmed" mt="md">
          Acesse <Code>/mercado-pago</Code> para testar o fluxo de checkout com Pix e cartão.
        </Text>
      </Stack>
    </Container>
  );
}