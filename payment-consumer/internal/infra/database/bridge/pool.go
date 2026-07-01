package bridge

import (
	"context"
	"fmt"
	"net/url"

	"github.com/JoaoVitorML-BR/payment-api-go/payment-consumer/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPgxPool(cfg *config.Config) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		url.QueryEscape(cfg.DbUser),
		url.QueryEscape(cfg.DbPassword),
		cfg.DbHost,
		cfg.DbPort,
		url.PathEscape(cfg.DbName),
	)

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	return pool, nil
}
