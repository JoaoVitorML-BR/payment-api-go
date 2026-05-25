payment-request-api/
  cmd/server/main.go
  =-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=
  internal/
    infra/
      database/
        migrations/ <- Migrations from database
          001_create_payment_request.sql
          002_payment_request_indexes.sql
          003_payment_request_updated_at_trigger.sql
        sqlc/
          queries.sql <- queries that gen inserts, gets, delete etc...
          schema.sql <- schema from database create
          sqlc.yaml <- config from sqlc 
    server/
      http.go
      router.go
    config/
      config.go
    payment/
      service.go <- useCase
      handler.go <- controller
      repository.go <- persists with databse
    messaging/
      publisher.go
  pkg/               
  Dockerfile
  go.mod
  go.sum