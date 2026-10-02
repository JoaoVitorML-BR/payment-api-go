# 💳 Payment Gateway Microservices (Go)

Sistema de microsserviços de alto desempenho e tolerância a falhas para processamento assíncrono de pagamentos, estornos e divisão de receitas (Split 1:1) com **Mercado Pago (Orders API & Pix)** e **Stripe**, construído em **Go**, **PostgreSQL** e **RabbitMQ**.

---

## 🧭 Índice de Navegação

- [🗺️ Visão Geral da Arquitetura](#-visão-geral-da-arquitetura)
- [📂 Estrutura do Repositório](#-estrutura-do-repositório)
  - [🚀 `payment-request-api`](#1-payment-request-api) &rarr; [`Ver documentação`](./payment-request-api/Readme.md)
  - [⚙️ `payment-consumer`](#2-payment-consumer) &rarr; [`Ver documentação`](./payment-consumer/README.md)
  - [📚 `payment-context`](#3-payment-context) &rarr; [`Ver documentação`](./payment-context/README.md)
  - [🧪 `MANUAL_TESTING.md`](#4-testes-manuais) &rarr; [`Ver guia de testes`](./MANUAL_TESTING.md)
- [📊 Workflows & Fluxogramas](#-workflows--fluxogramas)
  - [1. Fluxo de Criação de Pagamento](#1-fluxo-de-criação-de-pagamento-pix--orders-api)
  - [2. Fluxo de Webhook & Trust Boundary](#2-fluxo-de-notificação-webhook--segurança-hmac-sha256)
  - [3. Fluxo de Reembolso & Prevenção de Concorrência](#3-fluxo-de-estorno--reembolso-parcialtotal)
  - [4. Fluxo OAuth de Vendedores & Split](#4-fluxo-oauth-do-vendedor--armazenamento-cifrado-aes-256-gcm)
  - [5. Worker de Reconciliação em Background](#5-worker-de-reconciliação-periódica-background)
- [🔒 Segurança & Conformidade Mercado Pago](#-segurança--conformidade-mercado-pago)
- [⚙️ Variáveis de Ambiente](#-variáveis-de-ambiente)
- [🚀 Como Executar](#-como-executar)
- [🧪 Execução de Testes](#-execução-de-testes)

---

## 🗺️ Visão Geral da Arquitetura

O sistema adota o padrão de arquitetura orientada a eventos (EDA) desacoplando a recepção do pedido de pagamento da execução síncrona junto às adquirentes:

```mermaid
flowchart TD
    Client(["Cliente / Frontend"]) -->|"1. POST /payment"| API["payment-request-api"]
    API -->|"2. Salva localmente (pending)"| DB[("PostgreSQL")]
    API -->|"3. Publica payment.requested"| RMQ[("RabbitMQ")]
    API -->|"4. Retorna UUID imediatamente"| Client
    
    RMQ -->|"5. Consome mensagem"| Worker["payment-consumer"]
    Worker -->|"6. Chama Orders API / Pix"| MP["Mercado Pago / Stripe"]
    MP -->|"7. QR Code / Gateway ID"| Worker
    Worker -->|"8. Atualiza attempt e status"| DB
    
    MP -->|"9. Webhook Notificação"| API
    API -->|"10. Valida HMAC e Consulta MP"| MP
    API -->|"11. Atualiza status (succeeded)"| DB
```

---

## 📂 Estrutura do Repositório

| Módulo | Descrição | Link Direto |
|---|---|---|
| **`payment-request-api`** | API REST (Gin) responsável pela ingestão de pedidos, autenticação OAuth de vendedores, estornos idempotentes, recepção segura de webhooks e reconciliador em background. | [📂 `payment-request-api/`](./payment-request-api/Readme.md) |
| **`payment-consumer`** | Worker assíncrono que consome eventos do RabbitMQ, orquestra chamadas resilientes às gateways (com retentativas) e persiste chaves PIX/QR Codes. | [📂 `payment-consumer/`](./payment-consumer/README.md) |
| **`payment-context`** | Fonte de verdade e histórico de decisões arquiteturais, regras financeiras imutáveis e auditorias. | [📂 `payment-context/`](./payment-context/README.md) |
| **`scripts`** | Scripts utilitários de migração do banco de dados para desenvolvimento e CI. | [📂 `scripts/`](./payment-request-api/scripts/README.md) |
| **`fake_frontend_payment_test`** | Frontend de testes (Next.js / Mantine) para simulação de checkout e integração de Bricks. | [📂 `fake_frontend/`](./fake_frontend_payment_test/frontend_test/README.md) |

---

## 📊 Workflows & Fluxogramas

### 1. Fluxo de Criação de Pagamento (Pix / Orders API)

Garante baixa latência ao cliente retornando imediatamente enquanto o processador gera o QR Code Pix com retentativa exponencial.

```mermaid
sequenceDiagram
    autonumber
    actor User as Cliente / App
    participant API as payment-request-api
    participant DB as PostgreSQL
    participant RMQ as RabbitMQ
    participant Worker as payment-consumer
    participant MP as Mercado Pago

    User->>API: POST /payment (amount, currency, customer, method)
    API->>DB: INSERT payment_requests (status='pending')
    API->>RMQ: Publica evento payment.requested.v1
    API-->>User: 202 Accepted (payment_id UUID)

    RMQ->>Worker: Consome evento payment.requested.v1
    Worker->>MP: POST /v1/orders ou /v1/payments (Pix/Card)
    MP-->>Worker: Resposta (gateway_payment_id, qr_code, emv)
    Worker->>DB: UPDATE payment_requests (gateway_id, attempts, status)
    
    User->>API: GET /payment/:payment_id/status (Polling)
    API->>DB: SELECT status, qr_code FROM payment_requests
    API-->>User: 200 OK (status, qr_code_base64)
```

### 2. Fluxo de Notificação Webhook & Segurança HMAC-SHA256

Implementa o princípio de **Trust Boundary**: nenhum dado financeiro vindo do payload do webhook é considerado confiável até ser validado criptograficamente e checado diretamente na API oficial do Mercado Pago.

```mermaid
sequenceDiagram
    autonumber
    participant MP as Mercado Pago
    participant WH as Webhook Handler (API)
    participant Sec as Signature Verifier (HMAC-SHA256)
    participant MPAPI as Mercado Pago API (GET /v1/orders)
    participant DB as PostgreSQL

    MP->>WH: POST /webhook/mercadopago (X-Signature, X-Request-Id, ?data.id=ORD123)
    WH->>Sec: Valida Manifest (id + request-id + ts) com Webhook Secret
    alt Assinatura Inválida ou Expirada (> 5 min)
        Sec-->>WH: Erro de Validação
        WH-->>MP: 403 Forbidden
    else Assinatura Válida
        Sec-->>WH: OK
        WH->>DB: SELECT valor_esperado, moeda, status FROM payment_requests WHERE gateway_payment_id=ORD123
        WH->>MPAPI: GET /v1/orders/ORD123 (Autoridade da Verdade)
        MPAPI-->>WH: Estado Oficial (status='processed', total_amount, external_reference)
        WH->>WH: Valida external_reference, currency e amount
        WH->>DB: UPDATE payment_requests SET status='succeeded' WHERE status NOT IN ('succeeded', 'failed')
        WH-->>MP: 200 OK
    end
```

### 3. Fluxo de Estorno / Reembolso (Parcial/Total)

Reserva atômica no banco de dados (`SELECT ... FOR UPDATE`) com validação de saldo remanescente antes de acionar a adquirente, impedindo estornos duplicados e condições de corrida.

```mermaid
sequenceDiagram
    autonumber
    actor Consult as Sistema de Consultoria / Admin
    participant API as payment-request-api
    participant DB as PostgreSQL
    participant MP as Mercado Pago Orders API

    Consult->>API: POST /payment/refund (payment_id, amount_cents, idempotency_key, reason)
    API->>DB: SELECT FOR UPDATE payment_requests + SUM(refunds)
    alt Saldo Insuficiente ou Estado Inválido
        DB-->>API: Rejeição
        API-->>Consult: 400 Bad Request
    else Reserva Aprovada
        API->>DB: INSERT payment_refunds (status='processing')
        API->>MP: POST /v1/orders/:id/refund (X-Idempotency-Key, Transaction Amount)
        alt Gateway Falha
            MP-->>API: Erro (Saldo insuficiente, etc.)
            API->>DB: UPDATE payment_refunds SET status='failed'
            API-->>Consult: 500 / 400 Erro no Gateway
        else Gateway Sucesso
            MP-->>API: Refund ID Confirmado
            API->>DB: UPDATE payment_refunds (status='succeeded') + UPDATE payment_requests (status='partially_refunded' ou 'refunded')
            API-->>Consult: 200 OK (Refund Processado)
        end
    end
```

### 4. Fluxo OAuth do Vendedor & Armazenamento Cifrado (AES-256-GCM)

Para viabilizar Split 1:1 (Marketplace), os vendedores autorizam a aplicação via OAuth. Os tokens sensíveis nunca são trafegados no frontend nem expostos em logs.

```mermaid
sequenceDiagram
    autonumber
    actor Seller as Vendedor / Consultor
    participant API as payment-request-api
    participant MP as Mercado Pago OAuth
    participant Store as Encrypted File Store (AES-256-GCM)

    Seller->>API: GET /oauth/mercadopago/start
    API->>API: Gera State criptográfico com TTL de 10 minutos
    API-->>Seller: Redireciona para Mercado Pago Auth URL
    Seller->>MP: Autoriza acesso
    MP-->>API: Redireciona para /oauth/mercadopago/callback?code=...&state=...
    API->>API: Valida e consome State (Anti-Replay)
    API->>MP: POST /oauth/token (code + client_id + client_secret)
    MP-->>API: Tokens (access_token, refresh_token, user_id)
    API->>Store: Cifra payload com chave 256-bit (AES-GCM Nonce) e grava em disco (0600)
    API-->>Seller: 200 OK (seller_id conectado, tokens ocultados)
```

### 5. Worker de Reconciliação Periódica (Background)

Garante a consistência eventual e recuperação em caso de quedas de rede ou webhooks perdidos.

```mermaid
flowchart TD
    Start(["Tick do Reconciliador - cada 60s"]) --> QueryPending["Busca pagamentos pending com mais de 2 min"]
    QueryPending --> CheckPayments{"Existem pagamentos?"}
    CheckPayments -- "Sim" --> LoopPayments["Para cada pagamento"]
    CheckPayments -- "Não" --> QueryRefunds["Busca reembolsos com status processing"]
    
    LoopPayments --> CallMP["Consulta Mercado Pago GET /v1/orders"]
    CallMP --> UpdateDB["Atualiza estado local se terminal no gateway"]
    UpdateDB --> QueryRefunds
    
    QueryRefunds --> CheckRefunds{"Existem refunds processing?"}
    CheckRefunds -- "Sim" --> RecoverRefund["Consulta lista de refunds na adquirente"]
    RecoverRefund --> MatchRefund{"Refund aprovado no MP?"}
    MatchRefund -- "Sim" --> MarkSucceeded["Marca status succeeded no banco local"]
    MatchRefund -- "Não" --> NextRefund["Mantém processing / auditoria"]
    MarkSucceeded --> FinishNode(["Fim do Ciclo"])
    NextRefund --> FinishNode
    CheckRefunds -- "Não" --> FinishNode
```

---

## 🔒 Segurança & Conformidade Mercado Pago

- **Imutabilidade Financeira**: O valor bruto original (`amount_cents`) registrado no momento da compra é estritamente imutável. Reembolsos são operações acumulativas rastreadas em tabela própria (`payment_refunds`).
- **Validação de Assinatura Webhook**:
  - Algoritmo: HMAC-SHA256.
  - Manifest oficial: `id:[data.id];request-id:[x-request-id];ts:[ts];`.
  - Tolerância de tempo: máximo de **5 minutos** entre o timestamp do header e o relógio local para mitigar ataques de repetição (Replay Attacks).
- **Sem Exposição de Segredos**: Segredos de webhook, tokens de acesso OAuth, chaves privadas e CPF/CNPJ de clientes nunca são registrados em logs nem expostos em payloads de retorno.
- **Idempotência Rigorosa**:
  - Ingestão de pagamentos protegida por chaves de idempotência únicas.
  - Estornos utilizam chaves compostas para evitar cobranças/devoluções duplicadas.

---

## ⚙️ Variáveis de Ambiente

### `payment-request-api` (`.env`)
```env
PORT=8080
DATABASE_URL=postgres://postgres:postgres@localhost:5432/payment_db?sslmode=disable
RABBITMQ_URL=amqp://guest:guest@localhost:5672/

# Mercado Pago
MERCADO_PAGO_ACCESS_TOKEN=TEST-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
MERCADO_PAGO_WEBHOOK_SECRET=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# OAuth & Split Vendedor (Opcional para ambiente de split)
MERCADO_PAGO_OAUTH_CLIENT_ID=xxxxxxxxxxxx
MERCADO_PAGO_OAUTH_CLIENT_SECRET=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
MERCADO_PAGO_OAUTH_REDIRECT_URI=http://localhost:8080/oauth/mercadopago/callback
MERCADO_PAGO_OAUTH_TOKEN_FILE=./data/seller-tokens.enc
MERCADO_PAGO_OAUTH_ENCRYPTION_KEY=sua-chave-base64-de-32-bytes
```

### `payment-consumer` (`.env`)
```env
DATABASE_URL=postgres://postgres:postgres@localhost:5432/payment_db?sslmode=disable
RABBITMQ_URL=amqp://guest:guest@localhost:5672/

# Mercado Pago
MERCADO_PAGO_ACCESS_TOKEN=TEST-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
MERCADO_PAGO_WEBHOOK_URL=https://seu-dominio-publico.com/payment/webhook/mercadopago
PIX_EXPIRATION_TIME=30

# OAuth Split Storage
MERCADO_PAGO_OAUTH_TOKEN_FILE=./data/seller-tokens.enc
MERCADO_PAGO_OAUTH_ENCRYPTION_KEY=sua-chave-base64-de-32-bytes
```

---

## 🚀 Como Executar

### 1. Pré-requisitos
- Go 1.22+ ou Go 1.23+
- Docker & Docker Compose
- PostgreSQL e RabbitMQ

### 2. Subindo a infraestrutura com Docker Compose
```bash
docker compose up -d postgres rabbitmq
```

### 3. Aplicando migrações no banco
```bash
cd payment-request-api
go run cmd/main.go
```

### 4. Executando os serviços
Em terminais separados:

```bash
# Terminal 1: Iniciar API
cd payment-request-api
go run cmd/main.go

# Terminal 2: Iniciar Consumer Worker
cd payment-consumer
go run cmd/main.go
```

---

## 🧪 Execução de Testes

Todas as suítes de testes unitários contam com mocks isolados de repositórios, mensageria e adquirentes:

```bash
# Testes da API
cd payment-request-api
go test -v -race ./...

# Testes do Consumer
cd payment-consumer
go test -v -race ./...
```

Para testes integrados ponta a ponta com contas de teste do Mercado Pago, consulte o [📋 Guia de Testes Manuais](./MANUAL_TESTING.md).

    participant Consumer as payment-consumer
    participant MP as Mercado Pago Orders API

    User->>API: POST /payment (Idempotency-Key, Amount, Customer, Split)
    API->>DB: INSERT payment_requests (status='pending')
    API->>RMQ: Publish Event 'payment.requested'
    API-->>User: 201 Created (payment_uuid, status='pending')

    Consumer->>RMQ: Consume 'payment.requested'
    Consumer->>MP: POST /v1/orders (Pix + Items + Payer + Split Fee)
    MP-->>Consumer: Order Criada (ID, QR Code, Emv)
    Consumer->>DB: INSERT payment_attempts & UPDATE payment_requests
    
    User->>API: GET /payment/:payment_id/client-secret
    API->>DB: SELECT PixQrCode, PixExpirationAt
    API-->>User: 200 OK (QR Code Copia-e-Cola + Base64)
```
