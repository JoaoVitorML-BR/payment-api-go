package bootstrap

import (
	"log"

	"github.com/JoaoVitorML-BR/payment-api-go/payment-consumer/internal/config"
	"github.com/JoaoVitorML-BR/payment-api-go/payment-consumer/internal/infra/database"
	consumerstripe "github.com/JoaoVitorML-BR/payment-api-go/payment-consumer/internal/infra/paymentStripe"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	Pool        *pgxpool.Pool
	StripeClient *consumerstripe.Client
}

func Bootstrap(cfg *config.Config) (*Dependencies, error) {
	log.Println("Bootstrapping payment consumer...")
	pool, err := database.NewPool(cfg)
	if err != nil {
		return nil, err
	}

	deps := &Dependencies{
		Pool:        pool,
		StripeClient: consumerstripe.NewClient(cfg.StripeAPIKey),
	}

	return deps, nil
}
