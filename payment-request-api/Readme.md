# 🚀 Payment Request API

API REST em Go (Gin) responsável pela ingestão de pedidos de pagamento, gerenciamento de reembolsos idempotentes, conexão OAuth de vendedores para Split 1:1, processamento seguro de Webhooks do Mercado Pago e Reconciliação em Background.

---

## 🧭 Navegação

- [⬅️ Voltar para o README Principal](../README.md)
- [⚙️ Ir para `payment-consumer`](../payment-consumer/README.md)
- [📚 Ir para `payment-context` (Regras de Negócio)](../payment-context/README.md)
- [🧪 Guia de Testes Manuais](../MANUAL_TESTING.md)

---

## 📌 Endpoints Principais

### 1. Pagamentos

| Método | Rota | Descrição |
|---|---|---|
| `POST` | `/payment` | Cria um novo pedido de pagamento com Idempotency-Key. Publica evento no RabbitMQ. |
| `GET` | `/payment/:payment_id/client-secret` | Obtém o QR Code Pix ou Client Secret do Stripe para o frontend. |
| `GET` | `/payment/:payment_id/status` | Retorna o status local comparado ao status da adquirente (in_sync). |

### 2. Estornos / Reembolsos

| Método | Rota | Descrição |
|---|---|---|
| `POST` | `/payment/refund` | Processa estorno parcial ou total com trava atômica no banco e idempotência. |

### 3. Webhooks & Notificações

| Método | Rota | Descrição |
|---|---|---|
| `POST` | `/payment/webhook/mercadopago` | Recebe notificação com validação HMAC-SHA256 (`X-Signature`), faz re-query oficial no Mercado Pago e atualiza status. |

### 4. OAuth de Vendedores (Marketplace Split)

| Método | Rota | Descrição |
|---|---|---|
| `GET` | `/oauth/mercadopago/start` | Inicia fluxo OAuth gerando State com TTL de 10 minutos. |
| `GET` | `/oauth/mercadopago/callback` | Recebe código de autorização, troca por tokens e armazena com AES-256-GCM. |

---

## 🔒 Segurança e Trust Boundary

- **Validação de Assinatura Webhook**: Verifica o cabeçalho `X-Signature` calculando o HMAC-SHA256 do manifest (`id:dataID;request-id:requestID;ts:ts;`).
- **Zero Trust nos Payloads**: O estado financeiro do pedido nunca é atualizado baseado no body do webhook; a API realiza uma consulta direta (`GET /v1/orders/{id}`) na adquirente.
- **Proteção Anti-Double Refund**: Reserva atômica usando `SELECT ... FOR UPDATE` no banco Postgres antes de contactar a adquirente.

---

## ⚙️ Variáveis de Ambiente

Crie um arquivo `.env` na raiz do módulo `payment-request-api/`:

```env
PORT=8080
DATABASE_URL=postgres://postgres:postgres@localhost:5432/payment_db?sslmode=disable
RABBITMQ_URL=amqp://guest:guest@localhost:5672/

# Mercado Pago
MERCADO_PAGO_ACCESS_TOKEN=TEST-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
MERCADO_PAGO_WEBHOOK_SECRET=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# OAuth Vendedor (Split 1:1)
MERCADO_PAGO_OAUTH_CLIENT_ID=xxxxxxxxxxxx
MERCADO_PAGO_OAUTH_CLIENT_SECRET=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
MERCADO_PAGO_OAUTH_REDIRECT_URI=http://localhost:8080/oauth/mercadopago/callback
MERCADO_PAGO_OAUTH_TOKEN_FILE=./data/seller-tokens.enc
MERCADO_PAGO_OAUTH_ENCRYPTION_KEY=sua-chave-base64-de-32-bytes
```

---

## 🧪 Testes

```bash
cd payment-request-api
go test -v -race ./...
```
