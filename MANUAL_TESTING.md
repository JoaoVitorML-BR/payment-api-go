# Manual Testing Guide

Roteiro manual para validar OAuth, Pix, Split, webhook, refunds e recuperação. Não coloque tokens, client secrets, CPF real ou dados sensíveis neste arquivo.

## 1. Pré-requisitos

- Docker Desktop em execução.
- Aplicação Mercado Pago de teste configurada para Marketplace/Split 1:1.
- Conta de vendedor de teste autorizada via OAuth.
- Credenciais de teste do Mercado Pago preenchidas nos `.env` locais.
- Redirect URI registrada exatamente como:

```text
http://localhost:8080/oauth/mercadopago/callback
```

Copie os `.env.example` para `.env` nos dois serviços. Gere uma chave OAuth com PowerShell e use o mesmo resultado nos dois serviços:

```powershell
[Convert]::ToBase64String([Security.Cryptography.RandomNumberGenerator]::GetBytes(32))
```

Não faça commit dos arquivos `.env`.

## 2. Subir infraestrutura

Na raiz do repositório:

```powershell
docker compose up -d postgres rabbitmq
pwsh -File .\payment-request-api\scripts\run-migrations.ps1
docker compose up -d --build payment-request-api payment-consumer
```

Verifique:

```powershell
Invoke-RestMethod http://localhost:8080/health
docker compose ps
docker compose logs --tail=100 payment-request-api payment-consumer
```

Resultado esperado: `/health` retorna `{"status":"ok"}` e os dois containers ficam ativos.

## 3. Testar OAuth do vendedor

1. Abra `http://localhost:8080/oauth/mercadopago/start`.
2. Autorize a conta de vendedor de teste.
3. Confirme que o callback retorna somente `seller_id` e `status`.
4. Confirme que tokens não aparecem na resposta nem nos logs.
5. Confirme o arquivo cifrado no volume compartilhado:

```powershell
docker compose exec payment-request-api ls -l /app/data
docker compose exec payment-consumer ls -l /app/data
```

Resultado esperado: ambos enxergam o mesmo arquivo e o conteúdo não é JSON legível.

## 4. Criar Pix sem Split

Use um `idempotency_key` novo:

```powershell
$body = @{
  idempotency_key = "pix-manual-001"
  amount_cents = 10000
  currency = "BRL"
  payment_method = "pix"
  customer = @{
    name = "Cliente Teste"
    email = "cliente@example.com"
    tax_id = "52998224725"
    address = "Rua Teste"
    city = "Sao Paulo"
    state = "SP"
    postal_code = "01000000"
  }
} | ConvertTo-Json -Depth 5

$created = Invoke-RestMethod -Method Post -Uri http://localhost:8080/payment -ContentType "application/json" -Body $body
$created | ConvertTo-Json -Depth 10
$paymentId = $created.data.payment_uuid
Invoke-RestMethod "http://localhost:8080/payment/client-secret/$paymentId" | ConvertTo-Json -Depth 10
```

Resultado esperado: status `pending`, QR Code Pix e `gateway_payment_id`. Pague o QR Code com a conta de teste e consulte novamente até `succeeded`. A criação `201` não significa pagamento aprovado.

## 5. Criar Pix com Split

Use o `seller_id` retornado pelo OAuth:

```powershell
$body = @{
  idempotency_key = "split-manual-001"
  amount_cents = 10000
  currency = "BRL"
  payment_method = "pix"
  seller_id = "SELLER_ID_DE_TESTE"
  marketplace_fee_cents = 2000
  customer = @{
    name = "Cliente Teste"
    email = "cliente@example.com"
    tax_id = "52998224725"
    address = "Rua Teste"
    city = "Sao Paulo"
    state = "SP"
    postal_code = "01000000"
  }
} | ConvertTo-Json -Depth 5

Invoke-RestMethod -Method Post -Uri http://localhost:8080/payment -ContentType "application/json" -Body $body | ConvertTo-Json -Depth 10
```

Resultado esperado: o consumer usa o token OAuth do vendedor, envia `application_fee` de R$ 20,00 e a cobrança aparece na conta do vendedor. Enviar `marketplace_fee_cents` sem `seller_id` deve retornar `400` sem criar cobrança.

## 6. Idempotência de criação

Envie novamente o mesmo payload do caso anterior com a mesma `idempotency_key`.

Resultado esperado: o mesmo `payment_uuid` é retornado, nenhum segundo evento é publicado e nenhuma segunda cobrança é criada.

## 7. Refund parcial

Depois de o pagamento estar `succeeded`, execute 75% de R$ 100,00:

```powershell
$refund = @{
  payment_id = $paymentId
  amount_cents = 7500
  idempotency_key = "refund-manual-001"
  reason = "customer_cancelled"
} | ConvertTo-Json

Invoke-RestMethod -Method Post -Uri http://localhost:8080/payment/refund -ContentType "application/json" -Body $refund | ConvertTo-Json
```

Resultado esperado: refund aprovado, registro local `succeeded` e pagamento `partially_refunded`.

## 8. Idempotência de refund

Envie novamente exatamente o mesmo payload do refund parcial. A API não deve criar outro refund no Mercado Pago. Confirme no painel/conta de teste que existe apenas uma operação de R$ 75,00.

## 9. Refund total complementar

Complete o pagamento com um segundo refund de R$ 25,00 e nova chave:

```powershell
$refund = @{
  payment_id = $paymentId
  amount_cents = 2500
  idempotency_key = "refund-manual-002"
  reason = "consultant_cancelled"
} | ConvertTo-Json

Invoke-RestMethod -Method Post -Uri http://localhost:8080/payment/refund -ContentType "application/json" -Body $refund | ConvertTo-Json
```

Resultado esperado: refund aprovado, estado local `refunded` e soma dos refunds igual ao valor original. O valor bruto original continua imutável.

## 10. Casos negativos

Cada caso deve falhar sem chamar o gateway:

- pagamento ainda `pending`;
- `amount_cents` zero ou negativo;
- ausência de `idempotency_key`;
- valor acumulado acima do valor original;
- `split_rule: "50/50"`;
- seller sem token OAuth correspondente.

## 11. Recuperação de refund processing

Em ambiente controlado, interrompa o processo depois de o Mercado Pago aprovar um refund e antes da persistência local. Reinicie a API:

```powershell
docker compose restart payment-request-api
docker compose logs -f payment-request-api
```

Resultado esperado: o ciclo de reconciliação consulta os refunds do Mercado Pago e marca a operação local como `succeeded`, sem criar outro refund.

## 12. Checklist final

- [ ] Migrations aplicadas.
- [ ] `/health` respondendo.
- [ ] OAuth retornou apenas `seller_id` e `status`.
- [ ] Arquivo de token existe nos dois containers e está cifrado.
- [ ] Pix sem Split aprovado.
- [ ] Pix com Split aprovado e comissão conferida.
- [ ] Criação idempotente verificada.
- [ ] Refund parcial aprovado.
- [ ] Refund repetido não duplicou operação.
- [ ] Refund complementar levou o pagamento a `refunded`.
- [ ] Casos inválidos recusados.
- [ ] Recuperação de operação `processing` verificada.

## Limites conhecidos

Este roteiro usa contas de teste. Tarifa, saldo disponível, prazo de refund e comportamento final do Split dependem do ambiente Mercado Pago. A API de pagamentos não calcula a política de cancelamento da consultoria; recebe o valor autorizado e executa a operação financeira.