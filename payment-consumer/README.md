# ⚙️ Payment Consumer Service

Microsserviço em Go responsável pelo processamento assíncrono e resiliente de pagamentos consumidos a partir do RabbitMQ.

---

## 🧭 Navegação

- [⬅️ Voltar para o README Principal](../README.md)
- [🚀 Ir para `payment-request-api`](../payment-request-api/Readme.md)
- [📚 Ir para `payment-context` (Regras de Negócio)](../payment-context/README.md)
- [🧪 Guia de Testes Manuais](../MANUAL_TESTING.md)

---

## 🎯 Responsabilidades

1. **Consumo de Eventos de Pagamento**: Ouve a fila RabbitMQ no tópico de novos pagamentos solicitados (`payment.requested`).
2. **Despacho para Adquirentes**:
   - **Mercado Pago (Orders API)**: Gera QR Code Pix (payload e Base64) configurado com itens, pagador e taxa de marketplace (Split).
   - **Stripe**: Integração de cartão de crédito.
3. **Persistência de Tentativas (`payment_attempts`)**:
   - Registra cada tentativa com número sequencial (`attempt_number`), erros retornados e resposta bruta do gateway para auditoria.
4. **Política de Retentativas**:
   - Falhas transitórias da adquirente disparam retentativa exponencial controlada (até 3 tentativas).
   - Falhas definitivas (ex: dados fiscais inválidos) encerram a operação e marcam o pedido como `failed`.

---

## 🏗️ Arquitetura do Worker

```mermaid
graph TD
    Queue[(RabbitMQ: payment.requested)] --> Processor[PaymentRequestedProcessor]
    Processor --> CheckAttempt{Verifica tentativas prévias}
    CheckAttempt -- Já processado com sucesso --> Ack[Descarta / Ack]
    CheckAttempt -- Limite de retentativas excedido --> MarkFailed[Marca status='failed' no DB]
    CheckAttempt -- Processar --> CallGateway[Chama Mercado Pago Orders API]
    
    CallGateway --> GatewayResp{Resposta do Gateway}
    GatewayResp -- Sucesso --> SaveAttemptSuccess[Salva tentativa no DB + QR Code Pix]
    GatewayResp -- Erro Transitório --> Retry[Registra tentativa e aguarda retentativa]
    GatewayResp -- Erro Permanente --> MarkFailed
```

---

## ⚙️ Variáveis de Ambiente

Crie um arquivo `.env` na raiz do módulo `payment-consumer/`:

```env
DATABASE_URL=postgres://postgres:postgres@localhost:5432/payment_db?sslmode=disable
RABBITMQ_URL=amqp://guest:guest@localhost:5672/

# Mercado Pago
MERCADO_PAGO_ACCESS_TOKEN=TEST-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
MERCADO_PAGO_WEBHOOK_URL=https://seu-dominio-publico.com/payment/webhook/mercadopago
PIX_EXPIRATION_TIME=30

# OAuth Split Storage (deve apontar para o mesmo volume/arquivo do payment-request-api)
MERCADO_PAGO_OAUTH_TOKEN_FILE=../payment-request-api/data/seller-tokens.enc
MERCADO_PAGO_OAUTH_ENCRYPTION_KEY=sua-chave-base64-de-32-bytes
```

---

## 🚀 Execução

```bash
cd payment-consumer
go run cmd/main.go
```

## 🧪 Testes

```bash
cd payment-consumer
go test -v -race ./...
```
