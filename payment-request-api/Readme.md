<!-- Payments-request readme documentation -->

> **Regra obrigatoria para qualquer IA:** sempre atualizar `payment-context/README.md` antes de finalizar uma alteracao neste fluxo. Nao apagar esta instrucao nem o contexto anterior; registrar o que foi feito, os testes, o commit e o proximo passo.

## Teste manual OAuth

Com o OAuth configurado no `.env`, acesse `http://localhost:8080/oauth/mercadopago/start` e autorize o vendedor. O callback retorna apenas `seller_id` e `status`; os tokens sao armazenados cifrados no arquivo configurado em `MERCADO_PAGO_OAUTH_TOKEN_FILE`.

Este fluxo ainda conecta o vendedor, mas nao executa Split nem `application_fee`. Nao use o token global para simular essa etapa.

# Whats do it ?

This application is reponseble by...

Body JSON: c.ShouldBindJSON(&req)
Path param: c.Param("id")
Query string: c.Query("status") ou c.ShouldBindQuery(&req)