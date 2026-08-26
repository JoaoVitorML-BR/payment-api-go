FROM golang:1.26-alpine AS builder
WORKDIR /app

COPY payment-consumer/go.mod payment-consumer/go.sum ./
RUN go mod download

COPY payment-consumer/ .

# Sync schema.sql from payment-request-api (centralized source of truth)
# This ensures payment-consumer uses the same schema as payment-request-api
COPY payment-request-api/internal/infra/database/sqlc/schema.sql \
     ./internal/infra/database/sqlc/schema.sql

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/main ./cmd/main.go

FROM alpine:3.21
WORKDIR /app

# Create a non-root user and copy the built application
RUN adduser -D -H -u 10001 appuser
COPY --from=builder /app/main /app/main
# NOTE: Removed COPY for .env — it is handled via docker-compose only.

USER appuser
CMD ["/app/main"]