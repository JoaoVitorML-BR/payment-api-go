FROM golang:1.26-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/main ./cmd/main.go

FROM alpine:3.21
WORKDIR /app

# Create a non-root user and copy the built application
RUN adduser -D -H -u 10001 appuser
COPY --from=builder /app/main /app/main
COPY --from=builder /app/.env /app/.env

EXPOSE 8080
USER appuser
CMD ["/app/main"]