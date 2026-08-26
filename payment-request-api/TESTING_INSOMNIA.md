# 🧪 Fluxo de Testes via Insomnia (antes do Frontend)

Este guia descreve como validar **todo o ciclo de pagamento** usando apenas chamadas HTTP via Insomnia (ou Postman), antes de integrar o frontend.

---

## 1️⃣ Preparar o ambiente

| Item | Como fazer |
|------|-------------|
| **Variáveis de ambiente** | Crie um arquivo `.env` (ou exporte no terminal):<br>`MERCADO_PAGO_WEBHOOK_SECRET=seu-segredo-aqui`<br>`MERCADO_PAGO_ACCESS_TOKEN=token-de-acesso` |
| **Banco de dados** | Rode `docker compose up -d` e aguarde o PostgreSQL saudável. |
| **Compilar / rodar a API** | ```bash<br>cd payment-request-api<br>go run ./cmd/main.go<br>```<br>Por padrão escuta em `http://localhost:8080`. |

> Se a porta for diferente, ajuste nas URLs abaixo.

---

## 2️⃣ Rotas disponíveis

| Método | Path | Descrição |
|--------|------|-----------|
| **POST** | `/payment` | Cria um `PaymentRequest` |
| **GET** | `/payment/client-secret/:payment_id` | Recupera client secret / QR code da tentativa mais recente |
| **POST** | `/webhook/mercadopago` | Recebe notificações de status do Mercado Pago (requer `X-Signature`) |
| **POST** | `/payment/refund` | Processa reembolso parcial ou total |

---

## 3️⃣ Payloads de exemplo

### ⚠️ Entendendo o payload (importante!)

O body enviado **deve usar `snake_case`** (minúsculo com underscores), exatamente como os campos JSON definidos na API. Exemplo do que **não** funciona:

```json
{
  "IdempotencyKey": "ecompativel-4abc2345hj",   // ❌ deve ser idempotency_key
  "AmountCents": "15000",                        // ❌ deve ser number, não string
  "Installments": "",                            // ❌ só para cartão, omita no pix
  "Customer": ""                                 // ❌ deve ser objeto, não string
}
```

Campos obrigatórios para **PIX**:

| Campo | Tipo | Obrigatório | Observação |
|-------|------|-------------|------------|
| `idempotency_key` | string | ✅ | Chave de idempotência única |
| `amount_cents` | **number** | ✅ | Valor em centavos (15000 = R$ 150,00) — **não é string** |
| `currency` | string | ✅ | `BRL` |
| `payment_method` | string | ✅ | `pix` |
| `customer.name` | string | ✅ | Nome do pagador |
| `customer.email` | string | ✅ | **Email do cidadão — o Mercado Pago exige** |
| `customer.phone` | string | ⬜ | Recomendado |
| `customer.tax_id` | string | ✅ | CPF/CNPJ do pagador |
| `customer.address` | string | ✅ | Endereço |
| `customer.city` | string | ✅ | Cidade |
| `customer.state` | string | ✅ | UF (ex.: `SP`) |
| `customer.postal_code` | string | ✅ | CEP |

**Por que nossa API precisa dos dados do pagador (email, CPF, endereço)?**

Porque o fluxo de PIX é **server-side**:

1. O **frontend** envia apenas os dados que a API exige (acima);
2. A **nossa API** (`payment-request-api`) salva o `payment_request` e publica um evento no RabbitMQ;
3. O **consumer** (`payment-consumer`) consome o evento e **chama a API do Mercado Pago** com os dados do pagador (`PayerEmail`, `PayerName`, `PayerTaxID`, `PayerAddress`, etc.) para **criar o PIX**;
4. O Mercado Pago retorna o **QR Code**, que é salvo na tabela `payment_attempts`;
5. O **frontend** busca o QR Code via `GET /payment/client-secret/:payment_id` e exibe para o pagador.

Ou seja, **não é o frontend chamando o Bricks de PIX diretamente** — o Bricks do Mercado Pago no frontend é usado para **cartão de crédito**. Para PIX, o QR Code é gerado pelo nosso backend via API do Mercado Pago.

> 📌 **Resumo:** o frontend manda **apenas** os campos que a nossa API requer (snake_case, amount_cents como number). A nossa API **precisa** do email e CPF do cidadão para que o Mercado Pago gere o PIX corretamente.

---

### 3.1 Criar pagamento (POST `/payment`)

```json
{
  "idempotency_key": "uniq-12345",
  "merchant_reference": "pedido-001",
  "amount_cents": 1500,
  "currency": "BRL",
  "payment_method": "pix",
  "customer": {
    "name": "João Silva",
    "email": "joao@email.com",
    "phone": "5511999999999",
    "tax_id": "123.456.789-00",
    "address": "Rua A, 123",
    "city": "São Paulo",
    "state": "SP",
    "postal_code": "01000-000"
  }
}
```

**Resposta esperada (201):**
```json
{
  "message": "payment request accepted",
  "data": {
    "payment_uuid": "c0a1b2d3-4e5f-6789-abcd-ef0123456789",
    "payment_method": "pix",
    "status": "pending",
    "created_at": "2026-08-14T11:45:00Z",
    "updated_at": "2026-08-14T11:45:00Z"
  }
}
```

> **Guarde o `payment_uuid`** — será usado nas próximas chamadas.

---

### 3.2 Obter client secret (GET `/payment/client-secret/:payment_id`)

**URL:** `http://localhost:8080/payment/client-secret/{{payment_uuid}}`

**Resposta esperada (200):**
```json
{
  "status": "pending",
  "client_secret": "some_secret_if_cc",
  "gateway": "mercado_pago",
  "gateway_payment_id": "some_id",
  "pix_qr_code": "https://.../qr.png",
  "pix_qr_code_base64": "iVBORw0KGgoAAA...",
  "pix_expiration_at": "2026-08-14T12:45:00Z"
}
```

> Se ainda não existir tentativa → **404** (já tratado no código).

---

### 3.3 Simular webhook do Mercado Pago (POST `/webhook/mercadopago`)

#### Gerar o header `X-Signature`

O header `X-Signature` é um **HMAC-SHA256** do body usando `MERCADO_PAGO_WEBHOOK_SECRET`:

```go
import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
)

func sign(body []byte, secret string) string {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(body)
    return hex.EncodeToString(mac.Sum(nil))
}
```

**Ferramenta online:** https://www.freeformatter.com/hmac-generator.html
- Algoritmo: **SHA256**
- Chave: valor de `MERCADO_PAGO_WEBHOOK_SECRET`
- Texto: payload JSON exato que será enviado

#### Payload de exemplo

```json
{
  "action": "payment.updated",
  "data": {
    "id": "c0a1b2d3-4e5f-6789-abcd-ef0123456789",
    "status": "succeeded",
    "amount": 1500
  }
}
```

#### Configurar a request

| Config | Valor |
|--------|-------|
| **Método** | `POST` |
| **URL** | `http://localhost:8080/webhook/mercadopago` |
| **Header** | `X-Signature: <hash-gerado>` |
| **Body** | JSON acima (_raw_ / _JSON_) |

**Resposta esperada (200):**
```json
{
  "status": "success"
}
```

> O handler converte o `id` para `paymentID` e chama `UpdatePaymentStatus` com status `"succeeded"`.

---

### 3.4 Verificar atualização (GET client-secret novamente)

Repita a chamada **3.2** → agora o `status` deve ser `"succeeded"`.

---

### 3.5 Reembolso (POST `/payment/refund`)

#### Parcial (sem split)

```json
{
  "payment_id": "c0a1b2d3-4e5f-6789-abcd-ef0123456789",
  "amount_cents": 500,
  "split_rule": ""
}
```

#### Total com split 50/50 (consultor)

```json
{
  "payment_id": "c0a1b2d3-4e5f-6789-abcd-ef0123456789",
  "amount_cents": 1500,
  "split_rule": "50/50"
}
```

**Resposta esperada (200):**
```json
{
  "message": "refund processed"
}
```

> Com `split_rule = "50/50"`, o `amount_cents` é divido por 2 no service (ex.: 1500 → 750 para o consultor).

---

## 4️⃣ Checklist de validação

- [ ] **Criar pagamento** → UUID retornado (`status=pending`)
- [ ] **GET client-secret** → `status = pending`
- [ ] **Webhook** (assinatura válida) → `status = succeeded`
- [ ] **GET client-secret** novamente → `status = succeeded`
- [ ] **Reembolso parcial** → `amount_cents` menor que o total → `status = refunded`
- [ ] **Reembolso total c/ split 50/50** → valor dividido corretamente (verificar logs/DB)

---

## 5️⃣ Dicas de depuração

| Problema | Como investigar |
|----------|------------------|
| **404 ao buscar client secret** | Verifique se o `payment_uuid` existe: `SELECT * FROM payment_requests WHERE uuid = '<uuid>';` |
| **Erro de assinatura no webhook** | Confirme que `MERCADO_PAGO_WEBHOOK_SECRET` está carregada. Compare o hash gerado manualmente com o enviado. |
| **Status não muda** | Adicione `log.Printf` temporário no service. Verifique se o `payment_uuid` passado ao repo está correto. |
| **Valor de reembolso errado** | O `ProcessRefund` divide por 2 apenas quando `split_rule = "50/50"` (case-sensitive). |