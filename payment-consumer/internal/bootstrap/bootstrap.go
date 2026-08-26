// payment-consumer\internal\bootstrap\bootstrap.go
package bootstrap

import (
	"log"

	"github.com/JoaoVitorML-BR/payment-api-go/payment-consumer/internal/config"
	"github.com/JoaoVitorML-BR/payment-api-go/payment-consumer/internal/infra/database/bridge"
	consumermercadopago "github.com/JoaoVitorML-BR/payment-api-go/payment-consumer/internal/infra/paymentMercadoPago"
	"github.com/JoaoVitorML-BR/payment-api-go/payment-consumer/internal/infra/paymentgateway"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	Pool    *pgxpool.Pool
	Gateway paymentgateway.Gateway
}

func Bootstrap(cfg *config.Config) (*Dependencies, error) {
	log.Println("Bootstrapping payment consumer...")
	pool, err := bridge.NewPgxPool(cfg)
	if err != nil {
		return nil, err
	}

	gateway, err := consumermercadopago.NewClient(cfg.MercadoPagoAccessToken)
	if err != nil {
		return nil, err
	}

	deps := &Dependencies{
		Pool:    pool,
		Gateway: gateway,
	}

	return deps, nil
}
